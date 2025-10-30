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
	Add(route string, handler func(c HttpContext))
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
	router.Add(routePrefix+"/plugin/init", server.Init)
	router.Add(routePrefix+"/plugin/list", server.List)
}

func (server *HTTPServer) Init(c HttpContext) {
	serviceName := c.Query("service")
	if serviceName == "" {
		serviceName = "default"
	}

	metaJSON := c.PostForm("meta")
	if metaJSON == "" {
		ErrorRet(c, fmt.Errorf("meta is required"))
		return
	}
	var meta Meta
	err := json.Unmarshal([]byte(metaJSON), &meta)
	if err != nil {
		ErrorRet(c, fmt.Errorf("invalid meta: %v", err))
		return
	}

	plugin, err := server.pluginManagers[serviceName].Load(meta, c)
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

func (server *HTTPServer) List(c HttpContext) {
	serviceName := c.Query("service")
	if serviceName == "" {
		serviceName = "default"
	}

	plugins := server.pluginManagers[serviceName].List()
	c.JSON(200, plugins)
}

func ErrorRet(c HttpContext, err error) {
	c.JSON(500, map[string]any{
		"error": err.Error(),
	})
}
