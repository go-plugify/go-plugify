package goplugify

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"unsafe"
)

type Util struct{}

// GetAttr used to get the attribute of an object using reflection and unsafe
func (u *Util) GetAttr(obj any, attrName string) any {
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

// CallMethod dynamically calls a method on an object, supporting both struct and pointer receivers,
// as well as converting map[string]any to struct parameters and supporting *struct parameters.
func (u *Util) CallMethod(obj any, methodName string, args ...any) ([]any, error) {
	return CallMethod(obj, methodName, args...)
}

func (u *Util) ToJSON(v any) string {
	return toJSON(v)
}

func (u *Util) StructToMap(obj any) (map[string]any, error) {
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

	result := make(map[string]any)
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

func (u *Util) StructsToMap(obj any) ([]map[string]any, error) {
	if obj == nil {
		return nil, nil
	}
	v := reflect.ValueOf(obj)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, fmt.Errorf("expected a slice or array, got %s", v.Kind())
	}

	result := make([]map[string]any, v.Len())
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

func (u *Util) Unmarshal(data []byte, v any) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

func (u *Util) Marshal(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

func (u *Util) GetContextValues(ctx context.Context) map[any]any {
	visited := map[context.Context]bool{}
	result := make(map[any]any)
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


func toJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

func CallMethod(obj any, methodName string, args ...any) ([]any, error) {
	if obj == nil {
		return nil, fmt.Errorf("object is nil")
	}

	var method reflect.Value
	v := reflect.ValueOf(obj)
	method = v.MethodByName(methodName)

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

	results := make([]any, len(out))
	for i, val := range out {
		results[i] = val.Interface()
	}
	return results, nil
}

// convertArgument attempts to convert arg to the expectedType.
// It supports basic type conversion and JSON-based conversion for complex types.
func convertArgument(arg any, expectedType reflect.Type) (reflect.Value, error) {
	argValue := reflect.ValueOf(arg)

	if argValue.Type().ConvertibleTo(expectedType) {
		return argValue.Convert(expectedType), nil
	}

	notBasicType := argValue.Kind() == reflect.Map ||
		argValue.Kind() == reflect.Struct ||
		argValue.Kind() == reflect.Slice ||
		argValue.Kind() == reflect.Array ||
		(argValue.Kind() == reflect.Ptr && argValue.Elem().Kind() == reflect.Struct)
	if notBasicType {
		if expectedType.Kind() == reflect.Ptr {
			newValue := reflect.New(expectedType.Elem())
			if err := json.Unmarshal([]byte(toJSON(arg)), newValue.Interface()); err != nil {
				return reflect.Value{}, fmt.Errorf("cannot convert %T to %s: %v", arg, expectedType, err)
			}
			return newValue, nil
		}
		newValue := reflect.New(expectedType).Elem()
		if err := json.Unmarshal([]byte(toJSON(arg)), newValue.Addr().Interface()); err != nil {
			return reflect.Value{}, fmt.Errorf("cannot convert %T to %s: %v", arg, expectedType, err)
		}
		return newValue, nil
	}

	return reflect.Value{}, fmt.Errorf("cannot convert %T to %s", arg, expectedType)
}
