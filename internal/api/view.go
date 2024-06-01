package api

import "time"

type ApiView struct {
	Name      string     `json:"name"`
	Path      string     `json:"path"`
	Method    HttpMethod `json:"method"`
	Flow      string     `json:"flow"`
	InputMap  IOMap      `json:"inputMap"`
	OutputMap IOMap      `json:"outputMap"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

func ToApiView(api *Api) ApiView {
	return ApiView{
		Name:      api.Name,
		Path:      api.Path,
		Method:    api.Method,
		Flow:      api.Flow,
		InputMap:  api.InputMap,
		OutputMap: api.OutputMap,
		UpdatedAt: api.UpdatedAt,
	}
}
