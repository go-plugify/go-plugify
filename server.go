package goplugify

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"sync"
)

type HTTPServer struct {
	pluginManagers PluginManagers

	handlers map[string]Handler
	lock     sync.RWMutex
}

func InitHTTPServer(pluginManagers PluginManagers) *HTTPServer {
	return &HTTPServer{
		pluginManagers: pluginManagers,
		handlers:       make(map[string]Handler),
	}
}

type Handler func(c HttpContext)

type HttpRouter interface {
	Add(method, route string, handler Handler)
}

type HttpContext interface {
	GetHeader(key string) string
	Body() io.ReadCloser
	FormFile(name string) (*multipart.FileHeader, error)
	Query(key string) string
	JSON(code int, obj any)
	PostForm(key string) string
}

func (server *HTTPServer) RegisterRoutes(router HttpRouter, routePrefix string) {
	router.Add("POST", routePrefix+"/plugin/init", server.Init)
	router.Add("POST", routePrefix+"/plugin/run", server.Run)
	router.Add("POST", routePrefix+"/plugin/load", server.Load)
	router.Add("GET", routePrefix+"/plugin/list", server.List)
	router.Add("POST", routePrefix+"/plugin/unload", server.Unload)
	router.Add("GET", routePrefix+"/plugin/components", server.Components)
	router.Add("POST", routePrefix+"/plugin/gateway", server.Gateway)
}

func (server *HTTPServer) AddHandler(path string, handler Handler) {
	server.lock.Lock()
	defer server.lock.Unlock()
	server.handlers[path] = handler
}

func (server *HTTPServer) GetHandler(path string) (Handler, bool) {
	server.lock.RLock()
	defer server.lock.RUnlock()
	h, ok := server.handlers[path]
	return h, ok
}

func (server *HTTPServer) Init(c HttpContext) {
	plugin, err := server.loadPluginFromHTTP(c)
	if err != nil {
		ErrorRet(c, fmt.Errorf("load plugin error: %v", err))
		return
	}
	resp, err := plugin.OnRun(nil)
	if err != nil {
		ErrorRet(c, fmt.Errorf("run plugin error: %v", err))
		return
	}
	c.JSON(200, resp)
}

func (server *HTTPServer) Run(c HttpContext) {
	serviceName := c.Query("service")
	if serviceName == "" {
		serviceName = "default"
	}

	pluginID := c.Query("plugin_id")
	if pluginID == "" {
		ErrorRet(c, fmt.Errorf("plugin_id is required"))
		return
	}

	plugin, err := server.pluginManagers[serviceName].GetPlugin(pluginID)
	if err != nil {
		ErrorRet(c, fmt.Errorf("get plugin error: %v", err))
		return
	}

	resp, err := plugin.OnRun(c)
	if err != nil {
		ErrorRet(c, fmt.Errorf("run plugin error: %v", err))
		return
	}
	c.JSON(200, resp)
}

func (server *HTTPServer) List(c HttpContext) {
	serviceName := c.Query("service")
	if serviceName == "" {
		serviceName = "default"
	}

	plugins := server.pluginManagers[serviceName].ListPlugins()
	c.JSON(200, plugins)
}

func (server *HTTPServer) Components(c HttpContext) {
	serviceName := c.Query("service")
	if serviceName == "" {
		serviceName = "default"
	}
	comps := make(PluginComponentItems, 0)
	for _, comp := range server.pluginManagers[serviceName].Components().Components {
		comps = append(comps, &PluginComponentItem{
			Name:    comp.Name(),
			PkgPath: GetPkgPathOfAny(comp),
		})
	}
	comps = append(comps, &PluginComponentItem{
		Name:    "Logger",
		PkgPath: GetPkgPathOfAny(server.pluginManagers[serviceName].Components().GetLogger()),
	})
	comps = append(comps, &PluginComponentItem{
		Name:    "Util",
		PkgPath: GetPkgPathOfAny(server.pluginManagers[serviceName].Components().GetUtil()),
	})
	c.JSON(200, comps)
}

func (server *HTTPServer) Load(c HttpContext) {
	plugin, err := server.loadPluginFromHTTP(c)
	if err != nil {
		ErrorRet(c, fmt.Errorf("load plugin error: %v", err))
		return
	}
	c.JSON(200, plugin.Meta())
}

func (server *HTTPServer) Unload(c HttpContext) {
	serviceName := c.Query("service")
	if serviceName == "" {
		serviceName = "default"
	}

	pluginID := c.Query("plugin_id")
	if pluginID == "" {
		ErrorRet(c, fmt.Errorf("plugin_id is required"))
		return
	}

	manager := server.pluginManagers[serviceName]
	err := manager.UnloadPlugin(pluginID)
	if err != nil {
		ErrorRet(c, fmt.Errorf("unload plugin error: %v", err))
		return
	}
	c.JSON(200, map[string]any{
		"message": "plugin unloaded",
	})
}

func (server *HTTPServer) loadPluginFromHTTP(c HttpContext) (IPlugin, error) {
	serviceName := c.Query("service")
	if serviceName == "" {
		serviceName = "default"
	}

	metaJSON := c.PostForm("meta")
	if metaJSON == "" {
		return nil, fmt.Errorf("meta is required")
	}
	var meta = new(Meta)
	err := json.Unmarshal([]byte(metaJSON), meta)
	if err != nil {
		return nil, fmt.Errorf("invalid meta: %v", err)
	}

	plugin, err := server.pluginManagers[serviceName].LoadPlugin(meta, c)
	if err != nil {
		return nil, err
	}
	return plugin, nil
}

func (server *HTTPServer) Gateway(c HttpContext) {

	path := c.Query("path")
	if path == "" {
		ErrorRet(c, fmt.Errorf("path is required"))
		return
	}

	escapedPath, _ := url.PathUnescape(path)

	handler, ok := server.GetHandler(escapedPath)
	if !ok {
		ErrorRet(c, fmt.Errorf("no handler for path: %s", path))
		return
	}

	handler(c)
}

func ErrorRet(c HttpContext, err error) {
	c.JSON(500, map[string]any{
		"error": err.Error(),
	})
}
