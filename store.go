package goportal

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"unsafe"
)

type Util struct{}

// 使用反射和unsafe获取某个对象的属性
func (u *Util) GetAttr(obj interface{}, attrName string) interface{} {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if !v.IsValid() {
		return nil
	}

	field := v.FieldByName(attrName)
	if !field.IsValid() {
		return nil
	}

	if field.CanInterface() {
		return field.Interface()
	}

	return getUnexportedField(field).Interface()
}

// CallMethod 动态调用对象的方法，支持结构体和指针接收者，支持将 map[string]interface{} 转换为 struct 入参，支持 *struct 入参
func (u *Util) CallMethod(obj interface{}, methodName string, args ...interface{}) ([]interface{}, error) {
	return CallMethod(obj, methodName, args...)
}

func (u *Util) ToJSON(v interface{}) string {
	return toJSON(v)
}

func (u *Util) StructToMap(obj interface{}) (map[string]interface{}, error) {
	if obj == nil {
		return nil, nil
	}
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected a struct, got %s", v.Kind())
	}

	result := make(map[string]interface{})
	for i := range v.NumField() {
		field := v.Type().Field(i)
		value := v.Field(i)
		if value.CanInterface() {
			result[field.Name] = value.Interface()
		} else {
			result[field.Name] = getUnexportedField(value).Interface()
		}
	}
	return result, nil
}

func (u *Util) StructsToMap(obj interface{}) ([]map[string]interface{}, error) {
	if obj == nil {
		return nil, nil
	}
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, fmt.Errorf("expected a slice or array, got %s", v.Kind())
	}

	result := make([]map[string]interface{}, v.Len())
	for i := range v.Len() {
		item := v.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		if item.Kind() != reflect.Struct {
			return nil, fmt.Errorf("expected a struct, got %s", item.Kind())
		}
		m, err := u.StructToMap(item.Interface())
		if err != nil {
			return nil, err
		}
		result[i] = m
	}
	return result, nil
}

func (u *Util) Unmarshal(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

func (u *Util) Marshal(v interface{}) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

func (u *Util) GetContextValues(ctx context.Context) map[interface{}]interface{} {
	visited := map[context.Context]bool{}
	result := make(map[interface{}]interface{})
	for ctx != nil {
		if visited[ctx] {
			break
		}
		visited[ctx] = true

		val := reflect.ValueOf(ctx)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		if val.Kind() != reflect.Struct {
			break
		}

		if val.NumField() >= 2 {
			keyField := u.GetAttr(ctx, "key")
			valField := u.GetAttr(ctx, "val")
			parentField := val.FieldByName("Context")
			result[keyField] = valField

			var ok bool
			ctx, ok = parentField.Interface().(context.Context)
			if ok && ctx != nil {
				continue
			}
		}
		break
	}
	return result
}

func getUnexportedField(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if !v.CanAddr() {
		panic("value is not addressable")
	}
	return reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
}
