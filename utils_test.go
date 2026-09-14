package layerengine

import (
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestConvertGoValueToLuaValueSupportsMapsAndTypedSlices(t *testing.T) {
	value, err := ConvertGoValueToLuaValue(map[string]any{
		"names": []string{"Ada", "Lin"},
		"count": int32(2),
	})
	if err != nil {
		t.Fatalf("ConvertGoValueToLuaValue returned error: %v", err)
	}

	table, ok := value.(*lua.LTable)
	if !ok {
		t.Fatalf("value type = %T, want *lua.LTable", value)
	}
	if got := table.RawGetString("count"); got != lua.LNumber(2) {
		t.Errorf("count = %v, want 2", got)
	}
	names, ok := table.RawGetString("names").(*lua.LTable)
	if !ok || names.RawGetInt(1) != lua.LString("Ada") || names.RawGetInt(2) != lua.LString("Lin") {
		t.Errorf("names = %v, want Lua array of strings", names)
	}
}

func TestConvertGoValueToLuaValueReturnsErrorForUnsupportedValue(t *testing.T) {
	_, err := ConvertGoValueToLuaValue(struct{}{})
	if err == nil || !strings.Contains(err.Error(), "unsupported Go value type") {
		t.Fatalf("error = %v, want unsupported-type error", err)
	}
}

func TestRunLayerReturnsInputConversionError(t *testing.T) {
	proto, err := ParseAndCompileLuaCode("function noop(value) end")
	if err != nil {
		t.Fatalf("compile Lua: %v", err)
	}

	_, err = runLayer(&Layer{Name: "noop", FnProto: proto}, []any{struct{}{}})
	if err == nil || !strings.Contains(err.Error(), "unsupported Go value type") {
		t.Fatalf("runLayer error = %v, want unsupported-type error", err)
	}
}

func TestConvertLuaTableToGoHandlesDenseMixedAndSparseTables(t *testing.T) {
	dense := &lua.LTable{}
	dense.RawSetInt(1, lua.LString("first"))
	dense.RawSetInt(2, lua.LString("second"))
	denseValue, err := ConvertLuaValueToGoValue(dense)
	if err != nil {
		t.Fatalf("convert dense table: %v", err)
	}
	denseArray, ok := denseValue.([]any)
	if !ok || len(denseArray) != 2 || denseArray[0] != "first" || denseArray[1] != "second" {
		t.Errorf("dense table = %#v, want []any{first, second}", denseValue)
	}

	mixed := &lua.LTable{}
	mixed.RawSetInt(1, lua.LString("first"))
	mixed.RawSetString("label", lua.LString("mixed"))
	mixedValue, err := ConvertLuaValueToGoValue(mixed)
	if err != nil {
		t.Fatalf("convert mixed table: %v", err)
	}
	mixedMap, ok := mixedValue.(map[any]any)
	if !ok || mixedMap[lua.LNumber(1)] != "first" || mixedMap["label"] != "mixed" {
		t.Errorf("mixed table = %#v, want map preserving numeric and string keys", mixedValue)
	}

	sparse := &lua.LTable{}
	sparse.RawSetInt(3, lua.LString("third"))
	sparseValue, err := ConvertLuaValueToGoValue(sparse)
	if err != nil {
		t.Fatalf("convert sparse table: %v", err)
	}
	sparseMap, ok := sparseValue.(map[any]any)
	if !ok || sparseMap[lua.LNumber(3)] != "third" {
		t.Errorf("sparse table = %#v, want map preserving key 3", sparseValue)
	}
}
