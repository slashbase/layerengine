package layerengine

import (
	"context"
	"testing"
	"time"

	"github.com/slashbase/layerengine/modules"
)

func TestLayerRunnerStopsWhenContextExpires(t *testing.T) {
	proto, err := ParseAndCompileLuaCode(`
function never_returns()
  while true do end
end
`)
	if err != nil {
		t.Fatalf("compile Lua: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	runner := NewLayerRunner(ctx, []modules.Module{})
	defer runner.Close()

	if err := runner.LoadFunction(proto); err != nil {
		t.Fatalf("load Lua function: %v", err)
	}

	started := time.Now()
	err = runner.RunFunction("never_returns", nil, 0)
	if err == nil {
		t.Fatal("RunFunction succeeded for an infinite loop")
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("RunFunction took %s after its context expired", elapsed)
	}
}

func TestOsExitCannotKillHost(t *testing.T) {
	proto, err := ParseAndCompileLuaCode(`
function exitos()
  os.exit()
end
`)
	if err != nil {
		t.Fatalf("compile Lua: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	runner := NewLayerRunner(ctx, []modules.Module{})
	defer runner.Close()

	if err := runner.LoadFunction(proto); err != nil {
		t.Fatalf("load Lua function: %v", err)
	}

	err = runner.RunFunction("exitos", nil, 0)
	if err == nil {
		t.Fatal("failed")
	}
}

// TestSandboxHidesHostCapabilities asserts the layer environment exposes none
// of the host-capable globals. Generated code must go through layer.* for
// anything beyond pure computation.
func TestSandboxHidesHostCapabilities(t *testing.T) {
	proto, err := ParseAndCompileLuaCode(`
function probe()
  return tostring(os) .. "," .. tostring(io) .. "," .. tostring(debug) ..
         "," .. tostring(package) .. "," .. tostring(loadstring) .. "," .. tostring(getfenv)
end
`)
	if err != nil {
		t.Fatalf("compile probe: %v", err)
	}

	layer := &Layer{
		Name:    "probe",
		Output:  []LayerOutput{{Name: "out", Type: "string"}},
		FnProto: proto,
	}

	res, err := runLayer(layer, nil, []modules.Module{})
	if err != nil {
		t.Fatalf("runLayer: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 output, got %d", len(res))
	}

	got, _ := res[0].(string)
	const want = "nil,nil,nil,nil,nil,nil"
	if got != want {
		t.Fatalf("host capabilities are reachable: got %q, want %q", got, want)
	}
}
