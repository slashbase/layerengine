package global

import (
	"github.com/paraswaykole/layerdotrun/internal/api"
	"github.com/paraswaykole/layerdotrun/pkg/layerengine"
)

type GlobalService struct {
	apiIndex    *map[api.HttpMethod]map[string]api.Api
	layerEngine *layerengine.LayerEngine
}

func NewGlobalService(apiService *api.ApiService, layerEngine *layerengine.LayerEngine) *GlobalService {
	apiIndex, _ := apiService.GetApiIndex()
	return &GlobalService{apiIndex: apiIndex, layerEngine: layerEngine}
}

func (service GlobalService) RunApi(path, methodStr string, body map[string]interface{}) (interface{}, error) {

	method := api.HttpMethod(methodStr)

	api := (*service.apiIndex)[method][path]

	inputValues := api.InputMap.GetInputArray(body)

	output, err := service.layerEngine.Run(layerengine.FLOW, api.Flow, inputValues)
	if err != nil {
		return nil, err
	}

	result := api.OutputMap.ToMapOutput(output.([]interface{}))

	return result, nil
}
