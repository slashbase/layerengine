package api

import "time"

type ApiView struct {
	Name      string         `json:"name"`
	Path      string         `json:"path"`
	Method    string         `json:"method"`
	Flow      string         `json:"flow"`
	InputMap  map[string]int `json:"inputMap"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

func ToApiView(api *Api) ApiView {
	return ApiView{
		Name:      api.Name,
		Path:      api.Path,
		Method:    api.Method,
		Flow:      api.Flow,
		InputMap:  api.InputMap,
		UpdatedAt: api.UpdatedAt,
	}
}
