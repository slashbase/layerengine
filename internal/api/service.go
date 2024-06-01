package api

import (
	"errors"
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
	Name    string     `json:"name"`
	Path    string     `json:"path"`
	Method  HttpMethod `json:"method"`
	Flow    string     `json:"flow"`
	Inputs  IOMap      `json:"inputs"`
	Outputs IOMap      `json:"outputs"`
}

func (service ApiService) UpdateApi(apiReq UpdateApiRequest) (*Api, error) {

	if !apiReq.Inputs.IsValid() {
		return nil, errors.New("invalid input map")
	}

	if !apiReq.Outputs.IsValid() {
		return nil, errors.New("invalid output map")
	}

	api := Api{
		Name:      apiReq.Name,
		Path:      apiReq.Path,
		Method:    apiReq.Method,
		Flow:      apiReq.Flow,
		InputMap:  apiReq.Inputs,
		OutputMap: apiReq.Outputs,
		UpdatedAt: time.Now(),
	}

	err := service.apiDao.PutApi(api)

	return &api, err
}

func (service ApiService) GetApiIndex() (*map[HttpMethod]map[string]Api, error) {
	apis, err := service.apiDao.GetApis()

	allApiIndex := map[HttpMethod]map[string]Api{}

	for _, api := range apis {
		allApiIndex[api.Method] = map[string]Api{
			api.Path: api,
		}
	}

	return &allApiIndex, err
}
