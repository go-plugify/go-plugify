package goportal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"plugin"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/gin-gonic/gin"
)

var logger Logger

type Logger interface {
	WarnCtx(ctx context.Context, format string, args ...interface{})
	ErrorCtx(ctx context.Context, format string, args ...interface{})
	InfoCtx(ctx context.Context, format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Info(format string, args ...interface{})
}

type RedisClient interface {
	LPush(key string, value interface{}) error
	LRange(key string, start, stop int64) (interface{}, error)
	LRem(key string, count int64, value interface{}) (int64, error)
}

type ObjectStoreService interface {
	DownloadFile(ctx context.Context, ukey string) (string, error)
	DeleteFile(ctx context.Context, ukey string) error
	UploadFile(ctx context.Context, fileName string, content []byte) (key string, err error)
}

func downloadFileFromOSS(ctx context.Context, oss ObjectStoreService, ukey string) ([]byte, error) {
	downloadURL, err := oss.DownloadFile(ctx, ukey)
	if err != nil {
		return nil, fmt.Errorf("request download file error: %v", err)
	}
	resp, err := http.Get(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("http get error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status code: %d", resp.StatusCode)
	}
	fileContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body error: %v", err)
	}
	return fileContent, nil
}

type PluginDepencies struct {
	// business services
	LogicService interface{}
	ExtService   interface{}
	Config       interface{}

	// plugin services
	Store   *IPluginDataStore
	Plugins *IPluginManager

	// utils
	Logger *Logger
	Util   *Util

	// middlewares
	RedisCli  RedisClient
	OSS       ObjectStoreService
	GinEngine *GinEngine
}

func (p *PluginDepencies) GetLogicService() interface{} {
	return p.LogicService
}

func (p *PluginDepencies) GetLogger() interface{} {
	return p.Logger
}

func (p *PluginDepencies) GetConfig() interface{} {
	return p.Config
}

func (p *PluginDepencies) GetRedis() interface{} {
	return p.RedisCli
}

func (p *PluginDepencies) GetGinEngine() interface{} {
	return p.GinEngine
}

func (p *PluginDepencies) GetStore() interface{} {
	return p.Store
}

func (p *PluginDepencies) GetOSS() interface{} {
	return p.OSS
}

func (p *PluginDepencies) GetPlugins() interface{} {
	return p.Plugins
}

func (p *PluginDepencies) GetUtil() interface{} {
	return p.Util
}

func (p *PluginDepencies) GetExtService() interface{} {
	return p.ExtService
}

type GinEngine struct {
	Engine *gin.Engine
}

func (p *GinEngine) ReplaceHandler(method, path string, handler func(ctx context.Context)) error {
	return ReplaceLastHandler(p.Engine, method, path, func(c *gin.Context) {
		handler(c)
	})
}

func (p *GinEngine) GetHandler(method, path string) (func(ctx context.Context), error) {
	handlers, err := getHandlerSlicePointer(p.Engine, method, path)
	if err != nil {
		return nil, err
	}
	handler := (*handlers)[len(*handlers)-1]
	return func(ctx context.Context) {
		handler(ctx.(*gin.Context))
	}, nil
}

func (p *GinEngine) GetHandlerName(method, path string) (string, error) {
	handlers, err := getHandlerSlicePointer(p.Engine, method, path)
	if err != nil {
		return "", err
	}
	if handlers == nil || len(*handlers) == 0 {
		return "", fmt.Errorf("no handlers found for method: %s, route: %s", method, path)
	}
	handler := (*handlers)[len(*handlers)-1]
	handlerName := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()
	return handlerName, nil
}

func (p *GinEngine) GetRoutes() []gin.RouteInfo {
	return p.Engine.Routes()
}

func getHandlerSlicePointer(engine *gin.Engine, method string, route string) (*[]gin.HandlerFunc, error) {
	engineVal := reflect.ValueOf(engine).Elem()
	trees := engineVal.FieldByName("trees")
	if !trees.IsValid() {
		return nil, fmt.Errorf("cannot find route trees")
	}

	for i := range trees.Len() {
		tree := trees.Index(i)
		methodField := getUnexportedField(tree.FieldByName("method")).String()
		if !strings.EqualFold(methodField, method) {
			continue
		}

		root := getUnexportedField(tree.FieldByName("root"))
		handlers := findHandlersInNode(root, route, "")
		if handlers == nil {
			continue
		} else {
			return handlers, nil
		}
	}
	return nil, fmt.Errorf("handler not found for method: %s, route: %s", method, route)
}

func findHandlersInNode(node reflect.Value, target, currentPath string) *[]gin.HandlerFunc {
	if !node.IsValid() {
		return nil
	}

	if node.Kind() == reflect.Ptr {
		if node.IsNil() {
			return nil
		}
		node = node.Elem()
	}
	node = getUnexportedField(node)

	path := getUnexportedField(node.FieldByName("path")).String()
	fullPath := currentPath + path

	if fullPath == target {
		handlersField := getUnexportedField(node.FieldByName("handlers"))
		handlersPtr := (*[]gin.HandlerFunc)(unsafe.Pointer(handlersField.UnsafeAddr()))
		return handlersPtr
	}

	childrenField := getUnexportedField(node.FieldByName("children"))
	for i := range childrenField.Len() {
		child := childrenField.Index(i)
		if child.Kind() == reflect.Ptr && child.IsNil() {
			continue
		}
		if res := findHandlersInNode(child, target, fullPath); res != nil {
			return res
		}
	}
	return nil
}

func ReplaceLastHandler(engine *gin.Engine, method, route string, newHandler gin.HandlerFunc) error {
	handlersPtr, err := getHandlerSlicePointer(engine, method, route)
	if err != nil {
		return fmt.Errorf("failed to get handler slice pointer: %w", err)
	}
	if handlersPtr == nil || len(*handlersPtr) == 0 {
		return fmt.Errorf("no handlers found for method: %s, route: %s", method, route)
	}
	(*handlersPtr)[len(*handlersPtr)-1] = newHandler
	return nil
}

const (
	RedisKeyPluginPodIP     = "plugin:%s:podip"
	RedisKeyPluginInstalled = "plugin:%s:installed"
)

type Plugin struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InstallTime time.Time              `json:"install_time"`
	UpgradeTime time.Time              `json:"upgrade_time"`
	Methods     map[string]interface{} `json:"-"`
	OSSKey      string                 `json:"oss_key,omitempty"`

	lock sync.RWMutex `json:"-"`
}

func (p *Plugin) RedisKey() string {
	return fmt.Sprintf("%s_%s", p.Name, p.OSSKey)
}

func (p *Plugin) Upgrade(description string, methods map[string]interface{}) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.Description = description
	p.Methods = methods
	p.UpgradeTime = time.Now()
}

func (manager *IPluginManager) LoadPlugin(pluginso []byte) (*Plugin, error) {

	tmpfile, err := os.CreateTemp("", fmt.Sprintf("plugin_%d_*.so", time.Now().UnixNano()))
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpfile.Name())

	if _, err := io.Copy(tmpfile, bytes.NewBuffer(pluginso)); err != nil {
		return nil, err
	}

	p, err := plugin.Open(tmpfile.Name())
	if err != nil {
		return nil, err
	}
	sym, err := p.Lookup("Exports")
	if err != nil {
		return nil, err
	}
	exports := *sym.(*map[string]interface{})

	existPlugin, ok := manager.plugins.Get(exports["name"].(string))
	if ok {
		logger.Warn("[Plugin] Plugin %s already exists, upgrading...", exports["name"].(string))
		existPlugin.Upgrade(exports["description"].(string), exports)
		existPlugin.Load(manager.dependencies)
		return existPlugin, nil
	}

	plugin := &Plugin{
		Name:        exports["name"].(string),
		Description: exports["description"].(string),
		InstallTime: time.Now(),
		Methods:     exports,
	}
	plugin.Load(manager.dependencies)

	manager.plugins.Add(plugin)

	return plugin, nil
}

func (p *Plugin) Load(plugDepencies *PluginDepencies) {
	p.lock.Lock()
	defer p.lock.Unlock()

	loadMethod := p.Methods["load"].(func(interface{}))
	if loadMethod != nil {
		loadMethod(plugDepencies)
	} else {
		logger.Warn("[Plugin] No load method found in plugin %s", p.Name)
	}
}

func (p *Plugin) Init(c context.Context) {
	p.lock.Lock()
	defer p.lock.Unlock()

	initMethod := p.Methods["initialize"].(func(context.Context))
	if initMethod != nil {
		initMethod(c)
	} else {
		logger.WarnCtx(c, "[Plugin] No initialize method found in plugin %s", p.Name)
	}
}

type IPluginManager struct {
	plugins      *Plugins
	dependencies *PluginDepencies

	serviceName string

	localIP string
}

var PluginManager = &IPluginManager{
	plugins: &Plugins{
		plugins: make(map[string]*Plugin),
	},
}

func (manager *IPluginManager) GetPlugins() *Plugins {
	return manager.plugins
}

func (manager *IPluginManager) LoadAndInstallPlugin(ctx context.Context, pluginso []byte) (*Plugin, error) {
	plugin, err := manager.LoadPlugin(pluginso)
	if err != nil {
		return nil, fmt.Errorf("load plugin error: %v", err)
	}

	if plugin.OSSKey != "" {
		_, err = manager.dependencies.RedisCli.LRem(fmt.Sprintf(RedisKeyPluginInstalled, manager.serviceName), 0, plugin.RedisKey())
		if err != nil {
			logger.ErrorCtx(ctx, "Failed to remove existing plugin key %s from Redis: %v", plugin.RedisKey(), err)
		}
		_, err = manager.dependencies.OSS.DownloadFile(ctx, plugin.OSSKey)
		if err != nil {
			logger.ErrorCtx(ctx, "Failed to delete existing plugin file from OSS: key %s, err %v", plugin.OSSKey, err)
		}
	}

	storeKey, err := manager.dependencies.OSS.UploadFile(ctx, plugin.Name, pluginso)
	if err != nil {
		return nil, fmt.Errorf("upload plugin to OSS error: %v", err)
	}
	err = manager.dependencies.RedisCli.LPush(fmt.Sprintf(RedisKeyPluginInstalled, manager.serviceName), fmt.Sprintf("%s_%s", plugin.Name, storeKey))
	if err != nil {
		return nil, err
	}
	plugin.OSSKey = storeKey
	return plugin, nil
}

func (manager *IPluginManager) LoadPluginsFromOSS(ctx context.Context) error {
	installedPlugins, err := manager.dependencies.RedisCli.LRange(fmt.Sprintf(RedisKeyPluginInstalled, manager.serviceName), 0, -1)
	if err != nil {
		return fmt.Errorf("get installed plugins from redis error: %v", err)
	}

	for _, pluginInfo := range installedPlugins.([]string) {
		parts := strings.Split(pluginInfo, "_")
		if len(parts) != 2 {
			logger.WarnCtx(ctx, "Invalid plugin info format: %s", pluginInfo)
			continue
		}
		name, ukey := parts[0], parts[1]
		fileContent, err := downloadFileFromOSS(ctx, manager.dependencies.OSS, ukey)
		if err != nil {
			logger.WarnCtx(ctx, "Get plugin file content error: %v", err)
			continue
		}
		plugin, err := manager.LoadPlugin(fileContent)
		if err != nil {
			logger.WarnCtx(ctx, "Load plugin error: %v", err)
			continue
		}
		logger.InfoCtx(ctx, "Loaded plugin: %s", name)
		plugin.OSSKey = ukey
		plugin.Init(ctx)
	}

	return nil
}

func (manager *IPluginManager) List() interface{} {
	return manager.plugins.List()
}

func (manager *IPluginManager) Remove(name string) error {
	plugin, ok := manager.plugins.Get(name)
	if !ok {
		logger.Warn("[PluginManager] Plugin %s not found", name)
		return fmt.Errorf("plugin %s not found", name)
	}
	plugin.lock.Lock()
	defer plugin.lock.Unlock()

	key := plugin.RedisKey()
	_, err := manager.dependencies.RedisCli.LRem(fmt.Sprintf(RedisKeyPluginInstalled, manager.serviceName), 0, key)
	if err != nil {
		logger.Error("[PluginManager] Failed to remove plugin key %s from Redis: %v", key, err)
		return fmt.Errorf("failed to remove plugin key %s from Redis: %v", key, err)
	}
	logger.Info("[PluginManager] Removed plugin key %s from Redis", key)

	manager.plugins.Remove(name)
	logger.Info("[PluginManager] Plugin %s removed successfully", name)
	return nil
}

type Plugins struct {
	plugins map[string]*Plugin
	mu      sync.RWMutex
}

func (p *Plugins) Add(plugin *Plugin) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.plugins[plugin.Name] = plugin
}

func (p *Plugins) Get(name string) (*Plugin, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	plugin, ok := p.plugins[name]
	return plugin, ok
}

func (p *Plugins) Remove(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.plugins, name)
}

func (p *Plugins) List() []*Plugin {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var plugins []*Plugin
	for _, plugin := range p.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}
