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
	SetMeta(meta *Meta)

	GetInstallTime() time.Time
	GetUpgradeTime() time.Time
	SetInstallTime(time.Time)
	SetUpgradeTime(time.Time)
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
	meta *Meta

	InstallTime time.Time `json:"install_time"`
	UpgradeTime time.Time `json:"upgrade_time"`

	run     func(any) (any, error)   `json:"-"`
	load    func(any) error          `json:"-"`
	methods map[string]func(any) any `json:"-"`

	lock sync.RWMutex `json:"-"`
}

func (p *Plugin) SetMeta(meta *Meta) {
	p.meta = meta
}

func (p *Plugin) GetInstallTime() time.Time {
	return p.InstallTime
}

func (p *Plugin) GetUpgradeTime() time.Time {
	return p.UpgradeTime
}

func (p *Plugin) SetInstallTime(t time.Time) {
	p.InstallTime = t
}

func (p *Plugin) SetUpgradeTime(t time.Time) {
	p.UpgradeTime = t
}

func (p *Plugin) Meta() *Meta {
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

func (p *Plugin) OnDestroy(any) error {
	return nil
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
