package goplugify

type Component interface {
	Name() string
	Service() any
}

type DefaultComponent struct {
	name string
	svr  any
}

func (d *DefaultComponent) Name() string {
	return d.name
}

func (d *DefaultComponent) Service() any {
	return d.svr
}

func ComponentWithName(name string, svr any) Component {
	return &DefaultComponent{
		name: name,
		svr:  svr,
	}
}

func LoggerComponent(logger Logger) Component {
	return &DefaultComponent{
		name: "logger",
		svr:  logger,
	}
}

type PluginComponents struct {
	Logger Logger
	Util   *Util

	Components Components
}

func (p *PluginComponents) GetLogger() any {
	return p.Logger
}

func (p *PluginComponents) GetUtil() any {
	return p.Util
}

func (p *PluginComponents) GetComponents() any {
	return p.Components
}

type Components map[string]Component

func (c Components) Get(name string) any {
	return c[name]
}