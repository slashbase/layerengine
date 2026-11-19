package layerengine

import (
	"context"
	"strings"
	"testing"

	"github.com/slashbase/layerengine/modules"
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

	_, err = runLayer(context.Background(), &Layer{Name: "noop", FnProto: proto}, []any{struct{}{}}, []modules.Module{})
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

func TestConvertLuaValueToGoValueRejectsCyclicTable(t *testing.T) {
	table := &lua.LTable{}
	table.RawSetString("self", table)

	_, err := ConvertLuaValueToGoValue(table)
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("error = %v, want a cycle error", err)
	}
}

func TestConvertLuaValueToGoValueRejectsOverlyDeepTable(t *testing.T) {
	root := &lua.LTable{}
	current := root
	for range maxValueDepth + 5 {
		next := &lua.LTable{}
		current.RawSetInt(1, next)
		current = next
	}

	_, err := ConvertLuaValueToGoValue(root)
	if err == nil || !strings.Contains(err.Error(), "nested deeper") {
		t.Fatalf("error = %v, want a depth error", err)
	}
}

// A table reached twice by different branches is a DAG, not a cycle. The
// conversion has to accept it, or shared structure in generated output would be
// rejected as if it were malformed.
func TestConvertLuaValueToGoValueAllowsTableSharedBySiblings(t *testing.T) {
	shared := &lua.LTable{}
	shared.RawSetString("name", lua.LString("shared"))

	root := &lua.LTable{}
	root.RawSetString("left", shared)
	root.RawSetString("right", shared)

	value, err := ConvertLuaValueToGoValue(root)
	if err != nil {
		t.Fatalf("converting a DAG: %v", err)
	}

	// The root's Go type depends on the key-shape policy, which this test is not
	// about: it only has to show the shared table was not mistaken for a cycle.
	var entries int
	switch mapped := value.(type) {
	case map[string]any:
		entries = len(mapped)
	case map[any]any:
		entries = len(mapped)
	default:
		t.Fatalf("root converted to %T, want a map", value)
	}
	if entries != 2 {
		t.Fatalf("root has %d entries, want 2", entries)
	}
}

func TestConvertGoValueToLuaValueRejectsInexactInteger(t *testing.T) {
	_, err := ConvertGoValueToLuaValue(int64(1<<53) + 1)
	if err == nil || !strings.Contains(err.Error(), "cannot be represented exactly") {
		t.Fatalf("error = %v, want an inexact-integer error", err)
	}

	if _, err := ConvertGoValueToLuaValue(int64(1 << 53)); err != nil {
		t.Fatalf("2^53 must convert exactly, got error: %v", err)
	}
}

func TestConvertLuaValueToGoValueReturnsStringKeyedMap(t *testing.T) {
	table := &lua.LTable{}
	table.RawSetString("city", lua.LString("Pune"))
	table.RawSetString("count", lua.LNumber(2))

	value, err := ConvertLuaValueToGoValue(table)
	if err != nil {
		t.Fatalf("convert string-keyed table: %v", err)
	}
	mapped, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("value type = %T, want map[string]any", value)
	}
	if mapped["city"] != "Pune" || mapped["count"] != lua.LNumber(2) {
		t.Fatalf("value = %#v, want the table's entries", mapped)
	}
}

func TestConvertLuaValueToGoValueReturnsEmptyMapForEmptyTable(t *testing.T) {
	value, err := ConvertLuaValueToGoValue(&lua.LTable{})
	if err != nil {
		t.Fatalf("convert empty table: %v", err)
	}
	mapped, ok := value.(map[string]any)
	if !ok || len(mapped) != 0 {
		t.Fatalf("value = %#v, want an empty map[string]any", value)
	}
}

func TestConvertGoValueToLuaValueRejectsSelfReferentialMap(t *testing.T) {
	cycle := map[string]any{}
	cycle["self"] = cycle

	_, err := ConvertGoValueToLuaValue(cycle)
	if err == nil || !strings.Contains(err.Error(), "nested deeper") {
		t.Fatalf("error = %v, want a depth error", err)
	}
}
