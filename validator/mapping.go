package validator

import (
	"reflect"
	"strings"
)

// StructToMap converts any struct into a map[string]any using its `key` struct
// tags. Nested structs and slices are handled recursively.
func StructToMap(v any) map[string]any {
	m, ok := structToMap(reflect.ValueOf(v))
	if !ok {
		return nil
	}
	return m
}

func structToMap(v reflect.Value) (map[string]any, bool) {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, false
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, false
	}

	t := v.Type()
	out := make(map[string]any, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("key")
		if tag == "" || tag == "-" {
			continue
		}
		// strip options after comma, e.g. "name,omitempty"
		if idx := strings.IndexByte(tag, ','); idx >= 0 {
			tag = tag[:idx]
		}

		out[tag] = valueToAny(v.Field(i))
	}

	return out, true
}

func valueToAny(v reflect.Value) any {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	switch v.Kind() {

	case reflect.Struct:
		if m, ok := structToMap(v); ok {
			return m
		}
		return nil

	case reflect.Slice, reflect.Array:
		if v.IsNil() {
			return nil
		}
		out := make([]any, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			out = append(out, valueToAny(v.Index(i)))
		}
		return out

	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return v.Interface()
		}
		out := make(map[string]any, v.Len())
		for _, k := range v.MapKeys() {
			out[k.String()] = valueToAny(v.MapIndex(k))
		}
		return out

	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.String:
		return v.String()

	default:
		return v.Interface()
	}
}
