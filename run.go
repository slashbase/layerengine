package layerengine

import (
	lua "github.com/yuin/gopher-lua"
)

func runLayer(layer *Layer, inputValues []interface{}) (interface{}, error) {

	layerRunner := NewLayerRunner()
	defer layerRunner.Close()

	if err := layerRunner.LoadFunction(layer.FnProto); err != nil {
		return nil, err
	}

	luaInputs := ConvertGoValuesToLuaValues(inputValues)

	if err := layerRunner.RunFunction(layer.Name, luaInputs, len(layer.Output)); err != nil {
		return nil, err
	}

	var err error
	luaOutput, err := layerRunner.ReadResult(len(layer.Output))
	if err != nil {
		return nil, err
	}

	result := ConvertLuaValuesToGoValues(luaOutput)

	return result, nil
}

func runFlow(layers []*Layer, inputValues map[string]any) (interface{}, error) {

	var luaOutput []lua.LValue
	for _, layer := range layers {
		layerRunner := NewLayerRunner()
		defer layerRunner.Close()

		if err := layerRunner.LoadFunction(layer.FnProto); err != nil {
			return nil, err
		}

		layerInputValues := make([]any, len(layer.Input))
		for i, inpName := range layer.Input {
			layerInputValues[i] = inputValues[inpName]
		}

		luaInputs := ConvertGoValuesToLuaValues(layerInputValues)

		if err := layerRunner.RunFunction(layer.Name, luaInputs, len(layer.Output)); err != nil {
			return nil, err
		}

		var err error
		luaOutput, err = layerRunner.ReadResult(len(layer.Output))
		if err != nil {
			return nil, err
		}

		outputValues := ConvertLuaValuesToGoValues(luaOutput)
		for i, key := range layer.Output {
			inputValues[key.Name] = outputValues[i]
		}

	}

	result := ConvertLuaValuesToGoValues(luaOutput)

	return result, nil
}
