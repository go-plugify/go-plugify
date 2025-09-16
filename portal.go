package goportal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"plugin"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/gin-gonic/gin"
)

var logger Logger

type Logger interface {
	WarnCtx(ctx context.Context, format string, args ...any)
	ErrorCtx(ctx context.Context, format string, args ...any)
	InfoCtx(ctx context.Context, format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
	Info(format string, args ...any)
}

type PluginDepencies struct {
	// business services
	Service any

	// plugin services
	Store   *IPluginDataStore
	Plugins *IPluginManager

	// utils
	Logger *Logger
	Util   *Util

	// middlewares
	GinEngine *GinEngine
}

func (p *PluginDepencies) GetLogger() any {
	return p.Logger
}

func (p *PluginDepencies) GetGinEngine() any {
	return p.GinEngine
}

func (p *PluginDepencies) GetStore() any {
	return p.Store
}

func (p *PluginDepencies) GetPlugins() any {
	return p.Plugins
}

func (p *PluginDepencies) GetUtil() any {
	return p.Util
}

func (p *PluginDepencies) GetService() any {
	return p.Service
}

type GinEngine struct {
	Engine *gin.Engine
}

func (p *GinEngine) ReplaceHandler(method, path string, handler func(ctx context.Context)) error {
	return ReplaceLastHandler(p.Engine, method, path, func(c *gin.Context) {
		handler(c)
	})
}

func (p *GinEngine) GetHandler(method, path string) (func(ctx context.Context), error) {
	handlers, err := getHandlerSlicePointer(p.Engine, method, path)
	if err != nil {
		return nil, err
	}
	handler := (*handlers)[len(*handlers)-1]
	return func(ctx context.Context) {
		handler(ctx.(*gin.Context))
	}, nil
}

func (p *GinEngine) GetHandlerName(method, path string) (string, error) {
	handlers, err := getHandlerSlicePointer(p.Engine, method, path)
	if err != nil {
		return "", err
	}
	if handlers == nil || len(*handlers) == 0 {
		return "", fmt.Errorf("no handlers found for method: %s, route: %s", method, path)
	}
	handler := (*handlers)[len(*handlers)-1]
	handlerName := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
	return handlerName, nil
}

func (p *GinEngine) GetRoutes() []gin.RouteInfo {
	return p.Engine.Routes()
}

func getHandlerSlicePointer(engine *gin.Engine, method string, route string) (*[]gin.HandlerFunc, error) {
	engineVal := reflect.ValueOf(engine).Elem()
	trees := engineVal.FieldByName("trees")
	if !trees.IsValid() {
		return nil, fmt.Errorf("cannot find route trees")
	}

	for i := range trees.Len() {
		tree := trees.Index(i)
		methodField := getUnexportedField(tree.FieldByName("method")).String()
		if !strings.EqualFold(methodField, method) {
			continue
		}

		root := getUnexportedField(tree.FieldByName("root"))
		handlers := findHandlersInNode(root, route, "")
		if handlers == nil {
			continue
		} else {
			return handlers, nil
		}
	}
	return nil, fmt.Errorf("handler not found for method: %s, route: %s", method, route)
}

func findHandlersInNode(node reflect.Value, target, currentPath string) *[]gin.HandlerFunc {
	if !node.IsValid() {
		return nil
	}

	if node.Kind() == reflect.Ptr {
		if node.IsNil() {
			return nil
		}
		node = node.Elem()
	}
	node = getUnexportedField(node)

	path := getUnexportedField(node.FieldByName("path")).String()
	fullPath := currentPath + path

	if fullPath == target {
		handlersField := getUnexportedField(node.FieldByName("handlers"))
		handlersPtr := (*[]gin.HandlerFunc)(unsafe.Pointer(handlersField.UnsafeAddr()))
		return handlersPtr
	}

	childrenField := getUnexportedField(node.FieldByName("children"))
	for i := range childrenField.Len() {
		child := childrenField.Index(i)
		if child.Kind() == reflect.Ptr && child.IsNil() {
			continue
		}
		if res := findHandlersInNode(child, target, fullPath); res != nil {
			return res
		}
	}
	return nil
}

func ReplaceLastHandler(engine *gin.Engine, method, route string, newHandler gin.HandlerFunc) error {
	handlersPtr, err := getHandlerSlicePointer(engine, method, route)
	if err != nil {
		return fmt.Errorf("failed to get handler slice pointer: %w", err)
	}
	if handlersPtr == nil || len(*handlersPtr) == 0 {
		return fmt.Errorf("no handlers found for method: %s, route: %s", method, route)
	}
	(*handlersPtr)[len(*handlersPtr)-1] = newHandler
	return nil
}

const (
	RedisKeyPluginPodIP     = "plugin:%s:podip"
	RedisKeyPluginInstalled = "plugin:%s:installed"
)

type Plugin struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	InstallTime time.Time                 `json:"install_time"`
	UpgradeTime time.Time                 `json:"upgrade_time"`
	Initialize  func(ctx context.Context) `json:"-"`
	Load        func(any)                 `json:"-"`

	lock sync.RWMutex `json:"-"`
}

func (p *Plugin) Upgrade(inputPlugin PluginInput) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.Description = inputPlugin.GetDescription()
	p.Initialize = inputPlugin.Initialize
	p.Load = inputPlugin.Load
	p.UpgradeTime = time.Now()
}

type PluginInput interface {
	GetName() string
	GetDescription() string
	Initialize(ctx context.Context)
	Load(dependencies any)
}

func (manager *IPluginManager) LoadPlugin(pluginso []byte) (*Plugin, error) {

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

	existPlugin, ok := manager.plugins.Get(exports.GetName())
	if ok {
		logger.Warn("[Plugin] Plugin %s already exists, upgrading...", exports.GetName())
		existPlugin.Upgrade(exports)
		existPlugin.LoadDepencies(manager.dependencies)
		return existPlugin, nil
	}

	plugin := &Plugin{
		Name:        exports.GetName(),
		Description: exports.GetDescription(),
		InstallTime: time.Now(),
		Initialize:  exports.Initialize,
		Load:        exports.Load,
	}
	plugin.LoadDepencies(manager.dependencies)

	manager.plugins.Add(plugin)

	return plugin, nil
}

func (p *Plugin) LoadDepencies(plugDepencies *PluginDepencies) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.Load != nil {
		p.Load(plugDepencies)
	} else {
		logger.Warn("[Plugin] No load method found in plugin %s", p.Name)
	}
}

func (p *Plugin) Init(c context.Context) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.Initialize != nil {
		p.Initialize(c)
	} else {
		logger.WarnCtx(c, "[Plugin] No initialize method found in plugin %s", p.Name)
	}
}

type IPluginManager struct {
	plugins      *Plugins
	dependencies *PluginDepencies

	serviceName string
}

func (manager *IPluginManager) GetPlugins() *Plugins {
	return manager.plugins
}

func (manager *IPluginManager) List() any {
	return manager.plugins.List()
}

type Plugins struct {
	plugins map[string]*Plugin
	mu      sync.RWMutex
}

func (p *Plugins) Add(plugin *Plugin) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.plugins[plugin.Name] = plugin
}

func (p *Plugins) Get(name string) (*Plugin, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	plugin, ok := p.plugins[name]
	return plugin, ok
}

func (p *Plugins) Remove(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.plugins, name)
}

func (p *Plugins) List() []*Plugin {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var plugins []*Plugin
	for _, plugin := range p.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}
