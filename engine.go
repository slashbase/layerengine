package layerengine

import (
	"context"
	"fmt"
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
	// layerOwner records which flow loaded each layer. Layer names are
	// engine-global, so this is what lets a second flow be refused instead of
	// silently overwriting the first flow's code.
	layerOwner map[string]string
	// flowDefs keeps each loaded flow's definition, so a caller can read back
	// what a spec declared after LoadSpec has thrown the parsed spec away.
	flowDefs map[string]Flow
	codegen  *codegen.CodeGen
	mods     []modules.Module
}

func NewLayerEngine(codegenertor *codegen.CodeGen) *LayerEngine {
	layerEngine := &LayerEngine{
		codegen: codegenertor,
		mods:    modules.AllModules(),
	}
	layerEngine.init()
	return layerEngine
}

// "Blank" means no LLM: the built-in modules are loaded all the same, so the sandbox is the one the
// codegen prompt describes and a hand-loaded layer calling system.version()
// behaves as it would on a full engine.
func NewBlankLayerEngine() *LayerEngine {
	layerEngine := &LayerEngine{mods: modules.AllModules()}
	layerEngine.init()
	return layerEngine
}

func (le *LayerEngine) init() {
	le.layers = map[string]*Layer{}
	le.flows = map[string][]*Layer{}
	le.flowInputs = map[string][]FlowInput{}
	le.layerOwner = map[string]string{}
	le.flowDefs = map[string]Flow{}
}

// LoadModules adds modules to the engine, after holding the set to what it
// claims. See checkModules for the three checks and their order.
//
// A rejected set leaves the engine unchanged. This is where the checks belong
// because it is where the module set becomes final: a module accepted here is
// trusted by every layer that runs afterwards.
func (le *LayerEngine) LoadModules(mods []modules.Module) error {
	le.mu.Lock()
	defer le.mu.Unlock()

	combined := append(append([]modules.Module(nil), le.mods...), mods...)
	if err := checkModules(combined); err != nil {
		return err
	}
	le.mods = combined
	return nil
}

func (le *LayerEngine) GenerateLayers(flow *Flow) error {
	if le.codegen == nil {
		return fmt.Errorf("code generator is not configured")
	}

	le.mu.RLock()
	mods := append([]modules.Module(nil), le.mods...)
	le.mu.RUnlock()

	for i, layer := range flow.Layers {
		inputs, err := layerInputs(flow, i)
		if err != nil {
			return err
		}
		outputs := toCodegenOutputs(layer.Output)

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

// layerInputs maps one layer's declared input names onto codegen.Input, looking
// each name up among the flow's inputs and the outputs of the layers before
// index.
//
// A name that is in neither is an error rather than an empty codegen.Input: it
// would otherwise reach the LLM as an input with no name and no type, and the
// layer would be generated against a contract nobody wrote.
func layerInputs(flow *Flow, index int) ([]codegen.Input, error) {
	flowInputs := make(map[string]FlowInput, len(flow.Input))
	for _, fi := range flow.Input {
		flowInputs[fi.Name] = fi
	}

	earlierOutputs := make(map[string]LayerOutput)
	for i := range index {
		for _, out := range flow.Layers[i].Output {
			earlierOutputs[out.Name] = out
		}
	}

	layer := flow.Layers[index]
	inputs := make([]codegen.Input, 0, len(layer.Input))
	for _, name := range layer.Input {
		if fi, ok := flowInputs[name]; ok {
			inputs = append(inputs, codegen.Input{
				Name:        fi.Name,
				Type:        fi.Type,
				Description: fi.Description,
				Optional:    fi.Optional,
			})
			continue
		}

		out, ok := earlierOutputs[name]
		if !ok {
			return nil, fmt.Errorf(
				"layer %q: input %q is not a flow input and is not produced by an earlier layer",
				layer.Name, name,
			)
		}
		inputs = append(inputs, codegen.Input{
			Name:        out.Name,
			Type:        out.Type,
			Description: out.Description,
			// An earlier layer always produces its declared outputs.
			Optional: false,
		})
	}
	return inputs, nil
}

// toCodegenOutputs maps a layer's declared outputs onto codegen.Output.
func toCodegenOutputs(outputs []LayerOutput) []codegen.Output {
	converted := make([]codegen.Output, len(outputs))
	for i, out := range outputs {
		converted[i] = codegen.Output{
			Name:        out.Name,
			Type:        out.Type,
			Description: out.Description,
		}
	}
	return converted
}

// LoadSpec validates a flow spec, generates every layer, and loads the flow.
//
// It is all or nothing: layers are compiled into a local map and the engine is
// written once, at the end, so a spec that fails at registration leaves no
// half-loaded layers behind.
func (le *LayerEngine) LoadSpec(spec string) error {
	f, err := validator.Run([]byte(spec))
	if err != nil {
		return err
	}

	var flow Flow
	MapToStruct(validator.StructToMap(f), &flow)

	if err := le.GenerateLayers(&flow); err != nil {
		return err
	}

	compiledLayers, err := le.compileLayers(flow.Layers)
	if err != nil {
		return err
	}

	return le.commitLayers(flow.Name, compiledLayers, &flow)
}

// LoadLayers compiles and loads all layers. It leaves the engine unchanged if
// any layer cannot be compiled, or if a compiled chunk does not define a global
// function named after its layer.
//
// Layers loaded this way are unowned: no flow is recorded as their source, so a
// later hand load may replace them. Flow-owned layers come from LoadSpec, which
// is the path that refuses to let a second flow replace a name.
func (le *LayerEngine) LoadLayers(layers []Layer) error {
	compiledLayers, err := le.compileLayers(layers)
	if err != nil {
		return err
	}
	return le.commitLayers("", compiledLayers, nil)
}

// compileLayers parses, compiles and checks every layer without touching engine
// state, so a set that fails partway through leaves the engine as it was.
func (le *LayerEngine) compileLayers(layers []Layer) (map[string]*Layer, error) {
	le.mu.RLock()
	mods := append([]modules.Module(nil), le.mods...)
	le.mu.RUnlock()

	compiledLayers := make(map[string]*Layer, len(layers))
	seenNames := make(map[string]struct{}, len(layers))
	for i := range layers {
		layer := layers[i]
		name := strings.TrimSpace(layer.Name)
		if _, exists := seenNames[name]; exists {
			return nil, fmt.Errorf("duplicate layer name %q", layer.Name)
		}
		seenNames[name] = struct{}{}

		fnProto, err := ParseAndCompileLuaCode(layer.Code)
		if err != nil {
			return nil, fmt.Errorf("compile layer %q: %w", layer.Name, err)
		}
		layer.FnProto = fnProto

		// Compiling proves the code parses; it does not prove the chunk defines
		// the function the runner looks up by name. Check that here so a
		// misnamed or empty layer fails at load, not at RunFlow.
		if err := checkLayerDefinesFunction(&layer, mods); err != nil {
			return nil, err
		}

		compiledLayers[layer.Name] = &layer
	}
	return compiledLayers, nil
}

// commitLayers installs compiled layers and, when flow is non-nil, registers it
// too — under one write lock, so a load happens completely or not at all.
//
// Layer names are engine-global. A name already owned by a flow cannot be
// replaced by different code, and the error names the flow that holds it.
// Identical code is accepted and the layer already loaded is kept, which makes
// re-loading the same spec idempotent and keeps pointers held by a registered
// flow valid. An unowned layer — one loaded by hand through LoadLayers — may be
// replaced, because that path is the explicit override.
func (le *LayerEngine) commitLayers(flowName string, compiledLayers map[string]*Layer, flow *Flow) error {
	le.mu.Lock()
	defer le.mu.Unlock()

	// Decide what each name resolves to before writing anything.
	effective := make(map[string]*Layer, len(compiledLayers))
	for name, layer := range compiledLayers {
		existing, ok := le.layers[name]
		switch {
		case !ok || existing == nil:
			effective[name] = layer
		case existing.Code == layer.Code:
			effective[name] = existing
		default:
			owner := le.layerOwner[name]
			if owner != "" && owner != flowName {
				return fmt.Errorf(
					"layer %q is already loaded by flow %q and this flow generates different code for it; layer names are engine-global, so rename one of the two layers",
					name, owner,
				)
			}
			effective[name] = layer
		}
	}

	// Resolve the flow's layer list against what this commit will install plus
	// what is already loaded, so a flow that cannot be registered leaves the
	// engine untouched.
	var flowLayers []*Layer
	if flow != nil {
		resolved, err := le.resolveFlowLayersLocked(*flow, effective)
		if err != nil {
			return err
		}
		flowLayers = resolved
	}

	for name, layer := range effective {
		if existing, ok := le.layers[name]; ok && existing == layer {
			continue
		}
		le.layers[name] = layer
		if flowName != "" {
			le.layerOwner[name] = flowName
		}
	}

	if flow != nil {
		le.flows[flow.Name] = flowLayers
		le.flowInputs[flow.Name] = flow.Input
		le.flowDefs[flow.Name] = *flow
	}
	return nil
}

// resolveFlowLayersLocked builds a flow's layer list from staged and already
// loaded layers. The caller holds le.mu for writing.
func (le *LayerEngine) resolveFlowLayersLocked(flow Flow, staged map[string]*Layer) ([]*Layer, error) {
	layers := make([]*Layer, 0, len(flow.Layers))
	seenNames := make(map[string]struct{}, len(flow.Layers))
	for _, layer := range flow.Layers {
		name := strings.TrimSpace(layer.Name)
		if _, exists := seenNames[name]; exists {
			return nil, fmt.Errorf("load flow %q: duplicate layer name %q", flow.Name, layer.Name)
		}
		seenNames[name] = struct{}{}

		loadedLayer, ok := staged[layer.Name]
		if !ok {
			loadedLayer, ok = le.layers[layer.Name]
		}
		if !ok || loadedLayer == nil {
			return nil, fmt.Errorf("load flow %q: layer %q not found", flow.Name, layer.Name)
		}
		layers = append(layers, loadedLayer)
	}
	return layers, nil
}

// LoadFlow registers a flow after confirming that every referenced layer was
// loaded successfully.
func (le *LayerEngine) LoadFlow(flow Flow) error {
	le.mu.Lock()
	defer le.mu.Unlock()

	layers, err := le.resolveFlowLayersLocked(flow, nil)
	if err != nil {
		return err
	}
	le.flows[flow.Name] = layers
	le.flowInputs[flow.Name] = flow.Input
	le.flowDefs[flow.Name] = flow
	return nil
}

// RunLayer runs one loaded layer on positional arguments.
func (le *LayerEngine) RunLayer(name string, inputValues []any) (any, error) {
	return le.RunLayerContext(context.Background(), name, inputValues)
}

// RunLayerContext is RunLayer with a caller-supplied context.
//
// Cancelling the context stops the layer: the per-layer timeout is derived from
// it, so cancellation interrupts the running Lua chunk instead of waiting out
// the 60s limit. A cancelled run reports context.Canceled, not a timeout.
func (le *LayerEngine) RunLayerContext(ctx context.Context, name string, inputValues []any) (any, error) {
	le.mu.RLock()
	layer, ok := le.layers[name]
	mods := append([]modules.Module(nil), le.mods...)
	le.mu.RUnlock()
	if !ok || layer == nil {
		return nil, fmt.Errorf("layer %q not found", name)
	}
	return runLayer(ctx, layer, inputValues, mods)
}

// RunFlow runs a loaded flow. Input values are converted to the types the flow
// declares, and arguments the flow does not declare are dropped; see
// CoerceInputs for the rules. Callers that want a different policy — strict type
// checking, or forwarding undeclared arguments — use RunFlowWithOptions.
func (le *LayerEngine) RunFlow(name string, inputValues map[string]any) (any, error) {
	return le.RunFlowWithOptions(name, inputValues, CoerceOptions{})
}

// RunFlowContext is RunFlow with a caller-supplied context; cancelling it stops
// the flow at the layer that is running.
func (le *LayerEngine) RunFlowContext(ctx context.Context, name string, inputValues map[string]any) (any, error) {
	return le.RunFlowWithOptionsContext(ctx, name, inputValues, CoerceOptions{})
}

// RunFlowWithOptions is RunFlow with an explicit input policy.
func (le *LayerEngine) RunFlowWithOptions(name string, inputValues map[string]any, opts CoerceOptions) (any, error) {
	return le.RunFlowWithOptionsContext(context.Background(), name, inputValues, opts)
}

// RunFlowWithOptionsContext is RunFlowWithOptions with a caller-supplied context.
func (le *LayerEngine) RunFlowWithOptionsContext(ctx context.Context, name string, inputValues map[string]any, opts CoerceOptions) (any, error) {
	_, result, err := le.executeFlow(ctx, name, inputValues, opts)
	return result, err
}

// RunFlowResult is RunFlow with the result named: the last layer's outputs keyed
// by the names that layer declares, and the name of the layer that produced
// them. Use it when the caller did not write the spec and cannot know the
// positional order of the result.
func (le *LayerEngine) RunFlowResult(name string, inputValues map[string]any) (FlowResult, error) {
	return le.RunFlowResultContext(context.Background(), name, inputValues)
}

// RunFlowResultContext is RunFlowResult with a caller-supplied context.
func (le *LayerEngine) RunFlowResultContext(ctx context.Context, name string, inputValues map[string]any) (FlowResult, error) {
	layers, result, err := le.executeFlow(ctx, name, inputValues, CoerceOptions{})
	if err != nil {
		return FlowResult{}, err
	}
	return namedResult(layers, result), nil
}

// executeFlow resolves a flow, coerces its inputs and runs it, returning the
// layers it ran so the caller can name the results.
func (le *LayerEngine) executeFlow(ctx context.Context, name string, inputValues map[string]any, opts CoerceOptions) ([]*Layer, []any, error) {
	le.mu.RLock()
	layers, ok := le.flows[name]
	if ok {
		layers = append([]*Layer(nil), layers...)
	}
	flowInputs := append([]FlowInput(nil), le.flowInputs[name]...)
	mods := append([]modules.Module(nil), le.mods...)
	le.mu.RUnlock()
	if !ok {
		return nil, nil, fmt.Errorf("flow %q not found", name)
	}

	workingValues, err := CoerceInputs(flowInputs, inputValues, opts)
	if err != nil {
		return nil, nil, err
	}

	result, err := runFlow(ctx, layers, workingValues, mods)
	if err != nil {
		return nil, nil, err
	}
	return layers, result, nil
}

// namedResult keys a run's results by the output names the last layer declares.
// runFlow returns them in declaration order, which is the only thing a caller
// holding a spec it did not write cannot recover on its own.
func namedResult(layers []*Layer, result []any) FlowResult {
	if len(layers) == 0 {
		return FlowResult{Outputs: map[string]any{}}
	}

	last := layers[len(layers)-1]
	outputs := make(map[string]any, len(last.Output))
	for i, out := range last.Output {
		if i >= len(result) {
			break
		}
		outputs[out.Name] = result[i]
	}
	return FlowResult{Outputs: outputs, Layer: last.Name}
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

// Flow returns a copy of a loaded flow's definition by name, so a caller can
// read back what a spec declared — the input types, and which outputs the last
// layer produces — after LoadSpec has thrown the parsed spec away.
//
// The copy is shallow: the Input and Layers slices are shared with the engine's
// value, so read them rather than mutate them.
func (le *LayerEngine) Flow(name string) (*Flow, bool) {
	le.mu.RLock()
	defer le.mu.RUnlock()
	flow, ok := le.flowDefs[name]
	if !ok {
		return nil, false
	}
	return &flow, true
}

// GenerateLayer generates the Lua source for a single layer of flow, using the
// flow's declared inputs and the outputs of the layers before it, and stores it
// on flow.Layers[index].Code.
//
// This is the API for callers that want to retry per layer — regenerate one
// layer without regenerating the whole flow.
func (le *LayerEngine) GenerateLayer(flow *Flow, index int) error {
	if le.codegen == nil {
		return fmt.Errorf("code generator is not configured")
	}
	if index < 0 || index >= len(flow.Layers) {
		return fmt.Errorf("layer index %d out of range (flow has %d layers)", index, len(flow.Layers))
	}

	inputs, err := layerInputs(flow, index)
	if err != nil {
		return err
	}
	outputs := toCodegenOutputs(flow.Layers[index].Output)

	le.mu.RLock()
	mods := append([]modules.Module(nil), le.mods...)
	le.mu.RUnlock()

	layer := flow.Layers[index]

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

// GetModules returns the module set every layer sandbox gets.
//
// The slice is a copy: appending to it cannot reach the engine's own set, and
// the read is guarded like every other accessor here.
func (le *LayerEngine) GetModules() []modules.Module {
	le.mu.RLock()
	defer le.mu.RUnlock()
	return append([]modules.Module(nil), le.mods...)
}
