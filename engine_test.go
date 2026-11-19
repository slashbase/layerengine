package layerengine

import (
	"errors"
	"strings"
	"sync"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestLoadSpecReturnsErrorWithoutCodeGenerator(t *testing.T) {
	for _, engine := range []*LayerEngine{NewBlankLayerEngine(), NewLayerEngine(nil)} {
		err := engine.LoadSpec("name: flow\n")
		if err == nil || !strings.Contains(err.Error(), "code generator is not configured") {
			t.Fatalf("LoadSpec error = %v, want unconfigured-generator error", err)
		}
	}
}

func TestRunLayerReturnsErrorForUnknownName(t *testing.T) {
	engine := NewBlankLayerEngine()

	_, err := engine.RunLayer("missing", nil)
	if err == nil || !strings.Contains(err.Error(), `layer "missing" not found`) {
		t.Fatalf("RunLayer error = %v, want missing-layer error", err)
	}
}

func TestRunFlowReturnsErrorForUnknownName(t *testing.T) {
	engine := NewBlankLayerEngine()

	_, err := engine.RunFlow("missing", nil)
	if err == nil || !strings.Contains(err.Error(), `flow "missing" not found`) {
		t.Fatalf("RunFlow error = %v, want missing-flow error", err)
	}
}

func TestRunFlowPassesLayerOutputsToFollowingLayers(t *testing.T) {
	engine := NewBlankLayerEngine()
	flow := Flow{
		Name:  "chained",
		Input: []FlowInput{{Name: "start", Type: "number"}},
		Layers: []Layer{
			{
				Name:   "increment",
				Input:  []string{"start"},
				Output: []LayerOutput{{Name: "incremented"}},
				Code:   "function increment(start) return start + 1 end",
			},
			{
				Name:   "double",
				Input:  []string{"incremented"},
				Output: []LayerOutput{{Name: "result"}},
				Code:   "function double(incremented) return incremented * 2 end",
			},
		},
	}
	if err := engine.LoadLayers(flow.Layers); err != nil {
		t.Fatalf("LoadLayers: %v", err)
	}
	if err := engine.LoadFlow(flow); err != nil {
		t.Fatalf("LoadFlow: %v", err)
	}

	inputs := map[string]any{"start": 3}
	output, err := engine.RunFlow(flow.Name, inputs)
	if err != nil {
		t.Fatalf("RunFlow: %v", err)
	}

	results, ok := output.([]any)
	if !ok || len(results) != 1 || results[0] != lua.LNumber(8) {
		t.Fatalf("RunFlow output = %#v, want []any{8}", output)
	}
	if len(inputs) != 1 || inputs["start"] != 3 {
		t.Fatalf("RunFlow mutated the caller input map: %#v", inputs)
	}
}

func TestLayerEngineAllowsConcurrentRunsAndLoads(t *testing.T) {
	engine := NewBlankLayerEngine()
	flow := Flow{
		Name:  "double",
		Input: []FlowInput{{Name: "value", Type: "number"}},
		Layers: []Layer{{
			Name:   "double",
			Input:  []string{"value"},
			Output: []LayerOutput{{Name: "result"}},
			Code:   "function double(value) return value * 2 end",
		}},
	}
	if err := engine.LoadLayers(flow.Layers); err != nil {
		t.Fatalf("LoadLayers: %v", err)
	}
	if err := engine.LoadFlow(flow); err != nil {
		t.Fatalf("LoadFlow: %v", err)
	}

	start := make(chan struct{})
	errs := make(chan error, 100)
	var group sync.WaitGroup
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			for range 20 {
				output, err := engine.RunFlow(flow.Name, map[string]any{"value": 4})
				if err != nil {
					errs <- err
					return
				}
				values := output.([]any)
				if len(values) != 1 || values[0] != lua.LNumber(8) {
					errs <- errors.New("unexpected flow output")
					return
				}
			}
		}()
	}
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			for range 20 {
				if err := engine.LoadLayers(flow.Layers); err != nil {
					errs <- err
					return
				}
				if err := engine.LoadFlow(flow); err != nil {
					errs <- err
					return
				}
			}
		}()
	}

	close(start)
	group.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

func TestLoadLayersDoesNotCommitPartialResults(t *testing.T) {
	engine := NewBlankLayerEngine()

	err := engine.LoadLayers([]Layer{
		{Name: "valid", Code: "function valid() return 1 end"},
		{Name: "invalid", Code: "function invalid("},
	})
	if err == nil || !strings.Contains(err.Error(), `compile layer "invalid"`) {
		t.Fatalf("LoadLayers error = %v, want invalid-layer compile error", err)
	}
	if _, loaded := engine.layers["valid"]; loaded {
		t.Fatal("LoadLayers committed a layer despite a later compile error")
	}
}

func TestLoadLayersRejectsDuplicateNamesWithoutCommit(t *testing.T) {
	engine := NewBlankLayerEngine()

	err := engine.LoadLayers([]Layer{
		{Name: "duplicate", Code: "function duplicate() return 1 end"},
		{Name: "duplicate", Code: "function duplicate() return 2 end"},
	})
	if err == nil || !strings.Contains(err.Error(), `duplicate layer name "duplicate"`) {
		t.Fatalf("LoadLayers error = %v, want duplicate-name error", err)
	}
	if _, loaded := engine.layers["duplicate"]; loaded {
		t.Fatal("LoadLayers committed layers despite duplicate names")
	}
}

func TestLoadFlowReturnsErrorForMissingLayer(t *testing.T) {
	engine := NewBlankLayerEngine()

	err := engine.LoadFlow(Flow{
		Name:   "broken",
		Layers: []Layer{{Name: "missing"}},
	})
	if err == nil || !strings.Contains(err.Error(), `layer "missing" not found`) {
		t.Fatalf("LoadFlow error = %v, want missing-layer error", err)
	}
	if _, loaded := engine.flows["broken"]; loaded {
		t.Fatal("LoadFlow registered a flow with a missing layer")
	}
}

func TestLoadLayersRejectsCodeThatDoesNotDefineTheLayerFunction(t *testing.T) {
	engine := NewBlankLayerEngine()

	err := engine.LoadLayers([]Layer{
		{Name: "add_numbers", Code: "function addNumbers(a, b) return a + b end"},
	})
	if err == nil || !strings.Contains(err.Error(), `layer "add_numbers": generated code does not define a global function "add_numbers"`) {
		t.Fatalf("LoadLayers error = %v, want missing-function error", err)
	}
	if _, loaded := engine.layers["add_numbers"]; loaded {
		t.Fatal("LoadLayers committed a layer whose code does not define it")
	}
}

func TestLoadLayersRejectsCodeThatDefinesNoFunction(t *testing.T) {
	engine := NewBlankLayerEngine()

	err := engine.LoadLayers([]Layer{{Name: "noop", Code: "local x = 1"}})
	if err == nil || !strings.Contains(err.Error(), "does not define a global function") {
		t.Fatalf("LoadLayers error = %v, want missing-function error", err)
	}
	if _, loaded := engine.layers["noop"]; loaded {
		t.Fatal("LoadLayers committed a layer whose code defines no function")
	}
}

func TestLoadLayersReportsChunkErrorAtLoadTime(t *testing.T) {
	engine := NewBlankLayerEngine()

	err := engine.LoadLayers([]Layer{{Name: "boom", Code: `error("kaboom")`}})
	if err == nil || !strings.Contains(err.Error(), "kaboom") {
		t.Fatalf("LoadLayers error = %v, want the chunk's own error", err)
	}
}

func TestBlankEngineLoadsBuiltInModules(t *testing.T) {
	engine := NewBlankLayerEngine()

	mods := engine.GetModules()
	if len(mods) == 0 {
		t.Fatal("NewBlankLayerEngine has no modules, want the built-in set")
	}

	// The sandbox a blank engine gives a hand-loaded layer must be the one the
	// codegen prompt describes, so system.version() has to resolve.
	if err := engine.LoadLayers([]Layer{{
		Name:   "version",
		Output: []LayerOutput{{Name: "out"}},
		Code:   "function version() return system.version() end",
	}}); err != nil {
		t.Fatalf("LoadLayers: %v", err)
	}
	output, err := engine.RunLayer("version", nil)
	if err != nil {
		t.Fatalf("RunLayer: %v", err)
	}
	values, ok := output.([]any)
	if !ok || len(values) != 1 {
		t.Fatalf("RunLayer output = %#v, want one value", output)
	}
}

func TestGetModulesReturnsACopy(t *testing.T) {
	engine := NewBlankLayerEngine()

	first := engine.GetModules()
	if len(first) == 0 {
		t.Fatal("GetModules returned no modules")
	}
	first[0] = nil

	second := engine.GetModules()
	if len(second) == 0 || second[0] == nil {
		t.Fatal("mutating the slice returned by GetModules changed the engine's module set")
	}
}

func TestCommitRejectsLayerNameOwnedByAnotherFlow(t *testing.T) {
	engine := NewBlankLayerEngine()

	firstCode := "function format(value) return tostring(value) end"
	if err := engine.commitLayers("flow_a", map[string]*Layer{
		"format": {Name: "format", Code: firstCode},
	}, nil); err != nil {
		t.Fatalf("commitLayers flow_a: %v", err)
	}

	err := engine.commitLayers("flow_b", map[string]*Layer{
		"format": {Name: "format", Code: "function format(value) return 'x' .. value end"},
	}, nil)
	if err == nil {
		t.Fatal("commitLayers flow_b succeeded, want a collision error")
	}
	if !strings.Contains(err.Error(), `"format"`) || !strings.Contains(err.Error(), `"flow_a"`) {
		t.Fatalf("collision error = %v, want it to name the layer and the owning flow", err)
	}

	layer, ok := engine.Layer("format")
	if !ok || layer.Code != firstCode {
		t.Fatalf("Layer(format) = %#v, want the code flow_a loaded", layer)
	}
}

func TestCommitSharesIdenticalCodeBetweenFlows(t *testing.T) {
	engine := NewBlankLayerEngine()
	code := "function format(value) return tostring(value) end"

	if err := engine.commitLayers("flow_a", map[string]*Layer{
		"format": {Name: "format", Code: code},
	}, nil); err != nil {
		t.Fatalf("commitLayers flow_a: %v", err)
	}
	if err := engine.commitLayers("flow_b", map[string]*Layer{
		"format": {Name: "format", Code: code},
	}, nil); err != nil {
		t.Fatalf("commitLayers flow_b with identical code: %v", err)
	}
}

func TestLoadLayersMayReplaceAnUnownedLayer(t *testing.T) {
	engine := NewBlankLayerEngine()

	if err := engine.LoadLayers([]Layer{{Name: "format", Code: "function format() return 1 end"}}); err != nil {
		t.Fatalf("LoadLayers: %v", err)
	}
	if err := engine.LoadLayers([]Layer{{Name: "format", Code: "function format() return 2 end"}}); err != nil {
		t.Fatalf("LoadLayers replacing an unowned layer: %v", err)
	}
}

func TestCommitLeavesEngineUnchangedWhenFlowRegistrationFails(t *testing.T) {
	engine := NewBlankLayerEngine()

	compiled := map[string]*Layer{
		"increment": {Name: "increment", Code: "function increment(value) return value + 1 end"},
	}
	flow := &Flow{
		Name: "broken",
		Layers: []Layer{
			{Name: "increment", Code: compiled["increment"].Code},
			{Name: "missing", Code: "function missing(value) return value end"},
		},
	}

	if err := engine.commitLayers("broken", compiled, flow); err == nil {
		t.Fatal("commitLayers succeeded, want a not-found error for layer \"missing\"")
	}
	if _, loaded := engine.Layer("increment"); loaded {
		t.Fatal("commitLayers left layer \"increment\" loaded after the flow failed to register")
	}
	if _, err := engine.RunFlow("broken", nil); err == nil {
		t.Fatal("commitLayers registered flow \"broken\" after failing")
	}
}

func TestLayerInputsRejectsUnknownName(t *testing.T) {
	flow := &Flow{
		Name:  "unknown_input",
		Input: []FlowInput{{Name: "start", Type: "number", Description: "start value"}},
		Layers: []Layer{{
			Name:  "consume",
			Input: []string{"nowhere"},
		}},
	}

	err := layerInputsError(flow, 0)
	if err == nil || !strings.Contains(err.Error(), `input "nowhere"`) {
		t.Fatalf("layerInputs error = %v, want an unknown-input error naming it", err)
	}
}

func TestLayerInputsMapsFlowInputsAndEarlierOutputs(t *testing.T) {
	flow := &Flow{
		Name: "mapped",
		Input: []FlowInput{
			{Name: "start", Type: "number", Description: "start value", Optional: true},
		},
		Layers: []Layer{
			{Name: "first", Output: []LayerOutput{{Name: "sum", Type: "number", Description: "the sum"}}},
			{Name: "second", Input: []string{"start", "sum"}},
		},
	}

	inputs, err := layerInputs(flow, 1)
	if err != nil {
		t.Fatalf("layerInputs: %v", err)
	}
	if len(inputs) != 2 {
		t.Fatalf("layerInputs returned %d inputs, want 2", len(inputs))
	}
	if inputs[0].Name != "start" || inputs[0].Type != "number" || inputs[0].Description != "start value" || !inputs[0].Optional {
		t.Fatalf("flow input mapped to %#v", inputs[0])
	}
	if inputs[1].Name != "sum" || inputs[1].Type != "number" || inputs[1].Description != "the sum" {
		t.Fatalf("earlier output mapped to %#v", inputs[1])
	}
	if inputs[1].Optional {
		t.Fatal("an input produced by an earlier layer must not be optional")
	}
}

// layerInputsError is layerInputs with the value discarded, so a test can assert
// on the error without naming an unused slice.
func layerInputsError(flow *Flow, index int) error {
	_, err := layerInputs(flow, index)
	return err
}

func TestFlowReturnsTheLoadedDefinition(t *testing.T) {
	engine := NewBlankLayerEngine()
	flow := Flow{
		Name:  "echo",
		Input: []FlowInput{{Name: "value", Type: "string", Description: "a value"}},
		Layers: []Layer{{
			Name:   "echo",
			Input:  []string{"value"},
			Output: []LayerOutput{{Name: "result", Type: "string"}},
			Code:   "function echo(value) return value end",
		}},
	}
	if err := engine.LoadLayers(flow.Layers); err != nil {
		t.Fatalf("LoadLayers: %v", err)
	}
	if err := engine.LoadFlow(flow); err != nil {
		t.Fatalf("LoadFlow: %v", err)
	}

	loaded, ok := engine.Flow("echo")
	if !ok {
		t.Fatal("Flow did not find the loaded flow")
	}
	if loaded.Name != "echo" || len(loaded.Input) != 1 || loaded.Input[0].Name != "value" {
		t.Fatalf("Flow returned %#v", loaded)
	}
	if len(loaded.Layers) != 1 || len(loaded.Layers[0].Output) != 1 || loaded.Layers[0].Output[0].Name != "result" {
		t.Fatalf("Flow layers = %#v, want the declared output", loaded.Layers)
	}

	loaded.Name = "mutated"
	if again, _ := engine.Flow("echo"); again.Name != "echo" {
		t.Fatal("mutating the returned flow changed the engine's copy")
	}

	if _, ok := engine.Flow("missing"); ok {
		t.Fatal("Flow found a flow that was never loaded")
	}
}

func TestRunFlowResultKeysOutputsByName(t *testing.T) {
	engine := NewBlankLayerEngine()
	flow := Flow{
		Name:  "shout",
		Input: []FlowInput{{Name: "text", Type: "string"}},
		Layers: []Layer{
			{
				Name:   "copy",
				Input:  []string{"text"},
				Output: []LayerOutput{{Name: "copied", Type: "string"}},
				Code:   "function copy(text) return text end",
			},
			{
				Name:   "upper",
				Input:  []string{"copied"},
				Output: []LayerOutput{{Name: "loud", Type: "string"}},
				Code:   "function upper(copied) return string.upper(copied) end",
			},
		},
	}
	if err := engine.LoadLayers(flow.Layers); err != nil {
		t.Fatalf("LoadLayers: %v", err)
	}
	if err := engine.LoadFlow(flow); err != nil {
		t.Fatalf("LoadFlow: %v", err)
	}

	result, err := engine.RunFlowResult("shout", map[string]any{"text": "hi"})
	if err != nil {
		t.Fatalf("RunFlowResult: %v", err)
	}
	if result.Layer != "upper" {
		t.Fatalf("result.Layer = %q, want the last layer", result.Layer)
	}
	if len(result.Outputs) != 1 {
		t.Fatalf("result.Outputs = %#v, want only the last layer's outputs", result.Outputs)
	}
	if got := result.Outputs["loud"]; got != "HI" {
		t.Fatalf("result.Outputs[loud] = %#v, want \"HI\"", got)
	}

	if _, err := engine.RunFlowResult("missing", nil); err == nil {
		t.Fatal("RunFlowResult ran a flow that was never loaded")
	}
}
