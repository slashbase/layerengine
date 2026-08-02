package layerengine

import (
	"fmt"

	"github.com/slashbase/layerengine/codegen"
	"github.com/slashbase/layerengine/validator"
)

type LayerEngine struct {
	layers     map[string]*Layer
	flows      map[string][]*Layer
	flowInputs map[string][]FlowInput
	codegen    *codegen.CodeGen
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
	le.flowInputs = map[string][]FlowInput{}
}

func (le *LayerEngine) GenerateLayers(flow *Flow) error {
	flowInputMap := map[string]FlowInput{}
	for _, fi := range flow.Input {
		flowInputMap[fi.Name] = fi
	}

	flowOutputMap := map[string]codegen.Output{}

	for i, layer := range flow.Layers {
		inputs := make([]codegen.Input, len(layer.Input))
		for j, inpName := range layer.Input {
			if fi, exists := flowInputMap[inpName]; exists {
				codegen.MapToStruct(StructToMap(fi), &inputs[j])
			} else {
				codegen.MapToStruct(StructToMap(flowOutputMap[inpName]), &inputs[j])
				inputs[j].Optional = false
			}
		}

		outputs := make([]codegen.Output, len(layer.Output))
		for j, out := range layer.Output {
			codegen.MapToStruct(StructToMap(out), &outputs[j])
			flowOutputMap[out.Name] = outputs[j]
		}

		code, err := le.codegen.GenerateLayerFunction(
			layer.Name,
			layer.Description,
			inputs,
			outputs,
		)
		if err != nil {
			return err
		}

		flow.Layers[i].Code = code
	}

	return nil
}

func (le *LayerEngine) LoadSpec(spec string) error {
	f, err := validator.Run([]byte(spec))
	if err != nil {
		return err
	}

	var flow Flow
	MapToStruct(validator.StructToMap(f), &flow)

	err = le.GenerateLayers(&flow)
	if err != nil {
		return err
	}

	layerNames := make([]string, len(flow.Layers))
	for i, layer := range flow.Layers {
		layerNames[i] = layer.Name
	}

	le.LoadLayers(flow.Layers)
	le.LoadFlow(map[string][]string{
		flow.Name: layerNames,
	})

	engineFlowInputs := make([]FlowInput, len(flow.Input))
	copy(engineFlowInputs, flow.Input)
	le.flowInputs[flow.Name] = engineFlowInputs

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
	if flowInputs, ok := le.flowInputs[name]; ok {
		for _, fi := range flowInputs {
			_, provided := inputValues[fi.Name]
			if !provided && !fi.Optional {
				return nil, fmt.Errorf("required input %q not provided", fi.Name)
			}
		}
	}
	return runFlow(le.flows[name], inputValues)
}
