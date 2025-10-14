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