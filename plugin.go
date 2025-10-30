package goplugify

import (
	"sync"
	"time"
)

type IPlugin interface {
	Meta() Meta
	OnInit(*PluginComponents) error
	OnRun(any) (any, error)
	OnDestroy(any) error
}

type Meta struct {
	ID          string
	Name        string
	Description string
	Author      string
	Version     string
	Type        string
	Entry       string
	Permissions []string
	HasUI       bool
	Loader      string
}

type Plugin struct {
	meta Meta

	InstallTime time.Time `json:"install_time"`
	UpgradeTime time.Time `json:"upgrade_time"`

	run     func(any) (any, error)   `json:"-"`
	load    func(any) error          `json:"-"`
	methods map[string]func(any) any `json:"-"`

	lock sync.RWMutex `json:"-"`
}

type PluginInput interface {
	GetName() string
	GetDescription() string
	Run(any) (any, error)
	Load(any) error
	Methods() map[string]func(any) any
}

func (p *Plugin) Meta() Meta {
	return p.meta
}

func (p *Plugin) OnInit(plugDepencies *PluginComponents) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.load == nil {
		return ErrPluginNoLoadMethod
	}

	return p.load(plugDepencies)
}

var ErrPluginNoLoadMethod = &PluginError{Message: "plugin has no load method"}

type PluginError struct {
	Message string
}

func (e *PluginError) Error() string {
	return e.Message
}

func (p *Plugin) OnDestroy(any) error {
	return nil
}

func (p *Plugin) Upgrade(inputPlugin PluginInput) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.meta.Description = inputPlugin.GetDescription()
	p.run = inputPlugin.Run
	p.load = inputPlugin.Load
	p.UpgradeTime = time.Now()
	p.methods = inputPlugin.Methods()
}

func (p *Plugin) OnRun(req any) (any, error) {
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
