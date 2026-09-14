package layerengine

import (
	"context"
	"time"
)

const layerExecutionTimeout = 15 * time.Second

func runLayer(layer *Layer, inputValues []any) ([]any, error) {

	ctx, cancel := context.WithTimeout(context.Background(), layerExecutionTimeout)
	defer cancel()

	layerRunner := NewLayerRunner(ctx)
	defer layerRunner.Close()

	if err := layerRunner.LoadFunction(layer.FnProto); err != nil {
		return nil, err
	}

	luaInputs, err := ConvertGoValuesToLuaValues(inputValues)
	if err != nil {
		return nil, err
	}

	if err := layerRunner.RunFunction(layer.Name, luaInputs, len(layer.Output)); err != nil {
		return nil, err
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

func runFlow(layers []*Layer, inputValues map[string]any) (any, error) {
	var result []any
	for _, layer := range layers {
		layerInputValues := make([]any, len(layer.Input))
		for i, inpName := range layer.Input {
			layerInputValues[i] = inputValues[inpName]
		}

		layerResult, err := runLayer(layer, layerInputValues)
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
