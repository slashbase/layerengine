package layerengine

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/slashbase/layerengine/modules"
)

const (
	layerExecutionTimeout = 60 * time.Second
	layerMemoryLimit      = uint64(256 << 20) // 256 MiB, approximate
)

var (
	errLayerTimeout = errors.New("layer execution timed out")
	errLayerMemory  = errors.New("layer exceeded memory limit")
)

func runLayer(layer *Layer, inputValues []any, mods []modules.Module) ([]any, error) {

	parent, cancelMem := context.WithCancelCause(context.Background())
	defer cancelMem(nil)

	ctx, cancelTimeout := context.WithTimeoutCause(parent, layerExecutionTimeout, errLayerTimeout)
	defer cancelTimeout()

	go guardMemory(ctx, cancelMem, layerMemoryLimit, errLayerMemory)

	layerRunner := NewLayerRunner(ctx, mods)
	defer layerRunner.Close()

	if err := layerRunner.LoadFunction(layer.FnProto); err != nil {
		return nil, wrapRunErr(layer, ctx, err)
	}

	luaInputs, err := ConvertGoValuesToLuaValues(inputValues)
	if err != nil {
		return nil, err
	}

	if err := layerRunner.RunFunction(layer.Name, luaInputs, len(layer.Output)); err != nil {
		return nil, wrapRunErr(layer, ctx, err)
	}

	luaOutput, err := layerRunner.ReadResult(len(layer.Output))
	if err != nil {
		return nil, err
	}

	result, err := ConvertLuaValuesToGoValues(luaOutput)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// checkLayerDefinesFunction executes layer's compiled chunk in a throwaway
// sandbox and reports whether it defines a global function named after the
// layer. The runner looks the function up by that name at call time, so a chunk
// that defines nothing — or defines it under a different name — would otherwise
// fail at RunFlow with a nil-call error far from the cause.
//
// The chunk's top-level statements run here as well as at run time; a generated
// layer is expected to contain nothing but the function definition.
func checkLayerDefinesFunction(layer *Layer, mods []modules.Module) error {
	parent, cancelMem := context.WithCancelCause(context.Background())
	defer cancelMem(nil)

	ctx, cancelTimeout := context.WithTimeoutCause(parent, layerExecutionTimeout, errLayerTimeout)
	defer cancelTimeout()

	go guardMemory(ctx, cancelMem, layerMemoryLimit, errLayerMemory)

	layerRunner := NewLayerRunner(ctx, mods)
	defer layerRunner.Close()

	if err := layerRunner.LoadFunction(layer.FnProto); err != nil {
		return wrapRunErr(layer, ctx, err)
	}
	if !layerRunner.HasGlobalFunction(layer.Name) {
		return fmt.Errorf(
			"layer %q: generated code does not define a global function %q (expected `function %s(...)`)",
			layer.Name, layer.Name, layer.Name,
		)
	}
	return nil
}

func runFlow(layers []*Layer, inputValues map[string]any, mods []modules.Module) (any, error) {
	var result []any
	for _, layer := range layers {
		layerInputValues := make([]any, len(layer.Input))
		for i, inpName := range layer.Input {
			layerInputValues[i] = inputValues[inpName]
		}

		layerResult, err := runLayer(layer, layerInputValues, mods)
		if err != nil {
			return nil, err
		}
		outputValues := layerResult
		for i, key := range layer.Output {
			inputValues[key.Name] = outputValues[i]
		}
		result = outputValues
	}

	return result, nil
}

func guardMemory(ctx context.Context, cancel context.CancelCauseFunc, limit uint64, cause error) {
	var base runtime.MemStats
	runtime.ReadMemStats(&base)
	t := time.NewTicker(50 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			var s runtime.MemStats
			runtime.ReadMemStats(&s)
			if s.HeapAlloc > base.HeapAlloc && s.HeapAlloc-base.HeapAlloc > limit {
				cancel(cause)
				return
			}
		}
	}
}

func wrapRunErr(layer *Layer, ctx context.Context, err error) error {
	switch context.Cause(ctx) {
	case errLayerMemory:
		return fmt.Errorf("layer %q: %w", layer.Name, errLayerMemory)
	case errLayerTimeout:
		return fmt.Errorf("layer %q: %w", layer.Name, errLayerTimeout)
	default:
		return err
	}
}
