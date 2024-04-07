package flow

import "time"

type FlowView struct {
	Name      string    `json:"name"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func ToFlowView(flow *Flow) FlowView {
	return FlowView{
		Name:      flow.Name,
		UpdatedAt: flow.UpdatedAt,
	}
}
