package main

import (
	"context"
	"fmt"

	client "github.com/chenhg5/go-portal/client"
)

var _ = client.Plugin{}

var ExportPlugin = client.PluginInfo{
	Name:        "example",
	Description: "An example plugin",
	Version:     "v0.1.0",
	Plugin:      &client.Plugin{},
}

func init() {
	ExportPlugin.Initialize = func(c context.Context) {
		defer func() {
			if err := recover(); err != nil {
				c.(client.GinContext).JSON(500, map[string]any{"error": fmt.Sprintf("%s", err)})
			}
		}()
		ginCtx := c.(client.GinContext)
		p := ExportPlugin.Plugin
		p.GetGinEngine().ReplaceHandler("GET", "/", func(ctx context.Context) {
			ginCtx := ctx.(client.GinContext)
			ginCtx.JSON(200, map[string]any{"message": "Hello from plugin"})
		})
		ginCtx.JSON(200, map[string]any{"message": "Hello"})
	}
}
