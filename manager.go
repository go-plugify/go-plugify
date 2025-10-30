package goplugify

import (
	"fmt"
)

func InitPluginManagers(serviceName string, components ...Component) PluginManagers {
	extendCompones := make(map[string]Component)
	for _, c := range components {
		extendCompones[c.Name()] = c
		if _, isLogger := c.Service().(Logger); isLogger {
			logger = c.Service().(Logger)
		}
	}
	if serviceName == "" {
		serviceName = "default"
	}
	if logger == nil {
		logger = &DefaultLogger{}
	}
	managers := make(PluginManagers)
	managers[serviceName] = &PluginManager{
		plugins: &Plugins{
			plugins: make(map[string]IPlugin),
		},
		components: &PluginComponents{
			Logger:     logger,
			Util:       new(Util),
			Components: extendCompones,
		},
	}
	managers[serviceName].AddLoader(new(NativePluginHTTPLoader))
	managers[serviceName].AddLoader(new(YaegiHTTPLoader))
	return managers
}

type Manager interface {
	AddLoader(loader Loader)

	LoadPlugin(meta *Meta, src any) (IPlugin, error)
	AddPlugin(plugin IPlugin)
	ListPlugins() []IPlugin
	GetPlugin(pluginID string) (IPlugin, error)
	UnloadPlugin(pluginID string) error

	Components() *PluginComponents
}

type PluginManager struct {
	plugins    *Plugins
	components *PluginComponents
	loads      map[string]Loader

	serviceName string
}

func (manager *PluginManager) Components() *PluginComponents {
	return manager.components
}

func (manager *PluginManager) AddLoader(loader Loader) {
	manager.loads[loader.Name()] = loader
}

func (manager *PluginManager) AddPlugin(plugin IPlugin) {
	manager.plugins.Add(plugin)
}

func (manager *PluginManager) UnloadPlugin(pluginID string) error {
	plugin, ok := manager.plugins.Get(pluginID)
	if !ok {
		return fmt.Errorf("plugin %s not found", pluginID)
	}
	err := plugin.OnDestroy(nil)
	if err != nil {
		return err
	}
	manager.plugins.Remove(pluginID)
	return nil
}

func (manager *PluginManager) LoadPlugin(meta *Meta, src any) (IPlugin, error) {

	if meta == nil || meta.ID == "" || meta.Loader == "" {
		return nil, ErrInvalidLoaderSource
	}

	loader, ok := manager.loads[meta.Loader]
	if !ok {
		return nil, fmt.Errorf("loader %s not found", meta.Loader)
	}

	loadPlug, err := loader.Load(meta, src)
	if err != nil {
		return nil, err
	}

	loadPlug.OnInit(manager.components)
	existPlug, ok := manager.plugins.Get(meta.ID)
	if ok {
		existPlug.Upgrade(loadPlug)
		return existPlug, nil
	}
	manager.plugins.Add(loadPlug)

	return loadPlug, nil
}

func (manager *PluginManager) ListPlugins() []IPlugin {
	return manager.plugins.List()
}

func (manager *PluginManager) GetPlugin(pluginID string) (IPlugin, error) {
	plugin, ok := manager.plugins.Get(pluginID)
	if !ok {
		return nil, fmt.Errorf("plugin %s not found", pluginID)
	}
	return plugin, nil
}

type PluginManagers map[string]Manager
