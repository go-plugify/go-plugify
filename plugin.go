package goplugify

import (
	"sync"
	"time"
)

type IPlugin interface {
	OnInit(*PluginComponents) error
	OnRun(any) (any, error)
	OnDestroy(any) error
	Meta() *Meta
	Upgrade(IPlugin)
	Method(string) (func(any) any, bool)
}

type PluginFunc interface {
	Run(any) (any, error)
	Load(any) error
	Methods() map[string]func(any) any
	Destroy(any) error
}

type Meta struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Version     string `json:"version"`
	Loader      string `json:"loader"`
}

type Plugin struct {
	MetaInfo *Meta `json:"meta"`

	InstallTime time.Time `json:"install_time"`
	UpgradeTime time.Time `json:"upgrade_time"`

	run     func(any) (any, error)   `json:"-"`
	load    func(any) error          `json:"-"`
	methods map[string]func(any) any `json:"-"`
	destroy func(any) error          `json:"-"`

	lock sync.RWMutex `json:"-"`
}

func (p *Plugin) Meta() *Meta {
	return p.MetaInfo
}

func (p *Plugin) Method(name string) (func(any) any, bool) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	method, ok := p.methods[name]
	return method, ok
}

func (p *Plugin) Upgrade(newPlug IPlugin) {

	newPlugin, ok := newPlug.(*Plugin)
	if !ok {
		return
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	p.run = newPlugin.run
	p.load = newPlugin.load
	p.methods = newPlugin.methods
	p.destroy = newPlugin.destroy

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
