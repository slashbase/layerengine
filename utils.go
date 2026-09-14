package layerengine

import (
	"fmt"
	"math"
	"reflect"

	lua "github.com/yuin/gopher-lua"
)

const maxExactLuaInteger = uint64(1 << 53)

// ConvertGoValueToLuaValue converts a Go value into a Lua value. Unsupported
// values return an error rather than crashing the caller.
func ConvertGoValueToLuaValue(input any) (lua.LValue, error) {
	if input == nil {
		return lua.LNil, nil
	}
	if value, ok := input.(lua.LValue); ok {
		return value, nil
	}
	return convertGoValueToLuaValue(reflect.ValueOf(input))
}

func convertGoValueToLuaValue(value reflect.Value) (lua.LValue, error) {
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
		return lua.LNumber(value.Int()), nil
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
			item, err := convertGoValueToLuaValue(value.Index(i))
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
			item, err := convertGoValueToLuaValue(iter.Value())
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
func ConvertLuaValueToGoValue(value lua.LValue) (any, error) {
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
		return convertLuaTableToGo(value.(*lua.LTable))
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

func convertLuaTableToGo(table *lua.LTable) (any, error) {
	entries := make([]luaTableEntry, 0, table.Len())
	table.ForEach(func(key, value lua.LValue) {
		entries = append(entries, luaTableEntry{key: key, value: value})
	})

	if isDenseLuaArray(entries) {
		array := make([]any, len(entries))
		for _, entry := range entries {
			index := int(lua.LVAsNumber(entry.key))
			converted, err := ConvertLuaValueToGoValue(entry.value)
			if err != nil {
				return nil, fmt.Errorf("convert array item %d: %w", index, err)
			}
			array[index-1] = converted
		}
		return array, nil
	}

	mapped := make(map[any]any, len(entries))
	for _, entry := range entries {
		key, err := ConvertLuaValueToGoValue(entry.key)
		if err != nil {
			return nil, fmt.Errorf("convert table key: %w", err)
		}
		value, err := ConvertLuaValueToGoValue(entry.value)
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
