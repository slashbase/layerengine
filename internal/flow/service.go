package flow

import (
	"time"

	"github.com/slashbase/layerengine/internal/layer"
	"github.com/slashbase/layerengine/pkg/codegen"
	"github.com/slashbase/layerengine/pkg/layerengine"
)

type FlowService struct {
	flowDao     *FlowDAO
	layerDao    *layer.LayerDAO
	layerEngine *layerengine.LayerEngine
}

func NewFlowService(flowDao *FlowDAO, layerDao *layer.LayerDAO, layerEngine *layerengine.LayerEngine) *FlowService {
	return &FlowService{flowDao: flowDao, layerDao: layerDao, layerEngine: layerEngine}
}

func (service FlowService) GetFlow(name string) (*Flow, error) {
	layer, err := service.flowDao.GetFlow(name)
	return layer, err
}

type UpdateFlowRequest struct {
	Name   string `json:"name"`
	Layers []struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		InputNames  []string `json:"inputNames"`
		OutputNames []string `json:"outputNames"`
	} `json:"layers"`
}

func (service FlowService) UpdateFlow(flowReq UpdateFlowRequest) (*Flow, error) {

	layerNames := []string{}
	eLayers := []layerengine.Layer{}

	for _, layerReq := range flowReq.Layers {
		var layerInstance layer.Layer
		if layerReq.Description == "" && len(layerReq.InputNames) == 0 && len(layerReq.OutputNames) == 0 {
			foundLayer, err := service.layerDao.GetLayer(layerReq.Name)
			if err != nil {
				return nil, err
			}
			layerInstance = *foundLayer
		} else {
			var code string

			code, err := codegen.GenerateLayerFunction(codegen.OPENAI_GPT3DOT5_TURBO, layerReq.Name, layerReq.Description, layerReq.InputNames, layerReq.OutputNames)
			if err != nil {
				return nil, err
			}

			layerInstance = layer.Layer{
				Name:        layerReq.Name,
				Description: layerReq.Description,
				InputNames:  layerReq.InputNames,
				OutputNames: layerReq.OutputNames,
				Code:        code,
				UpdatedAt:   time.Now(),
			}

			err = service.layerDao.PutLayer(layerInstance)
			if err != nil {
				return nil, err
			}
		}
		layerNames = append(layerNames, layerInstance.Name)

		eLayers = append(eLayers, layerengine.Layer{
			Name:      layerInstance.Name,
			Code:      layerInstance.Code,
			OutputLen: len(layerInstance.OutputNames),
		})
	}

	flow := Flow{
		Name:      flowReq.Name,
		Layers:    layerNames,
		UpdatedAt: time.Now(),
	}

	err := service.flowDao.PutFlow(flow)
	if err != nil {
		return nil, err
	}

	service.layerEngine.LoadLayers(eLayers)
	service.layerEngine.LoadFlow(map[string][]string{flow.Name: flow.Layers})

	return &flow, nil
}

func (service FlowService) LoadAllFlows() {
	flows, err := service.flowDao.GetFlows()
	if err != nil {
		return
	}
	data := map[string][]string{}
	for _, flow := range flows {
		data[flow.Name] = flow.Layers
	}
	service.layerEngine.LoadFlow(data)
}
