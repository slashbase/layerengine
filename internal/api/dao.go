package api

import (
	"encoding/json"
	"time"

	"github.com/paraswaykole/layerdotrun/internal/store"
)

const BucketName = "apis"

type Api struct {
	Name      string         `json:"name"`
	Path      string         `json:"path"`
	Method    string         `json:"method"`
	Flow      string         `json:"flow"`
	InputMap  map[string]int `json:"inputMap"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

func toApi(data map[string]interface{}) Api {
	var api Api
	jsonData, _ := json.Marshal(data)
	if err := json.Unmarshal(jsonData, &api); err != nil {
		return api
	}
	return api
}

type ApiDAO struct {
	store *store.Store
}

func NewApiDAO(store *store.Store) *ApiDAO {
	return &ApiDAO{store: store}
}

func (dao *ApiDAO) GetApi(name string) (*Api, error) {
	var api Api
	err := dao.store.Read(BucketName, name, &api)
	return &api, err
}

func (dao *ApiDAO) PutApi(api Api) error {
	err := dao.store.Update(BucketName, api.Name, api)
	return err
}

func (dao *ApiDAO) GetApis() (map[string]Api, error) {
	allItems, err := dao.store.ReadAll(BucketName)
	if err != nil {
		return nil, err
	}
	var data = map[string]Api{}
	for key, item := range allItems {
		data[key] = toApi(item.(map[string]interface{}))
	}
	return data, nil
}
