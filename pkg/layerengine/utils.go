package layerengine

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

func ConvertGoValueToLuaValue(input interface{}) lua.LValue {
	switch val := input.(type) {
	case nil:
		return lua.LNil
	case bool:
		return lua.LBool(val)
	case int:
		return lua.LNumber(val)
	case float64:
		return lua.LNumber(val)
	case string:
		return lua.LString(val)
	// Add more cases as needed for other types
	default:
		panic(fmt.Sprintf("Unsupported type: %T", val))
	}
}

func ConvertLuaValueToGoValue(val lua.LValue) interface{} {
	switch val.Type() {
	case lua.LTNil:
		return nil
	case lua.LTBool:
		return lua.LVAsBool(val)
	case lua.LTNumber:
		return lua.LVAsNumber(val)
	case lua.LTString:
		return lua.LVAsString(val)
	case lua.LTTable:
		return convertLuaTableToGo(val.(*lua.LTable))
	// Add cases for other Lua types as needed
	default:
		return val
	}
}

func ConvertGoValuesToLuaValues(inputs []interface{}) []lua.LValue {
	outputs := make([]lua.LValue, len(inputs))
	for i, v := range inputs {
		outputs[i] = ConvertGoValueToLuaValue(v)
	}
	return outputs
}

func ConvertLuaValuesToGoValues(inputs []lua.LValue) []interface{} {
	outputs := make([]interface{}, len(inputs))
	for i, v := range inputs {
		outputs[i] = ConvertLuaValueToGoValue(v)
	}
	return outputs
}

func convertLuaTableToGo(tbl *lua.LTable) map[interface{}]interface{} {
	goTable := make(map[interface{}]interface{})

	tbl.ForEach(func(key, value lua.LValue) {
		goKey := key.String()                      // Convert Lua key to string for simplicity
		goValue := ConvertLuaValueToGoValue(value) // Recursive call for nested values
		goTable[goKey] = goValue
	})

	return goTable
}
