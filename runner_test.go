package layerengine

import (
	"context"
	"testing"
	"time"
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
	runner := NewLayerRunner(ctx)
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
