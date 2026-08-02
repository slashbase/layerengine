package codegen

import (
	"reflect"
	"strings"
)

// MapToStruct populates the struct pointed to by v from a map[string]any,
// using each field's `key` tag to look up values in the map. It is the
// reflect-based counterpart of layerengine.MapToStruct for the codegen
// package's simple structs (Input, Output, and any future ones).
func MapToStruct(m map[string]any, v any) {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}
	populateStruct(m, rv)
}

func populateStruct(m map[string]any, rv reflect.Value) {
	t := rv.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("key")
		if tag == "" || tag == "-" {
			continue
		}
		if idx := strings.IndexByte(tag, ','); idx >= 0 {
			tag = tag[:idx]
		}

		val, ok := m[tag]
		if !ok {
			continue
		}

		setField(rv.Field(i), val)
	}
}

func setField(fv reflect.Value, val any) {
	if val == nil {
		return
	}

	switch fv.Kind() {

	case reflect.String:
		if s, ok := val.(string); ok {
			fv.SetString(s)
		}

	case reflect.Bool:
		if b, ok := val.(bool); ok {
			fv.SetBool(b)
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch n := val.(type) {
		case int:
			fv.SetInt(int64(n))
		case int64:
			fv.SetInt(n)
		case float64:
			fv.SetInt(int64(n))
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch n := val.(type) {
		case uint:
			fv.SetUint(uint64(n))
		case uint64:
			fv.SetUint(n)
		case int:
			fv.SetUint(uint64(n))
		case float64:
			fv.SetUint(uint64(n))
		}

	case reflect.Float32, reflect.Float64:
		switch n := val.(type) {
		case float64:
			fv.SetFloat(n)
		case int:
			fv.SetFloat(float64(n))
		}

	case reflect.Slice:
		setSlice(fv, val)

	case reflect.Struct:
		if m, ok := val.(map[string]any); ok {
			populateStruct(m, fv)
		}

	case reflect.Ptr:
		rv := reflect.ValueOf(val)
		if rv.IsValid() && rv.Type().AssignableTo(fv.Type()) {
			fv.Set(rv)
		}

	default:
		rv := reflect.ValueOf(val)
		if rv.IsValid() && rv.Type().AssignableTo(fv.Type()) {
			fv.Set(rv)
		}
	}
}

func setSlice(fv reflect.Value, val any) {
	arr, ok := val.([]any)
	if !ok {
		return
	}

	elemType := fv.Type().Elem()

	switch elemType.Kind() {

	case reflect.String:
		strs := make([]string, 0, len(arr))
		for _, v := range arr {
			if s, ok := v.(string); ok {
				strs = append(strs, s)
			}
		}
		fv.Set(reflect.ValueOf(strs))

	case reflect.Struct:
		slice := reflect.MakeSlice(fv.Type(), 0, len(arr))
		for _, v := range arr {
			if m, ok := v.(map[string]any); ok {
				elem := reflect.New(elemType).Elem()
				populateStruct(m, elem)
				slice = reflect.Append(slice, elem)
			}
		}
		fv.Set(slice)

	default:
		slice := reflect.MakeSlice(fv.Type(), 0, len(arr))
		for _, v := range arr {
			elem := reflect.New(elemType).Elem()
			setField(elem, v)
			slice = reflect.Append(slice, elem)
		}
		fv.Set(slice)
	}
}
