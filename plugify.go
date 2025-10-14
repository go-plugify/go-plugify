package goplugify

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"plugin"
	"strings"
	"sync"
	"time"
)

type PluginComponents struct {
	Logger Logger
	Util   *Util

	Components map[string]Component
}

func (p *PluginComponents) GetLogger() any {
	return p.Logger
}

func (p *PluginComponents) GetUtil() any {
	return p.Util
}

type Plugin struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	InstallTime time.Time `json:"install_time"`
	UpgradeTime time.Time `json:"upgrade_time"`

	run     func(any)         `json:"-"`
	load    func(any)         `json:"-"`
	methods map[string]func(any) `json:"-"`

	lock sync.RWMutex `json:"-"`
}

func (p *Plugin) Upgrade(inputPlugin PluginInput) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.Description = inputPlugin.GetDescription()
	p.run = inputPlugin.Run
	p.load = inputPlugin.Load
	p.UpgradeTime = time.Now()
	p.methods = inputPlugin.Methods()
}

type PluginInput interface {
	GetName() string
	GetDescription() string
	Run(any)
	Load(any)
	Methods() map[string]func(any)
}

func (manager *PluginManager) LoadPlugin(pluginso []byte) (*Plugin, error) {

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
		existPlugin.Load(manager.components)
		return existPlugin, nil
	}

	plugin := &Plugin{
		Name:        exports.GetName(),
		Description: exports.GetDescription(),
		InstallTime: time.Now(),
		run:         exports.Run,
		load:        exports.Load,
		methods:     exports.Methods(),
	}
	plugin.Load(manager.components)

	manager.plugins.Add(plugin)

	return plugin, nil
}

func (p *Plugin) Load(plugDepencies *PluginComponents) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.load != nil {
		p.load(plugDepencies)
	} else {
		logger.Warn("[Plugin] No load method found in plugin %s", p.Name)
	}
}

func (p *Plugin) RunHTTP(c HttpContext) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.run != nil {
		p.run(c)
	} else {
		logger.Warn("[Plugin] No run method found in plugin %s", p.Name)
	}
}

type PluginManager struct {
	plugins    *Plugins
	components *PluginComponents

	serviceName string
}

func (manager *PluginManager) GetPlugins() *Plugins {
	return manager.plugins
}

func (manager *PluginManager) List() any {
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

type HttpRouter interface {
	Add(route string, handler func(c HttpContext))
}

func Init(serviceName string, components ...Component) {
	var logger Logger
	extendCompones := make(map[string]Component)
	for _, c := range components {
		extendCompones[c.Name()] = c
		if _, isLogger := c.Service().(Logger); isLogger {
			logger = c.Service().(Logger)
		}
	}
	globalPluginManager = make(map[string]*PluginManager)
	if serviceName == "" {
		serviceName = "default"
	}
	if logger == nil {
		logger = &DefaultLogger{}
	}
	globalPluginManager[serviceName] = &PluginManager{
		plugins: &Plugins{
			plugins: make(map[string]*Plugin),
		},
		components: &PluginComponents{
			Logger:     logger,
			Util:       &Util{},
			Components: make(map[string]Component),
		},
	}
}

func (managers PluginManagers) RegisterRoutes(router HttpRouter, routePrefix string) {
	router.Add(routePrefix+"/plugin/init", managers.LoadPluginFromHTTP)
}

type HttpContext interface {
	GetHeader(key string) string
	Body() io.ReadCloser
	FormFile(name string) (*multipart.FileHeader, error)
	Query(key string) string
	JSON(code int, obj any)
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

func (managers PluginManagers) LoadPluginFromHTTP(c HttpContext) {
	fileContent, err := getPluginContent(c)
	if err != nil {
		ErrorRet(c, fmt.Errorf("get plugin content error: %v", err))
		return
	}

	serviceName := c.Query("service")
	if serviceName == "" {
		serviceName = "default"
	}

	plugin, err := globalPluginManager[serviceName].LoadPlugin(fileContent)
	if err != nil {
		ErrorRet(c, fmt.Errorf("load plugin error: %v", err))
		return
	}
	plugin.RunHTTP(c)
}

func ErrorRet(c HttpContext, err error) {
	c.JSON(500, map[string]interface{}{
		"error": err.Error(),
	})
}

type PluginManagers map[string]*PluginManager

var globalPluginManager = make(PluginManagers)
