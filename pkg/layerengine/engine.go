package layerengine

import (
	"errors"
)

type LayerEngine struct {
	layers map[string]Layer
	flows  map[string][]Layer
}

func NewLayerEngine() *LayerEngine {
	layerEngine := &LayerEngine{}
	layerEngine.init()
	return layerEngine
}

func (le *LayerEngine) init() {
	le.layers = map[string]Layer{}
	le.flows = map[string][]Layer{}
}

func (le *LayerEngine) Run(runtype RunType, name string, inputValues []interface{}) (interface{}, error) {
	switch runtype {
	case LAYER:
		runLayer(le.layers[name], inputValues)
	case FLOW:
		runFlow(le.flows[name], inputValues)
	}
	return nil, errors.New("invald runtype")
}
