package layerengine

import (
	"fmt"
	"maps"
	"strings"
	"sync"

	"github.com/slashbase/layerengine/codegen"
	"github.com/slashbase/layerengine/modules"
	"github.com/slashbase/layerengine/validator"
)

type LayerEngine struct {
	mu         sync.RWMutex
	layers     map[string]*Layer
	flows      map[string][]*Layer
	flowInputs map[string][]FlowInput
	codegen    *codegen.CodeGen
	mods       []modules.Module
}

func NewLayerEngine(codegenertor *codegen.CodeGen) *LayerEngine {
	layerEngine := &LayerEngine{
		codegen: codegenertor,
		mods:    modules.AllModules(),
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

func (le *LayerEngine) LoadModules(mods []modules.Module) {
	le.mu.Lock()
	defer le.mu.Unlock()
	le.mods = append(le.mods, mods...)
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

	le.mu.RLock()
	mods := append([]modules.Module(nil), le.mods...)
	le.mu.RUnlock()

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
			mods,
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
	seenNames := make(map[string]struct{}, len(layers))
	for i := range layers {
		layer := layers[i]
		name := strings.TrimSpace(layer.Name)
		if _, exists := seenNames[name]; exists {
			return fmt.Errorf("duplicate layer name %q", layer.Name)
		}
		seenNames[name] = struct{}{}

		fnProto, err := ParseAndCompileLuaCode(layer.Code)
		if err != nil {
			return fmt.Errorf("compile layer %q: %w", layer.Name, err)
		}
		layer.FnProto = fnProto
		compiledLayers[layer.Name] = &layer
	}

	le.mu.Lock()
	defer le.mu.Unlock()
	maps.Copy(le.layers, compiledLayers)
	return nil
}

// LoadFlow registers a flow after confirming that every referenced layer was
// loaded successfully.
func (le *LayerEngine) LoadFlow(flow Flow) error {
	le.mu.Lock()
	defer le.mu.Unlock()

	layers := make([]*Layer, 0, len(flow.Layers))
	seenNames := make(map[string]struct{}, len(flow.Layers))
	for _, layer := range flow.Layers {
		name := strings.TrimSpace(layer.Name)
		if _, exists := seenNames[name]; exists {
			return fmt.Errorf("load flow %q: duplicate layer name %q", flow.Name, layer.Name)
		}
		seenNames[name] = struct{}{}

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
	le.mu.RLock()
	layer, ok := le.layers[name]
	mods := append([]modules.Module(nil), le.mods...)
	le.mu.RUnlock()
	if !ok || layer == nil {
		return nil, fmt.Errorf("layer %q not found", name)
	}
	result, err := runLayer(layer, inputValues, mods)
	return result, err
}

func (le *LayerEngine) RunFlow(name string, inputValues map[string]any) (any, error) {
	le.mu.RLock()
	layers, ok := le.flows[name]
	if ok {
		layers = append([]*Layer(nil), layers...)
	}
	flowInputs := append([]FlowInput(nil), le.flowInputs[name]...)
	mods := append([]modules.Module(nil), le.mods...)
	le.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("flow %q not found", name)
	}

	workingValues := make(map[string]any, len(inputValues))
	maps.Copy(workingValues, inputValues)

	for _, fi := range flowInputs {
		_, provided := workingValues[fi.Name]
		if !provided && !fi.Optional {
			return nil, fmt.Errorf("required input %q not provided", fi.Name)
		}
	}
	return runFlow(layers, workingValues, mods)
}

// Layer returns a copy of a loaded layer by name. The copy carries the
// generated Lua in its Code field, so callers can inspect or persist it without
// holding the engine's own value.
//
// The copy is shallow: the Input/Output slices and the compiled FnProto are
// shared with the engine's layer, so read them rather than mutate them.
func (le *LayerEngine) Layer(name string) (*Layer, bool) {
	le.mu.RLock()
	defer le.mu.RUnlock()
	layer, ok := le.layers[name]
	if !ok || layer == nil {
		return nil, false
	}
	snapshot := *layer
	return &snapshot, true
}

// GenerateLayer generates the Lua source for a single layer of flow, using the
// flow's declared inputs and the outputs of the layers before it. It does not
// mutate flow: assign the result to flow.Layers[index].Code yourself, or call
// GenerateLayers to do the whole flow at once.
//
// This is the api for callers that want to retry per layer — regenerate one
// layer without re-generating the whole flow.
func (le *LayerEngine) GenerateLayer(flow *Flow, index int) error {
	if le.codegen == nil {
		return fmt.Errorf("code generator is not configured")
	}
	if index < 0 || index >= len(flow.Layers) {
		return fmt.Errorf("layer index %d out of range (flow has %d layers)", index, len(flow.Layers))
	}

	flowInputMap := make(map[string]FlowInput, len(flow.Input))
	for _, fi := range flow.Input {
		flowInputMap[fi.Name] = fi
	}

	// Outputs of earlier layers are available as inputs to this one.
	flowOutputMap := make(map[string]codegen.Output)
	for i := range index {
		for _, out := range flow.Layers[i].Output {
			var o codegen.Output
			codegen.MapToStruct(StructToMap(out), &o)
			flowOutputMap[out.Name] = o
		}
	}

	layer := flow.Layers[index]

	inputs := make([]codegen.Input, len(layer.Input))
	for i, inpName := range layer.Input {
		if fi, exists := flowInputMap[inpName]; exists {
			codegen.MapToStruct(StructToMap(fi), &inputs[i])
		} else {
			codegen.MapToStruct(StructToMap(flowOutputMap[inpName]), &inputs[i])
			inputs[i].Optional = false
		}
	}

	outputs := make([]codegen.Output, len(layer.Output))
	for i, out := range layer.Output {
		codegen.MapToStruct(StructToMap(out), &outputs[i])
	}

	le.mu.RLock()
	mods := append([]modules.Module(nil), le.mods...)
	le.mu.RUnlock()

	code, err := le.codegen.GenerateLayerFunction(
		layer.Name,
		layer.Description,
		inputs,
		outputs,
		mods,
	)

	if err != nil {
		return err
	}

	flow.Layers[index].Code = code
	return nil
}
