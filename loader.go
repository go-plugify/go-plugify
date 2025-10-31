package goplugify

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"plugin"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

type Loader interface {
	Load(meta *Meta, src any) (IPlugin, error)
	Name() string
}

type NativePluginHTTPLoader struct{}

func (l *NativePluginHTTPLoader) Name() string {
	return "native_plugin_http"
}

func (l *NativePluginHTTPLoader) Load(meta *Meta, src any) (IPlugin, error) {
	httpContext, ok := src.(HttpContext)
	if !ok {
		return nil, ErrInvalidLoaderSource
	}

	pluginso, err := getPluginContent(httpContext)
	if err != nil {
		return nil, err
	}

	serviceName := httpContext.Query("service")
	if serviceName == "" {
		serviceName = "default"
	}

	tmpfile, err := os.CreateTemp("", fmt.Sprintf("plugin_%d_*.so", time.Now().UnixNano()))
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpfile.Name())

	if _, err := io.Copy(tmpfile, bytes.NewBuffer(pluginso)); err != nil {
		return nil, err
	}

	openPlugin, err := plugin.Open(tmpfile.Name())
	if err != nil {
		return nil, err
	}
	sym, err := openPlugin.Lookup("ExportPlugin")
	if err != nil {
		return nil, err
	}
	exports := sym.(PluginFunc)

	plugin := &Plugin{
		MetaInfo:    meta,
		run:         exports.Run,
		load:        exports.Load,
		methods:     exports.Methods(),
		destroy:     exports.Destroy,
		InstallTime: time.Now(),
	}

	return plugin, nil
}

func getPluginContent(c HttpContext) ([]byte, error) {
	var fileContent []byte

	ct := c.GetHeader("Content-Type")
	if !strings.Contains(ct, "multipart/form-data") {
		fileContent, _ = io.ReadAll(c.Body())
		return fileContent, nil
	}

	file, err := c.FormFile("file")
	if err != nil {
		body, _ := io.ReadAll(c.Body())
		return nil, fmt.Errorf("file error: %v raw ct: %s, body length: %d", err, ct, len(body))
	}
	f, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fileContent, err = io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read file error: %v", err)
	}

	return fileContent, nil
}

type YaegiHTTPLoader struct{}

func (l *YaegiHTTPLoader) Name() string {
	return "yaegi_http"
}

func (l *YaegiHTTPLoader) Load(meta *Meta, src any) (IPlugin, error) {
	httpContext, ok := src.(HttpContext)
	if !ok {
		return nil, ErrInvalidLoaderSource
	}

	scriptContent, err := getPluginContent(httpContext)
	if err != nil {
		return nil, err
	}

	plugin := &YaegiPlugin{
		Plugin: &Plugin{
			MetaInfo:    meta,
			methods:     map[string]func(any) any{},
			InstallTime: time.Now(),
		},
		scriptContent: scriptContent,
		symbols:       make(map[string]map[string]reflect.Value),
	}

	return plugin, nil
}

type YaegiPlugin struct {
	*Plugin

	symbols       map[string]map[string]reflect.Value
	scriptContent []byte
}

func toTitle(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (p *YaegiPlugin) OnInit(plugDepencies *PluginComponents) error {

	defPkgPath := "plugify/plugify"

	p.symbols[defPkgPath] = make(map[string]reflect.Value)
	p.symbols[defPkgPath]["Util"] = reflect.ValueOf(plugDepencies.Util)
	p.symbols[defPkgPath]["Logger"] = reflect.ValueOf(NewLoggerWrapper(plugDepencies.Logger))

	for _, comp := range plugDepencies.Components {
		plugDepencies.Logger.Info("Injecting component into plugin %s, component %s", p.Meta().ID, toTitle(comp.Name()))
		p.symbols[defPkgPath][toTitle(comp.Name())] = reflect.ValueOf(comp.Service())
		if len(p.Meta().Components) > 0 {
			fields := MakeStructTypeMap(comp.Service(), p.Meta().Components)
			for k, v := range fields {
				plugDepencies.Logger.Info("Injecting component into plugin %s, component %s", p.Meta().ID, toTitle(k))
				p.symbols[defPkgPath][toTitle(k)] = v
			}
		}
	}

	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols)
	i.Use(p.symbols)

	_, err := i.Eval(string(p.scriptContent))
	if err != nil {
		return err
	}

	runFn, err := i.Eval("Run")
	if err != nil {
		return err
	}
	p.run = func(a any) (any, error) {
		return runFn.Interface().(func(map[string]any) (any, error))(map[string]any{"input": a})
	}

	methodsFn, err := i.Eval("Methods")
	if err != nil {
		return err
	}
	p.methods = methodsFn.Interface().(func() map[string]func(any) any)()

	destroyFn, err := i.Eval("Destroy")
	if err != nil {
		return err
	}
	p.destroy = func(a any) error {
		return destroyFn.Interface().(func(map[string]any) error)(map[string]any{"input": a})
	}

	return nil
}

// MakeStructTypeMap Scans the struct and its method parameters, collecting only custom struct types (non-primitive types, non-standard library).
// Key format: <PkgNameCapitalized><TypeName>[#hash], with hash added only when the package paths differ but the keys are the same.
func MakeStructTypeMap(sample any, needComps PluginComponentItems) map[string]reflect.Value {
	result := make(map[string]reflect.Value)
	keyPkgMap := make(map[string][]string)

	t := reflect.TypeOf(sample)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Scan fields recursively
	buildFieldTypeMap(t, result, keyPkgMap, needComps)

	// Scan method parameters
	buildMethodTypeMap(reflect.TypeOf(sample), result, keyPkgMap, needComps)

	return result
}

func buildFieldTypeMap(t reflect.Type, result map[string]reflect.Value, keyPkgMap map[string][]string, needComps PluginComponentItems) {
	if t.Kind() != reflect.Struct {
		return
	}
	for i := range t.NumField() {
		f := t.Field(i)
		ft := f.Type
		addTypeToMap(ft, result, keyPkgMap, needComps)

		if ft.Kind() == reflect.Struct {
			buildFieldTypeMap(ft, result, keyPkgMap, needComps)
		}
	}
}

func buildMethodTypeMap(t reflect.Type, result map[string]reflect.Value, keyPkgMap map[string][]string, needComps PluginComponentItems) {
	for i := range t.NumMethod() {
		m := t.Method(i)
		mt := m.Type
		for j := 1; j < mt.NumIn(); j++ {
			argType := mt.In(j)
			addTypeToMap(argType, result, keyPkgMap, needComps)
		}
	}
}

// addTypeToMap adds the type to the result map if it's a custom struct type.
func addTypeToMap(t reflect.Type, result map[string]reflect.Value, keyPkgMap map[string][]string, needComps PluginComponentItems) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}

	pkg := t.PkgPath()
	if pkg == "" || isStdLibType(pkg) {
		return
	}

	typeName := t.Name()
	if typeName == "" {
		return
	}

	if !needComps.Has(pkg, typeName) {
		return
	}

	pkgBase := path.Base(pkg)
	if pkgBase == "" {
		pkgBase = "Main"
	}

	key := toTitle(pkgBase) + typeName

	// If the key already exists but the package paths are different => add a hash suffix to distinguish
	if prevPkg, ok := keyPkgMap[key]; ok && len(prevPkg) > 0 {
		if !slices.Contains(prevPkg, pkg) {
			key = fmt.Sprintf("%s%d", key, len(prevPkg))
		} else {
			return
		}
	}

	result[key] = reflect.ValueOf(reflect.Zero(reflect.PointerTo(t)).Interface())
	keyPkgMap[key] = append(keyPkgMap[key], pkg)
}

// isStdLibType checks if the package is from the standard library.
func isStdLibType(pkg string) bool {
	if slices.Contains([]string{"time", "fmt", "io", "os", "net", "strings", "bytes", "reflect", "sync"}, pkg) {
		return true
	}
	return false
}
