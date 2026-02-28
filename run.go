package layerengine

import (
	lua "github.com/yuin/gopher-lua"
)

type Layer struct {
	Name    string
	FnProto *lua.FunctionProto
	Input   []string
	Output  []string
	Code    string
}

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
		for i, key := range layer.Input {
			layerInputValues[i] = inputValues[key]
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
			inputValues[key] = outputValues[i]
		}

	}

	result := ConvertLuaValuesToGoValues(luaOutput)

	return result, nil
}
