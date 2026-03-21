package layerengine

import (
	"github.com/slashbase/layerengine/codegen"
	"github.com/slashbase/layerengine/validator"
)

type LayerEngine struct {
	layers  map[string]*Layer
	flows   map[string][]*Layer
	codegen *codegen.CodeGen
}

func NewLayerEngine(codegenertor *codegen.CodeGen) *LayerEngine {
	layerEngine := &LayerEngine{
		codegen: codegenertor,
	}
	layerEngine.init()
	return layerEngine
}

func NewBlankLayerEngine() *LayerEngine {
	layerEngine := &LayerEngine{}
	layerEngine.init()
	return layerEngine
}

func (le *LayerEngine) init() {
	le.layers = map[string]*Layer{}
	le.flows = map[string][]*Layer{}
}

func (le *LayerEngine) LoadSpec(spec string) error {
	flow, err := validator.Run([]byte(spec))
	if err != nil {
		return err
	}

	layers := []Layer{}
	layerNames := []string{}
	for _, layer := range flow.Layers {
		code, err := le.codegen.GenerateLayerFunction(layer.Name, layer.Description, layer.Input, layer.Output)
		if err != nil {
			return err
		}

		layerInstance := Layer{
			Name:    layer.Name,
			FnProto: nil,
			Input:   layer.Input,
			Output:  layer.Output,
			Code:    code,
		}

		layers = append(layers, layerInstance)
		layerNames = append(layerNames, layer.Name)
	}

	le.LoadLayers(layers)
	le.LoadFlow(map[string][]string{flow.Name: layerNames})
	return nil
}

func (le *LayerEngine) LoadLayers(layers []Layer) {
	for i := range layers {
		layer := layers[i]
		if fnProto, err := ParseAndCompileLuaCode(layer.Code); err == nil {
			layer.FnProto = fnProto
			le.layers[layer.Name] = &layer
		}
	}
}

func (le *LayerEngine) LoadFlow(flows map[string][]string) {
	for name, layerNames := range flows {
		layers := []*Layer{}
		for _, lname := range layerNames {
			layers = append(layers, le.layers[lname])
		}
		le.flows[name] = layers
	}
}

func (le *LayerEngine) RunLayer(name string, inputValues []any) (any, error) {
	return runLayer(le.layers[name], inputValues)
}

func (le *LayerEngine) RunFlow(name string, inputValues map[string]any) (any, error) {
	return runFlow(le.flows[name], inputValues)
}
