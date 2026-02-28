package flow

import (
	"encoding/json"
	"time"

	"github.com/slashbase/layerengine/internal/store"
)

const BucketName = "flows"

type Flow struct {
	Name      string    `json:"name"`
	Layers    []string  `json:"layers"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func toFlow(data map[string]interface{}) Flow {
	var layer Flow
	jsonData, _ := json.Marshal(data)
	if err := json.Unmarshal(jsonData, &layer); err != nil {
		return layer
	}
	return layer
}

type FlowDAO struct {
	store *store.Store
}

func NewFlowDAO(store *store.Store) *FlowDAO {
	return &FlowDAO{store: store}
}

func (dao *FlowDAO) GetFlow(name string) (*Flow, error) {
	var flow Flow
	err := dao.store.Read(BucketName, name, &flow)
	return &flow, err
}

func (dao *FlowDAO) PutFlow(flow Flow) error {
	err := dao.store.Update(BucketName, flow.Name, flow)
	return err
}

func (dao *FlowDAO) GetFlows() (map[string]Flow, error) {
	allItems, err := dao.store.ReadAll(BucketName)
	if err != nil {
		return nil, err
	}
	var data = map[string]Flow{}
	for key, item := range allItems {
		data[key] = toFlow(item.(map[string]interface{}))
	}
	return data, nil
}
