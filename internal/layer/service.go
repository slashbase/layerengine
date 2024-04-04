package layer

type LayerService struct {
	layerDao *LayerDAO
}

func NewLayerService(layerDao *LayerDAO) *LayerService {
	return &LayerService{layerDao: layerDao}
}

type UpdateLayerRequest struct {
	FunctionName string   `json:"functionName"`
	Description  string   `json:"description"`
	InputNames   []string `json:"inputNames"`
	OutputNames  []string `json:"outputNames"`
}

func (service LayerService) UpdateLayer(name string, layerReq UpdateLayerRequest) {

}
