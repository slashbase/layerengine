package layerengine

import (
	"errors"
)

type LayerEngine struct {
	layers map[string]*Layer
	flows  map[string][]*Layer
}

func NewLayerEngine() *LayerEngine {
	layerEngine := &LayerEngine{}
	layerEngine.init()
	return layerEngine
}

func (le *LayerEngine) init() {
	le.layers = map[string]*Layer{}
	le.flows = map[string][]*Layer{}
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

func (le *LayerEngine) Run(runtype RunType, name string, inputValues []interface{}) (interface{}, error) {
	switch runtype {
	case LAYER:
		return runLayer(le.layers[name], inputValues)
	case FLOW:
		return runFlow(le.flows[name], inputValues)
	}
	return nil, errors.New("invald runtype")
}
