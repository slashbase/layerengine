package layerengine

import lua "github.com/yuin/gopher-lua"

type Layer struct {
	Name      string
	FnProto   *lua.FunctionProto
	OutputLen int
}

func runLayer(layer Layer, inputValues []interface{}) (interface{}, error) {

	layerRunner := NewLayerRunner()
	defer layerRunner.Close()

	if err := layerRunner.LoadFunction(layer.FnProto); err != nil {
		return nil, err
	}

	luaInputs := ConvertGoValuesToLuaValues(inputValues)

	if err := layerRunner.RunFunction(layer.Name, luaInputs, layer.OutputLen); err != nil {
		return nil, err
	}

	var err error
	luaOutput, err := layerRunner.ReadResult(layer.OutputLen)
	if err != nil {
		return nil, err
	}

	result := ConvertLuaValuesToGoValues(luaOutput)

	return result, nil
}

func runFlow(layers []Layer, inputValues []interface{}) (interface{}, error) {

	var luaOutput []lua.LValue
	for i, layer := range layers {
		layerRunner := NewLayerRunner()
		defer layerRunner.Close()

		if err := layerRunner.LoadFunction(layer.FnProto); err != nil {
			return nil, err
		}

		luaInputs := ConvertGoValuesToLuaValues(inputValues)
		if i != 0 {
			luaInputs = luaOutput
		}

		if err := layerRunner.RunFunction(layer.Name, luaInputs, layer.OutputLen); err != nil {
			return nil, err
		}

		var err error
		luaOutput, err = layerRunner.ReadResult(layer.OutputLen)
		if err != nil {
			return nil, err
		}
	}

	result := ConvertLuaValuesToGoValues(luaOutput)

	return result, nil
}
