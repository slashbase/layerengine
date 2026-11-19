package layerengine

import (
	"errors"
	"fmt"
	"math"
	"reflect"

	lua "github.com/yuin/gopher-lua"
)

// maxExactLuaInteger is 2^53, the largest integer a float64 — and therefore a
// Lua number — represents exactly. Values past it are rejected rather than
// silently rounded, in the signed case as well as the unsigned one.
const maxExactLuaInteger = 1 << 53

// maxValueDepth bounds how far a value is walked when converting between Go and
// Lua.
//
// A Lua table can reference itself, and the conversion below is Go recursion
// that runs after the layer's context has been torn down: nothing can interrupt
// it. Without a bound, a layer that returns a cyclic table overflows the
// goroutine stack, which is a fatal error rather than a recoverable panic, and
// takes the whole process down with it.
const maxValueDepth = 100

// ConvertGoValueToLuaValue converts a Go value into a Lua value. Unsupported
// values return an error rather than crashing the caller.
func ConvertGoValueToLuaValue(input any) (lua.LValue, error) {
	if input == nil {
		return lua.LNil, nil
	}
	if value, ok := input.(lua.LValue); ok {
		return value, nil
	}
	return convertGoValueToLuaValue(reflect.ValueOf(input), 0)
}

// convertGoValueToLuaValue walks a Go value into Lua. depth bounds the walk: a
// Go map, slice or pointer can contain itself, so the recursion has to end in
// an error rather than in a stack overflow.
func convertGoValueToLuaValue(value reflect.Value, depth int) (lua.LValue, error) {
	if depth > maxValueDepth {
		return nil, fmt.Errorf("value nested deeper than %d levels", maxValueDepth)
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return lua.LNil, nil
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Bool:
		return lua.LBool(value.Bool()), nil
	case reflect.String:
		return lua.LString(value.String()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		integer := value.Int()
		if integer > maxExactLuaInteger || integer < -maxExactLuaInteger {
			return nil, fmt.Errorf("integer %d cannot be represented exactly as a Lua number", integer)
		}
		return lua.LNumber(integer), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		integer := value.Uint()
		if integer > maxExactLuaInteger {
			return nil, fmt.Errorf("unsigned integer %d cannot be represented exactly as a Lua number", integer)
		}
		return lua.LNumber(integer), nil
	case reflect.Float32, reflect.Float64:
		number := value.Float()
		if math.IsNaN(number) || math.IsInf(number, 0) {
			return nil, fmt.Errorf("non-finite float %v cannot be represented as a Lua number", number)
		}
		return lua.LNumber(number), nil
	case reflect.Slice, reflect.Array:
		if value.Kind() == reflect.Slice && value.IsNil() {
			return lua.LNil, nil
		}
		table := &lua.LTable{}
		for i := 0; i < value.Len(); i++ {
			item, err := convertGoValueToLuaValue(value.Index(i), depth+1)
			if err != nil {
				return nil, fmt.Errorf("convert array item %d: %w", i, err)
			}
			table.RawSetInt(i+1, item)
		}
		return table, nil
	case reflect.Map:
		if value.IsNil() {
			return lua.LNil, nil
		}
		if value.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("unsupported map key type %s; Lua maps require string keys", value.Type().Key())
		}
		table := &lua.LTable{}
		iter := value.MapRange()
		for iter.Next() {
			item, err := convertGoValueToLuaValue(iter.Value(), depth+1)
			if err != nil {
				return nil, fmt.Errorf("convert map value for key %q: %w", iter.Key().String(), err)
			}
			table.RawSetString(iter.Key().String(), item)
		}
		return table, nil
	default:
		return nil, fmt.Errorf("unsupported Go value type %s", value.Type())
	}
}

// ConvertLuaValueToGoValue converts a Lua value to its Go representation.
//
// The conversion is bounded: a table that reaches itself, or nests deeper than
// maxValueDepth, returns an error instead of recursing. This runs after the
// layer's context is gone, so the bound is the only thing standing between
// generated code and a process-wide crash.
func ConvertLuaValueToGoValue(value lua.LValue) (any, error) {
	return convertLuaValue(value, make(map[*lua.LTable]struct{}), 0)
}

// convertLuaValue is ConvertLuaValueToGoValue carrying the recursion state.
//
// seen holds the tables on the current path, so a table reached twice by
// different branches — a DAG, which is legal and common — converts fine, while
// a table that reaches itself is reported as a cycle.
func convertLuaValue(value lua.LValue, seen map[*lua.LTable]struct{}, depth int) (any, error) {
	if value == nil {
		return nil, nil
	}

	switch value.Type() {
	case lua.LTNil:
		return nil, nil
	case lua.LTBool:
		return lua.LVAsBool(value), nil
	case lua.LTNumber:
		return lua.LVAsNumber(value), nil
	case lua.LTString:
		return lua.LVAsString(value), nil
	case lua.LTTable:
		return convertLuaTableToGo(value.(*lua.LTable), seen, depth)
	default:
		return nil, fmt.Errorf("unsupported Lua value type %s", value.Type())
	}
}

func ConvertGoValuesToLuaValues(inputs []any) ([]lua.LValue, error) {
	outputs := make([]lua.LValue, len(inputs))
	for i, value := range inputs {
		converted, err := ConvertGoValueToLuaValue(value)
		if err != nil {
			return nil, fmt.Errorf("convert input %d: %w", i, err)
		}
		outputs[i] = converted
	}
	return outputs, nil
}

func ConvertLuaValuesToGoValues(inputs []lua.LValue) ([]any, error) {
	outputs := make([]any, len(inputs))
	for i, value := range inputs {
		converted, err := ConvertLuaValueToGoValue(value)
		if err != nil {
			return nil, fmt.Errorf("convert output %d: %w", i, err)
		}
		outputs[i] = converted
	}
	return outputs, nil
}

type luaTableEntry struct {
	key   lua.LValue
	value lua.LValue
}

func convertLuaTableToGo(table *lua.LTable, seen map[*lua.LTable]struct{}, depth int) (any, error) {
	if depth > maxValueDepth {
		return nil, fmt.Errorf("table nested deeper than %d levels", maxValueDepth)
	}
	if _, cycle := seen[table]; cycle {
		return nil, errors.New("table contains a cycle and cannot be converted")
	}
	seen[table] = struct{}{}
	// Deleting on the way out makes the check path-based: the same table reached
	// by two sibling branches is a DAG, not a cycle, and must still convert.
	defer delete(seen, table)

	entries := make([]luaTableEntry, 0, table.Len())
	table.ForEach(func(key, value lua.LValue) {
		entries = append(entries, luaTableEntry{key: key, value: value})
	})

	if isDenseLuaArray(entries) {
		array := make([]any, len(entries))
		for _, entry := range entries {
			index := int(lua.LVAsNumber(entry.key))
			converted, err := convertLuaValue(entry.value, seen, depth+1)
			if err != nil {
				return nil, fmt.Errorf("convert array item %d: %w", index, err)
			}
			array[index-1] = converted
		}
		return array, nil
	}

	// A table whose keys are all strings is a record, and converts to
	// map[string]any: it round-trips back through ConvertGoValueToLuaValue, which
	// accepts only string-keyed maps, and it marshals as JSON. Numeric or mixed
	// keys keep map[any]any so no key is lost. An empty table lands here and
	// becomes an empty map[string]any — an empty table cannot be told apart from
	// an empty array, so that choice is arbitrary but stable.
	if allStringKeys(entries) {
		mapped := make(map[string]any, len(entries))
		for _, entry := range entries {
			key := lua.LVAsString(entry.key)
			value, err := convertLuaValue(entry.value, seen, depth+1)
			if err != nil {
				return nil, fmt.Errorf("convert table value for key %q: %w", key, err)
			}
			mapped[key] = value
		}
		return mapped, nil
	}

	mapped := make(map[any]any, len(entries))
	for _, entry := range entries {
		key, err := convertLuaValue(entry.key, seen, depth+1)
		if err != nil {
			return nil, fmt.Errorf("convert table key: %w", err)
		}
		value, err := convertLuaValue(entry.value, seen, depth+1)
		if err != nil {
			return nil, fmt.Errorf("convert table value for key %v: %w", key, err)
		}
		mapped[key] = value
	}
	return mapped, nil
}

func isDenseLuaArray(entries []luaTableEntry) bool {
	if len(entries) == 0 {
		return false
	}

	indexes := make(map[int]struct{}, len(entries))
	for _, entry := range entries {
		if entry.key.Type() != lua.LTNumber {
			return false
		}
		number := float64(lua.LVAsNumber(entry.key))
		if number < 1 || number != math.Trunc(number) || number > float64(len(entries)) {
			return false
		}
		indexes[int(number)] = struct{}{}
	}

	return len(indexes) == len(entries)
}

// allStringKeys reports whether every key in entries is a Lua string. The empty
// case is vacuously true, which is what makes an empty table a map rather than
// an array.
func allStringKeys(entries []luaTableEntry) bool {
	for _, entry := range entries {
		if entry.key.Type() != lua.LTString {
			return false
		}
	}
	return true
}
