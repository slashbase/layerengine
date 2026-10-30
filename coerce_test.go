package layerengine

import (
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func mustCoerce(t *testing.T, inputs []FlowInput, args map[string]any, opts CoerceOptions) map[string]any {
	t.Helper()
	values, err := CoerceInputs(inputs, args, opts)
	if err != nil {
		t.Fatalf("CoerceInputs: %v", err)
	}
	return values
}

func TestCoerceInputsConvertsNumericStringToNumber(t *testing.T) {
	values := mustCoerce(t, []FlowInput{{Name: "count", Type: "number"}}, map[string]any{"count": "5"}, CoerceOptions{})
	if got := values["count"]; got != 5.0 {
		t.Fatalf("count = %#v, want float64(5)", got)
	}
}

func TestCoerceInputsLeavesMatchingValuesAlone(t *testing.T) {
	values := mustCoerce(t, []FlowInput{
		{Name: "count", Type: "number"},
		{Name: "name", Type: "string"},
		{Name: "flag", Type: "boolean"},
	}, map[string]any{"count": 3, "name": "Ada", "flag": true}, CoerceOptions{})

	if got, ok := values["count"].(int); !ok || got != 3 {
		t.Errorf("count = %#v, want int(3) preserved", values["count"])
	}
	if values["name"] != "Ada" {
		t.Errorf("name = %#v, want \"Ada\"", values["name"])
	}
	if values["flag"] != true {
		t.Errorf("flag = %#v, want true", values["flag"])
	}
}

func TestCoerceInputsFormatsNumberAndBoolAsString(t *testing.T) {
	values := mustCoerce(t, []FlowInput{
		{Name: "label", Type: "string"},
		{Name: "other", Type: "string"},
	}, map[string]any{"label": 42.0, "other": false}, CoerceOptions{})

	if values["label"] != "42" {
		t.Errorf("label = %#v, want \"42\"", values["label"])
	}
	if values["other"] != "false" {
		t.Errorf("other = %#v, want \"false\"", values["other"])
	}
}

func TestCoerceInputsParsesBooleanString(t *testing.T) {
	values := mustCoerce(t, []FlowInput{{Name: "flag", Type: "boolean"}}, map[string]any{"flag": "TRUE"}, CoerceOptions{})
	if values["flag"] != true {
		t.Fatalf("flag = %#v, want true", values["flag"])
	}
}

func TestCoerceInputsAcceptsWholeNumberForInteger(t *testing.T) {
	values := mustCoerce(t, []FlowInput{{Name: "n", Type: "integer"}}, map[string]any{"n": "7"}, CoerceOptions{})
	if got := values["n"]; got != 7.0 {
		t.Fatalf("n = %#v, want float64(7)", got)
	}
}

func TestCoerceInputsRejectsFractionalInteger(t *testing.T) {
	_, err := CoerceInputs([]FlowInput{{Name: "n", Type: "integer"}}, map[string]any{"n": 2.5}, CoerceOptions{})
	if err == nil || !strings.Contains(err.Error(), `input "n":`) || !strings.Contains(err.Error(), "not a whole number") {
		t.Fatalf("error = %v, want wrapped whole-number error", err)
	}
}

func TestCoerceInputsSkipsOmittedOptionalInput(t *testing.T) {
	values := mustCoerce(t, []FlowInput{
		{Name: "required", Type: "number"},
		{Name: "omitted", Type: "number", Optional: true},
		{Name: "nulled", Type: "number", Optional: true},
	}, map[string]any{"required": 1, "nulled": nil}, CoerceOptions{})

	if _, ok := values["omitted"]; ok {
		t.Error("omitted optional input is present in the result")
	}
	if _, ok := values["nulled"]; ok {
		t.Error("optional input supplied as nil is present in the result")
	}
	if len(values) != 1 {
		t.Fatalf("result = %#v, want only the required input", values)
	}
}

func TestCoerceInputsReportsMissingRequiredInput(t *testing.T) {
	_, err := CoerceInputs([]FlowInput{{Name: "count", Type: "number"}}, nil, CoerceOptions{})
	if err == nil || err.Error() != `required input "count" not provided` {
		t.Fatalf("error = %v, want the exact required-input message", err)
	}

	_, err = CoerceInputs([]FlowInput{{Name: "count", Type: "number"}}, map[string]any{"count": nil}, CoerceOptions{})
	if err == nil || err.Error() != `required input "count" not provided` {
		t.Fatalf("nil required input error = %v, want the exact required-input message", err)
	}
}

func TestCoerceInputsStrictRejectsConvertibleValue(t *testing.T) {
	_, err := CoerceInputs([]FlowInput{{Name: "count", Type: "number"}}, map[string]any{"count": "5"}, CoerceOptions{Strict: true})
	if err == nil || !strings.Contains(err.Error(), `input "count": cannot convert string to number`) {
		t.Fatalf("error = %v, want strict type-mismatch error", err)
	}

	// Strict only rejects values that would have been converted.
	values := mustCoerce(t, []FlowInput{{Name: "count", Type: "number"}}, map[string]any{"count": 5.0}, CoerceOptions{Strict: true})
	if values["count"] != 5.0 {
		t.Fatalf("count = %#v, want 5", values["count"])
	}
}

func TestCoerceInputsRejectsUnconvertibleValue(t *testing.T) {
	_, err := CoerceInputs([]FlowInput{{Name: "count", Type: "number"}}, map[string]any{"count": true}, CoerceOptions{})
	if err == nil || !strings.Contains(err.Error(), `input "count": cannot convert boolean to number`) {
		t.Fatalf("error = %v, want type-mismatch error naming the received kind", err)
	}
}

func TestCoerceInputsDropsUndeclaredArgumentsAndReportsThem(t *testing.T) {
	var reported []string
	values := mustCoerce(t, []FlowInput{{Name: "count", Type: "number"}}, map[string]any{
		"count":   1,
		"extra":   "x",
		"another": 2,
	}, CoerceOptions{OnUndeclared: func(name string) { reported = append(reported, name) }})

	if _, ok := values["extra"]; ok {
		t.Error("undeclared argument survived in the result")
	}
	if len(values) != 1 {
		t.Fatalf("result = %#v, want only declared inputs", values)
	}
	if got := strings.Join(reported, ","); got != "another,extra" {
		t.Fatalf("OnUndeclared reported %q, want \"another,extra\" in sorted order", got)
	}
}

func TestCoerceInputsKeepsUndeclaredArgumentsWhenAsked(t *testing.T) {
	reported := 0
	values := mustCoerce(t, []FlowInput{{Name: "count", Type: "number"}}, map[string]any{
		"count": 1,
		"extra": "x",
	}, CoerceOptions{KeepUndeclared: true, OnUndeclared: func(string) { reported++ }})

	if values["extra"] != "x" {
		t.Fatalf("extra = %#v, want \"x\" forwarded untouched", values["extra"])
	}
	if reported != 0 {
		t.Fatalf("OnUndeclared called %d times, want 0 when nothing is dropped", reported)
	}
}

func TestCoerceInputsPassesThroughUnrecognisedType(t *testing.T) {
	values := mustCoerce(t, []FlowInput{
		{Name: "items", Type: "array"},
		{Name: "typo", Type: "str"},
	}, map[string]any{"items": []any{"a"}, "typo": "5"}, CoerceOptions{})

	items, ok := values["items"].([]any)
	if !ok || len(items) != 1 || items[0] != "a" {
		t.Fatalf("items = %#v, want the []any passed through", values["items"])
	}
	if values["typo"] != "5" {
		t.Fatalf("typo = %#v, want \"5\" left unconverted", values["typo"])
	}
}

func TestCoerceInputsDoesNotMutateArguments(t *testing.T) {
	args := map[string]any{"count": "5", "extra": true}
	mustCoerce(t, []FlowInput{{Name: "count", Type: "number"}}, args, CoerceOptions{})

	if len(args) != 2 || args["count"] != "5" {
		t.Fatalf("args = %#v, want the caller's map untouched", args)
	}
}

func newIncrementEngine(t *testing.T) *LayerEngine {
	t.Helper()

	engine := NewBlankLayerEngine()
	flow := Flow{
		Name:  "increment",
		Input: []FlowInput{{Name: "start", Type: "number"}},
		Layers: []Layer{{
			Name:   "increment",
			Input:  []string{"start"},
			Output: []LayerOutput{{Name: "result", Type: "number"}},
			Code:   "function increment(start) return start + 1 end",
		}},
	}
	if err := engine.LoadLayers(flow.Layers); err != nil {
		t.Fatalf("LoadLayers: %v", err)
	}
	if err := engine.LoadFlow(flow); err != nil {
		t.Fatalf("LoadFlow: %v", err)
	}
	return engine
}

func TestRunFlowCoercesDeclaredInputTypes(t *testing.T) {
	engine := newIncrementEngine(t)

	output, err := engine.RunFlow("increment", map[string]any{"start": "5"})
	if err != nil {
		t.Fatalf("RunFlow: %v", err)
	}
	results, ok := output.([]any)
	if !ok || len(results) != 1 || results[0] != lua.LNumber(6) {
		t.Fatalf("RunFlow output = %#v, want []any{6}", output)
	}
}

func TestRunFlowWithOptionsStrictRejectsStringInput(t *testing.T) {
	engine := newIncrementEngine(t)

	_, err := engine.RunFlowWithOptions("increment", map[string]any{"start": "5"}, CoerceOptions{Strict: true})
	if err == nil || !strings.Contains(err.Error(), `input "start": cannot convert string to number`) {
		t.Fatalf("RunFlowWithOptions error = %v, want strict type-mismatch error", err)
	}
}

func TestRunFlowWithOptionsKeepUndeclaredForwardsUndeclaredInputs(t *testing.T) {
	// A flow loaded through LoadFlow can reference a layer input that is not a
	// declared flow input. Coercion drops it, so such a flow needs
	// KeepUndeclared — this test pins that behaviour rather than endorsing it.
	engine := NewBlankLayerEngine()
	flow := Flow{
		Name: "undeclared",
		Layers: []Layer{{
			Name:   "increment",
			Input:  []string{"start"},
			Output: []LayerOutput{{Name: "result", Type: "number"}},
			Code:   "function increment(start) return start + 1 end",
		}},
	}
	if err := engine.LoadLayers(flow.Layers); err != nil {
		t.Fatalf("LoadLayers: %v", err)
	}
	if err := engine.LoadFlow(flow); err != nil {
		t.Fatalf("LoadFlow: %v", err)
	}

	if _, err := engine.RunFlow(flow.Name, map[string]any{"start": 5}); err == nil {
		t.Fatal("RunFlow forwarded an undeclared argument")
	}

	output, err := engine.RunFlowWithOptions(flow.Name, map[string]any{"start": 5}, CoerceOptions{KeepUndeclared: true})
	if err != nil {
		t.Fatalf("RunFlowWithOptions: %v", err)
	}
	results, ok := output.([]any)
	if !ok || len(results) != 1 || results[0] != lua.LNumber(6) {
		t.Fatalf("RunFlowWithOptions output = %#v, want []any{6}", output)
	}
}
