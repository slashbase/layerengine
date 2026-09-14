package layerengine

import (
	"strings"
	"testing"
)

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
