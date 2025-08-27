package goportal

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
)

type IPluginDataStore struct {
	kv sync.Map
}

func (p *IPluginDataStore) Set(key string, value interface{}) {
	p.kv.Store(key, value)
}
func (p *IPluginDataStore) Get(key string) (interface{}, bool) {
	value, ok := p.kv.Load(key)
	if !ok {
		return nil, false
	}
	return value, true
}

var PluginDataStore = &IPluginDataStore{
	kv: sync.Map{},
}

func (p *IPluginDataStore) ConfigBool(key string) bool {
	boolValue, ok := p.Config(key)
	return ok && boolValue != nil && boolValue.(bool)
}

func (p *IPluginDataStore) ConfigInt64(key string) int64 {
	value, ok := p.Config(key)
	if !ok || value == nil {
		return 0
	}
	if intValue, ok := value.(int64); ok {
		return intValue
	}
	return 0
}

func (p *IPluginDataStore) ConfigInt64Slice(key string) []int64 {
	value, ok := p.Config(key)
	if !ok || value == nil {
		return nil
	}
	if intValue, ok := value.([]int64); ok {
		return intValue
	}
	return nil
}

func (p *IPluginDataStore) ConfigIgnore(key string) interface{} {
	value, ok := p.Config(key)
	if !ok {
		return nil
	}
	return value
}

func (p *IPluginDataStore) Config(key string) (interface{}, bool) {
	config, ok := p.Get("config")
	if !ok {
		return nil, false
	}
	if configMap, ok := config.(map[string]interface{}); ok {
		value, ok := configMap[key]
		if !ok {
			return nil, false
		}
		return value, true
	}
	return nil, false
}

func (p *IPluginDataStore) Stub(stubKey string, ctx context.Context, args ...interface{}) {
	stubs, ok := p.Get("stubs")
	if !ok {
		return
	}
	if stubMap, ok := stubs.(map[string]interface{}); ok {
		if stubFunc, ok := stubMap[stubKey].(func(context.Context, []interface{})); ok {
			stubFunc(ctx, args)
		}
	}
}

func Stub(stubKey string, ctx context.Context, args ...interface{}) {
	PluginDataStore.Stub(stubKey, ctx, args...)
}

func (p *IPluginDataStore) StubFunc(stubKey string, ctx context.Context, args []interface{}, resp []interface{}, rawFn func()) {
	stubs, ok := p.Get("stubs")
	if !ok {
		rawFn()
		return
	}
	stubMap, ok := stubs.(map[string]interface{})
	if !ok {
		rawFn()
		return
	}
	stubFunc, ok := stubMap[stubKey].(func(context.Context, []interface{}, []interface{}, func()))
	if !ok {
		rawFn()
		return
	}
	stubFunc(ctx, args, resp, rawFn)
}

func StubFunc(stubKey string, ctx context.Context, args []interface{}, resp []interface{}, rawFn func()) {
	PluginDataStore.StubFunc(stubKey, ctx, args, resp, rawFn)
}

func CtxWithStub(ctx context.Context) context.Context {
	return context.WithValue(ctx, "plugin_stub", PluginDataStore.Stub)
}

func CtxWithStubFunc(ctx context.Context) context.Context {
	return context.WithValue(ctx, "plugin_stub_func", PluginDataStore.StubFunc)
}

func toJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

// CallMethod 动态调用对象的方法，支持结构体和指针接收者，支持将 map[string]interface{} 转换为 struct 入参，支持 *struct 入参
func CallMethod(obj interface{}, methodName string, args ...interface{}) ([]interface{}, error) {
	if obj == nil {
		return nil, fmt.Errorf("object is nil")
	}

	var method reflect.Value
	v := reflect.ValueOf(obj)
	method = v.MethodByName(methodName)

	// 如果方法在值上找不到，尝试在指针上找
	if !method.IsValid() && v.Kind() != reflect.Ptr {
		vPtr := reflect.New(v.Type())
		vPtr.Elem().Set(v)
		method = vPtr.MethodByName(methodName)
	}

	if !method.IsValid() {
		return nil, fmt.Errorf("method %s not found", methodName)
	}

	methodType := method.Type()

	in := make([]reflect.Value, 0, len(args))
	for i, arg := range args {
		expectedType := methodType.In(i)
		argValue, err := convertArgument(arg, expectedType)
		if err != nil {
			return nil, fmt.Errorf("argument %d: %v", i, err)
		}
		in = append(in, argValue)
	}

	out := method.Call(in)

	results := make([]interface{}, len(out))
	for i, val := range out {
		results[i] = val.Interface()
	}
	return results, nil
}

// convertArgument 处理参数转换
func convertArgument(arg interface{}, expectedType reflect.Type) (reflect.Value, error) {
	argValue := reflect.ValueOf(arg)

	// 普通类型转换
	if argValue.Type().ConvertibleTo(expectedType) {
		return argValue.Convert(expectedType), nil
	}

	// 如果入参是复杂类型，比如map或者包含struct的类型，则使用json进行转换
	notBasicType := argValue.Kind() == reflect.Map ||
		argValue.Kind() == reflect.Struct ||
		argValue.Kind() == reflect.Slice ||
		argValue.Kind() == reflect.Array ||
		// 结构体指针
		(argValue.Kind() == reflect.Ptr && argValue.Elem().Kind() == reflect.Struct)
	if notBasicType {
		// 如果期望的类型是指针类型，则需要转换为指针
		if expectedType.Kind() == reflect.Ptr {
			// 如果是指针类型，先创建一个新的指针
			newValue := reflect.New(expectedType.Elem())
			// 将arg转换为expectedType.Elem()类型
			if err := json.Unmarshal([]byte(toJSON(arg)), newValue.Interface()); err != nil {
				return reflect.Value{}, fmt.Errorf("cannot convert %T to %s: %v", arg, expectedType, err)
			}
			// 返回指针类型的值
			return newValue, nil
		}
		// 如果期望的类型是非指针类型，则直接转换为expectedType
		newValue := reflect.New(expectedType).Elem()
		if err := json.Unmarshal([]byte(toJSON(arg)), newValue.Addr().Interface()); err != nil {
			return reflect.Value{}, fmt.Errorf("cannot convert %T to %s: %v", arg, expectedType, err)
		}
		// 返回非指针类型的值
		return newValue, nil
	}

	return reflect.Value{}, fmt.Errorf("cannot convert %T to %s", arg, expectedType)
}
