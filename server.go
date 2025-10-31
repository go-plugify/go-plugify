package goplugify

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
)

type HTTPServer struct {
	pluginManagers PluginManagers
}

func InitHTTPServer(pluginManagers PluginManagers) *HTTPServer {
	return &HTTPServer{
		pluginManagers: pluginManagers,
	}
}

type HttpRouter interface {
	Add(method, route string, handler func(c HttpContext))
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

	resp, err := plugin.OnRun(nil)
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

func ErrorRet(c HttpContext, err error) {
	c.JSON(500, map[string]any{
		"error": err.Error(),
	})
}
