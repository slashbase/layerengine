package layerengine

import (
	"reflect"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

type Layer struct {
	Name        string             `key:"name"`
	Description string             `key:"description"`
	Input       []string           `key:"input"`
	Output      []LayerOutput      `key:"output"`
	FnProto     *lua.FunctionProto `key:"-"`
	Code        string             `key:"-"`
}

type LayerOutput struct {
	Name        string `key:"name"`
	Type        string `key:"type"`
	Description string `key:"description"`
}

type FlowInput struct {
	Name        string `key:"name"`
	Type        string `key:"type"`
	Description string `key:"description"`
	Optional    bool   `key:"optional"`
}

type Flow struct {
	Name        string      `key:"name"`
	Description string      `key:"description"`
	Input       []FlowInput `key:"input"`
	Layers      []Layer     `key:"layers"`
}

// Keep all structs above this line
// Mapping utilities for above structs

// MapToStruct populates the struct pointed to by v from a map[string]any,
// using each field's `key` tag to look up values in the map.
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
		// opaque pointer types like *lua.FunctionProto —
		// stored as-is in the map, assign directly if types match
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

// StructToMap converts the struct (or struct pointed to by v) into a
// map[string]any, using each field's `key` tag as the map key. Fields
// tagged `key:"-"` or with no tag are skipped. This is the inverse of
// MapToStruct for round-tripping Layer/Flow values.
func StructToMap(v any) map[string]any {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}
	return readStruct(rv)
}

func readStruct(rv reflect.Value) map[string]any {
	t := rv.Type()
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
		if idx := strings.IndexByte(tag, ','); idx >= 0 {
			tag = tag[:idx]
		}

		out[tag] = readField(rv.Field(i))
	}
	return out
}

func readField(fv reflect.Value) any {
	switch fv.Kind() {

	case reflect.String:
		return fv.String()

	case reflect.Bool:
		return fv.Bool()

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fv.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fv.Uint()

	case reflect.Float32, reflect.Float64:
		return fv.Float()

	case reflect.Slice:
		return readSlice(fv)

	case reflect.Struct:
		return readStruct(fv)

	case reflect.Ptr:
		// opaque pointer types like *lua.FunctionProto — return as-is
		// so it survives a round-trip through MapToStruct.
		if fv.IsNil() {
			return nil
		}
		return fv.Interface()

	default:
		return fv.Interface()
	}
}

func readSlice(fv reflect.Value) []any {
	if fv.IsNil() {
		return nil
	}
	n := fv.Len()
	out := make([]any, n)

	elemKind := fv.Type().Elem().Kind()

	for i := 0; i < n; i++ {
		ev := fv.Index(i)

		switch elemKind {
		case reflect.String:
			out[i] = ev.String()
		case reflect.Struct:
			out[i] = readStruct(ev)
		default:
			out[i] = readField(ev)
		}
	}
	return out
}
