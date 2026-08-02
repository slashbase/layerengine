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

func (le *LayerEngine) GenerateLayers(flow *validator.Flow) ([]Layer, error) {
	flowInputMap := map[string]FlowInput{}
	for _, fi := range flow.Input {
		flowInputMap[fi.Name] = FlowInput{fi}
	}

	flowOutputMap := map[string]codegen.Output{}

	layers := make([]Layer, 0, len(flow.Layers))

	for _, layer := range flow.Layers {
		inputs := make([]codegen.Input, len(layer.Input))
		for i, inpName := range layer.Input {

			if _, exists := flowInputMap[inpName]; exists {
				inputs[i] = codegen.Input{
					Name:        inpName,
					Type:        flowInputMap[inpName].Type,
					Description: flowInputMap[inpName].Description,
					Optional:    flowInputMap[inpName].Optional,
				}
			} else {
				inputs[i] = codegen.Input{
					Name:        inpName,
					Type:        flowOutputMap[inpName].Type,
					Description: flowOutputMap[inpName].Description,
					Optional:    false,
				}
			}

		}

		outputs := make([]codegen.Output, len(layer.Output))
		for i, out := range layer.Output {
			outputs[i] = codegen.Output{
				Name:        out.Name,
				Type:        out.Type,
				Description: out.Description,
			}
			flowOutputMap[out.Name] = outputs[i]
		}

		code, err := le.codegen.GenerateLayerFunction(
			layer.Name,
			layer.Description,
			inputs,
			outputs,
		)
		if err != nil {
			return nil, err
		}

		layers = append(layers, Layer{
			layer,
			nil,
			code,
		})
	}

	return layers, nil
}

func (le *LayerEngine) LoadSpec(spec string) error {
	flow, err := validator.Run([]byte(spec))
	if err != nil {
		return err
	}

	layers, err := le.GenerateLayers(flow)
	if err != nil {
		return err
	}

	layerNames := make([]string, len(layers))
	for i, layer := range layers {
		layerNames[i] = layer.Name
	}

	le.LoadLayers(layers)
	le.LoadFlow(map[string][]string{
		flow.Name: layerNames,
	})

	engineFlowInputs := make([]FlowInput, len(flow.Input))
	for i, fi := range flow.Input {
		engineFlowInputs[i] = FlowInput{fi}
	}
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
