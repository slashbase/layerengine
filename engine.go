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
	if le.codegen == nil {
		return fmt.Errorf("code generator is not configured")
	}

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

	if err := le.LoadLayers(flow.Layers); err != nil {
		return err
	}
	if err := le.LoadFlow(flow); err != nil {
		return err
	}

	return nil
}

// LoadLayers compiles and loads all layers. It leaves the engine unchanged if
// any layer cannot be compiled.
func (le *LayerEngine) LoadLayers(layers []Layer) error {
	compiledLayers := make(map[string]*Layer, len(layers))
	for i := range layers {
		layer := layers[i]
		fnProto, err := ParseAndCompileLuaCode(layer.Code)
		if err != nil {
			return fmt.Errorf("compile layer %q: %w", layer.Name, err)
		}
		layer.FnProto = fnProto
		compiledLayers[layer.Name] = &layer
	}

	for name, layer := range compiledLayers {
		le.layers[name] = layer
	}
	return nil
}

// LoadFlow registers a flow after confirming that every referenced layer was
// loaded successfully.
func (le *LayerEngine) LoadFlow(flow Flow) error {
	layers := make([]*Layer, 0, len(flow.Layers))
	for _, layer := range flow.Layers {
		loadedLayer, ok := le.layers[layer.Name]
		if !ok || loadedLayer == nil {
			return fmt.Errorf("load flow %q: layer %q not found", flow.Name, layer.Name)
		}
		layers = append(layers, loadedLayer)
	}
	le.flows[flow.Name] = layers
	le.flowInputs[flow.Name] = flow.Input
	return nil
}

func (le *LayerEngine) RunLayer(name string, inputValues []any) (any, error) {
	layer, ok := le.layers[name]
	if !ok || layer == nil {
		return nil, fmt.Errorf("layer %q not found", name)
	}
	result, err := runLayer(layer, inputValues)
	return result, err
}

func (le *LayerEngine) RunFlow(name string, inputValues map[string]any) (any, error) {
	layers, ok := le.flows[name]
	if !ok {
		return nil, fmt.Errorf("flow %q not found", name)
	}

	for _, fi := range le.flowInputs[name] {
		_, provided := inputValues[fi.Name]
		if !provided && !fi.Optional {
			return nil, fmt.Errorf("required input %q not provided", fi.Name)
		}
	}
	return runFlow(layers, inputValues)
}
