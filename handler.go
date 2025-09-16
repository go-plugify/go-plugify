package goportal

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
)

func InitGin(router *gin.Engine, routePrefix string) {
	router.POST(routePrefix+"/plugin/init", PluginInit)
	PluginManager = make(map[string]*IPluginManager)
	PluginManager["default"] = &IPluginManager{
		plugins: &Plugins{
			plugins: make(map[string]*Plugin),
		},
		dependencies: &PluginDepencies{
			GinEngine: &GinEngine{
				Engine: router,
			},
		},
	}
	logger = &DefaultLogger{}
}

type DefaultLogger struct{}

func (l *DefaultLogger) WarnCtx(ctx context.Context, format string, args ...any) {
	fmt.Printf("[WARN] "+format+"\n", args...)
}
func (l *DefaultLogger) ErrorCtx(ctx context.Context, format string, args ...any) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}
func (l *DefaultLogger) InfoCtx(ctx context.Context, format string, args ...any) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}
func (l *DefaultLogger) Warn(format string, args ...any) {
	fmt.Printf("[WARN] "+format+"\n", args...)
}
func (l *DefaultLogger) Error(format string, args ...any) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}
func (l *DefaultLogger) Info(format string, args ...any) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

func _getPluginContent(c *gin.Context) ([]byte, error) {
	var fileContent []byte

	ct := c.Request.Header.Get("Content-Type")
	if !strings.Contains(ct, "multipart/form-data") {
		fileContent, _ = io.ReadAll(c.Request.Body)
	} else {
		file, err := c.FormFile("file")
		if err != nil {
			body, _ := io.ReadAll(c.Request.Body)
			return nil, fmt.Errorf("file error: %v raw ct: %s, body length: %d", err, ct, len(body))
		}
		f, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer f.Close()

		fileContent, err = io.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("read file error: %v", err)
		}
	}

	return fileContent, nil
}

func PluginInit(c *gin.Context) {
	fileContent, err := _getPluginContent(c)
	if err != nil {
		ErrorRet(c, fmt.Errorf("get plugin content error: %v", err))
		return
	}

	serviceName := c.Query("service")
	if serviceName == "" {
		serviceName = "default"
	}

	plugin, err := PluginManager[serviceName].LoadPlugin(fileContent)
	if err != nil {
		ErrorRet(c, fmt.Errorf("load plugin error: %v", err))
		return
	}
	plugin.Init(c)
}

func ErrorRet(c *gin.Context, err error) {
	c.JSON(500, gin.H{
		"error": err.Error(),
	})
}

var PluginManager = make(map[string]*IPluginManager)
