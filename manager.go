package goplugify

import "fmt"

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
	Load(meta Meta, src any) (IPlugin, error)
	List() []IPlugin
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

func (manager *PluginManager) Load(meta Meta, src any) (IPlugin, error) {

	loader, ok := manager.loads[meta.Loader]
	if !ok {
		return nil, fmt.Errorf("loader %s not found", meta.Loader)
	}

	exports, err := loader.Load(meta, src)
	if err != nil {
		return nil, err
	}

	existPlugin, ok := manager.plugins.Get(exports.Meta().ID)
	if ok {
		logger.Warn("[Plugin] Plugin %s already exists, upgrading...", exports.Meta().ID)
		existPlugin.OnInit(manager.components)
		return existPlugin, nil
	}
	exports.OnInit(manager.components)
	manager.plugins.Add(exports)

	return exports, nil
}

func (manager *PluginManager) List() []IPlugin {
	return manager.plugins.List()
}

type PluginManagers map[string]Manager
