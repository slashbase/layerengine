package api

import (
	"time"
)

type ApiService struct {
	apiDao *ApiDAO
}

func NewApiService(apiDao *ApiDAO) *ApiService {
	return &ApiService{apiDao: apiDao}
}

func (service ApiService) GetApi(name string) (*Api, error) {
	api, err := service.apiDao.GetApi(name)
	return api, err
}

type UpdateApiRequest struct {
	Name   string   `json:"name"`
	Path   string   `json:"path"`
	Method string   `json:"method"`
	Flow   string   `json:"flow"`
	Inputs []string `json:"inputs"`
}

func (service ApiService) UpdateApi(apiReq UpdateApiRequest) (*Api, error) {

	inputMap := map[string]int{}
	for i, input := range apiReq.Inputs {
		inputMap[input] = i
	}

	api := Api{
		Name:      apiReq.Name,
		Path:      apiReq.Path,
		Method:    apiReq.Method,
		Flow:      apiReq.Flow,
		InputMap:  inputMap,
		UpdatedAt: time.Now(),
	}

	err := service.apiDao.PutApi(api)

	return &api, err
}
