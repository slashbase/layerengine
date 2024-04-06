package layer

import (
	"time"

	"github.com/paraswaykole/layerdotrun/pkg/codegen"
	"github.com/paraswaykole/layerdotrun/pkg/layerengine"
)

type LayerService struct {
	layerDao    *LayerDAO
	layerEngine *layerengine.LayerEngine
}

func NewLayerService(layerDao *LayerDAO, layerEngine *layerengine.LayerEngine) *LayerService {
	return &LayerService{layerDao: layerDao, layerEngine: layerEngine}
}

func (service LayerService) GetLayer(name string) (*Layer, error) {
	layer, err := service.layerDao.GetLayer(name)
	return layer, err
}

type UpdateLayerRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	InputNames  []string `json:"inputNames"`
	OutputNames []string `json:"outputNames"`
}

func (service LayerService) UpdateLayer(layerReq UpdateLayerRequest) (*Layer, error) {

	var code string

	code, err := codegen.GenerateLayerFunction(codegen.OPENAI_GPT3DOT5_TURBO, layerReq.Name, layerReq.Description, layerReq.InputNames, layerReq.OutputNames)
	if err != nil {
		return nil, err
	}

	layer := Layer{
		Name:        layerReq.Name,
		Description: layerReq.Description,
		InputNames:  layerReq.InputNames,
		OutputNames: layerReq.OutputNames,
		Code:        code,
		UpdatedAt:   time.Now(),
	}

	err = service.layerDao.PutLayer(layer)

	eLayers := []layerengine.Layer{
		{
			Name:      layer.Name,
			Code:      layer.Code,
			OutputLen: len(layer.OutputNames),
		},
	}

	service.layerEngine.LoadLayers(eLayers)

	return &layer, err
}

func (service LayerService) LoadAllLayers() {
	layers, err := service.layerDao.GetLayers()
	if err != nil {
		return
	}
	eLayers := []layerengine.Layer{}
	for _, layer := range layers {
		lay := layerengine.Layer{
			Name:      layer.Name,
			OutputLen: len(layer.OutputNames),
			Code:      layer.Code,
		}
		eLayers = append(eLayers, lay)
	}
	service.layerEngine.LoadLayers(eLayers)
}
