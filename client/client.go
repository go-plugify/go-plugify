package client

import (
	"context"
)


type Plugin struct{ 
	Dependencies Dependencies

	Initialize func(c context.Context)
}

type PluginInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version,omitempty"`

	*Plugin
}

func (p PluginInfo) GetName() string {
	return p.Name
}

func (p PluginInfo) GetDescription() string {
	return p.Description
}

func (p *Plugin) Load(dependencies any) {
	p.Dependencies = dependencies.(Dependencies)
}

type Dependencies interface {
	GetGinEngine() any
}

type GinContext interface {
	context.Context
	Query(key string) string
	JSON(code int, obj any)
	Data(code int, contentType string, data []byte)
	Get(key string) (value any, exists bool)
	ShouldBindJSON(obj any) error
}

type GinEngine interface {
	ReplaceHandler(method, path string, handler func(ctx context.Context)) error
}