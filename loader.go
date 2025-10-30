package goplugify

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"plugin"
	"reflect"
	"strings"
	"time"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

type Loader interface {
	Load(meta Meta, src any) (IPlugin, error)
	Name() string
}

type NativePluginHTTPLoader struct{}

func (l *NativePluginHTTPLoader) Name() string {
	return "native_plugin_http"
}

func (l *NativePluginHTTPLoader) Load(meta Meta, src any) (IPlugin, error) {
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

	p, err := plugin.Open(tmpfile.Name())
	if err != nil {
		return nil, err
	}
	sym, err := p.Lookup("ExportPlugin")
	if err != nil {
		return nil, err
	}
	exports := sym.(PluginInput)

	plugin := &Plugin{
		meta:        meta,
		InstallTime: time.Now(),
		run:         exports.Run,
		load:        exports.Load,
		methods:     exports.Methods(),
	}

	return plugin, nil
}

var (
	ErrInvalidLoaderSource = NewError("invalid loader source")
)

func NewError(message string) error {
	return &LoadError{message: message}
}

type LoadError struct {
	message string
}

func (e *LoadError) Error() string {
	return e.message
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

func (l *YaegiHTTPLoader) Load(meta Meta, src any) (IPlugin, error) {
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
			meta:        meta,
			InstallTime: time.Now(),
			methods:     map[string]func(any) any{},
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

func (p *YaegiPlugin) OnRun(req any) (any, error) {
	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols)
	i.Use(p.symbols)

	_, err := i.Eval(string(p.scriptContent))
	if err != nil {
		return nil, err
	}

	val, err := i.Eval("Run")
	if err != nil {
		return nil, err
	}

	fn := val.Interface().(func() map[string]any)
	return fn(), nil
}

func (p *YaegiPlugin) OnInit(plugDepencies *PluginComponents) error {

	defPkgPath := "plugify/plugify"

	p.symbols[defPkgPath] = make(map[string]reflect.Value)
	p.symbols[defPkgPath]["Util"] = reflect.ValueOf(plugDepencies.Util)
	p.symbols[defPkgPath]["Logger"] = reflect.ValueOf(plugDepencies.Logger)

	for _, comp := range plugDepencies.Components {
		p.symbols[defPkgPath][strings.ToTitle(comp.Name())] = reflect.ValueOf(comp.Service())
	}

	return nil
}
