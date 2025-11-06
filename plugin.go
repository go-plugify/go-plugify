package goplugify

import (
	"os"
	"sync"
	"time"
)

type IPlugin interface {
	OnInit(*PluginComponents) error
	OnRun(any) (any, error)
	OnDestroy(any) error
	Meta() *Meta
	Upgrade(PluginFunc)
	Method(string) (func(any) any, bool)
	ExportFunc() PluginFunc
}

type PluginFunc interface {
	Run(any) (any, error)
	Load(any) error
	Methods() map[string]func(any) any
	Destroy(any) error
}

type Meta struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Author      string               `json:"author"`
	Version     string               `json:"version"`
	Loader      LoaderType           `json:"loader"`
	Components  PluginComponentItems `json:"components"`
}

type PluginComponentItems []*PluginComponentItem

func (p PluginComponentItems) Has(pkgPath, name string) bool {
	for _, item := range p {
		if item.PkgPath == pkgPath && item.Name == name {
			return true
		}
	}
	return false
}

type PluginComponentItem struct {
	PkgPath string `json:"pkg_path"`
	Name    string `json:"name"`
}

type Plugin struct {
	MetaInfo *Meta `json:"meta"`

	InstallTime time.Time `json:"install_time"`
	UpgradeTime time.Time `json:"upgrade_time"`
	RunTime     time.Time `json:"latest_run_time"`
	RunTimes    int       `json:"run_times"`
	Host        string    `json:"run_host"`

	run     func(any) (any, error)   `json:"-"`
	load    func(any) error          `json:"-"`
	methods map[string]func(any) any `json:"-"`
	destroy func(any) error          `json:"-"`

	lock sync.RWMutex `json:"-"`
}

func (p *Plugin) Meta() *Meta {
	return p.MetaInfo
}

func (p *Plugin) ExportFunc() PluginFunc {
	return &exportedPluginFunc{
		run:     p.run,
		load:    p.load,
		methods: p.methods,
		destroy: p.destroy,
	}
}

type exportedPluginFunc struct {
	run     func(any) (any, error)
	load    func(any) error
	methods map[string]func(any) any
	destroy func(any) error
}

func (e *exportedPluginFunc) Run(req any) (any, error) {
	return e.run(req)
}

func (e *exportedPluginFunc) Load(src any) error {
	return e.load(src)
}

func (e *exportedPluginFunc) Methods() map[string]func(any) any {
	return e.methods
}

func (e *exportedPluginFunc) Destroy(req any) error {
	return e.destroy(req)
}

func (p *Plugin) Method(name string) (func(any) any, bool) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	method, ok := p.methods[name]
	return method, ok
}

func (p *Plugin) Upgrade(newPlugin PluginFunc) {

	p.lock.Lock()
	defer p.lock.Unlock()

	p.run = newPlugin.Run
	p.load = newPlugin.Load
	p.methods = newPlugin.Methods()
	p.destroy = newPlugin.Destroy

	p.UpgradeTime = time.Now()
}

func (p *Plugin) OnInit(plugDepencies *PluginComponents) error {
	if p.load == nil {
		return ErrPluginNoLoadMethod
	}
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.load(plugDepencies)
}

func (p *Plugin) OnDestroy(req any) error {
	if p.destroy == nil {
		return nil
	}
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.destroy(req)
}

func (p *Plugin) OnRun(req any) (any, error) {
	if p.run == nil {
		return nil, ErrPluginNoRunMethod
	}
	p.lock.Lock()
	defer p.lock.Unlock()
	p.RunTime = time.Now()
	p.RunTimes++
	p.Host, _ = os.Hostname()
	return p.run(req)
}

type Plugins struct {
	plugins map[string]IPlugin
	mu      sync.RWMutex
}

func (p *Plugins) Add(plugin IPlugin) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.plugins[plugin.Meta().ID] = plugin
}

func (p *Plugins) Get(name string) (IPlugin, bool) {
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

func (p *Plugins) List() []IPlugin {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var plugins []IPlugin
	for _, plugin := range p.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}
