package layer

import "time"

type LayerView struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	InputNames  []string  `json:"inputNames"`
	OutputNames []string  `json:"outputNames"`
	Code        string    `json:"code"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func ToLayerView(layer *Layer) LayerView {
	return LayerView{
		Name:        layer.Name,
		Description: layer.Description,
		InputNames:  layer.InputNames,
		OutputNames: layer.OutputNames,
		Code:        layer.Code,
		UpdatedAt:   layer.UpdatedAt,
	}
}
