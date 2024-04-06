package layer

import (
	"encoding/json"
	"time"

	"github.com/paraswaykole/layerdotrun/internal/store"
)

const BucketName = "layers"

type Layer struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	InputNames  []string  `json:"inputNames"`
	OutputNames []string  `json:"outputNames"`
	Code        string    `json:"code"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func toLayer(data map[string]interface{}) Layer {
	var layer Layer
	jsonData, _ := json.Marshal(data)
	if err := json.Unmarshal(jsonData, &layer); err != nil {
		return layer
	}
	return layer
}

type LayerDAO struct {
	store *store.Store
}

func NewLayerDAO(store *store.Store) *LayerDAO {
	return &LayerDAO{store: store}
}

func (dao *LayerDAO) GetLayer(name string) (*Layer, error) {
	var layer Layer
	err := dao.store.Read(BucketName, name, &layer)
	return &layer, err
}

func (dao *LayerDAO) PutLayer(layer Layer) error {
	err := dao.store.Update(BucketName, layer.Name, layer)
	return err
}

func (dao *LayerDAO) GetLayers() (map[string]Layer, error) {
	allItems, err := dao.store.ReadAll(BucketName)
	if err != nil {
		return nil, err
	}
	var data = map[string]Layer{}
	for key, item := range allItems {
		data[key] = toLayer(item.(map[string]interface{}))
	}
	return data, nil
}
