package layerengine

import (
	"strings"
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
		Name: "chained",
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
	if inputs["incremented"] != lua.LNumber(4) || inputs["result"] != lua.LNumber(8) {
		t.Fatalf("RunFlow did not retain layer outputs in the shared values map: %#v", inputs)
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
