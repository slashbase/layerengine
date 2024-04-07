package flow

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type FlowController struct {
	flowService *FlowService
}

func NewFlowController(flowService *FlowService) *FlowController {
	return &FlowController{flowService: flowService}
}

func (ctrl FlowController) Get(c *fiber.Ctx) error {
	name := c.Params("name", "")
	if name == "" {
		return c.SendStatus(http.StatusBadRequest)
	}
	flow, err := ctrl.flowService.GetFlow(name)
	if err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}
	return c.JSON(map[string]interface{}{
		"success": true,
		"result":  ToFlowView(flow),
	})
}

func (ctrl FlowController) Update(c *fiber.Ctx) error {
	var body UpdateFlowRequest
	if err := c.BodyParser(&body); err != nil {
		return c.SendStatus(http.StatusBadRequest)
	}
	layer, err := ctrl.flowService.UpdateFlow(body)
	if err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}
	return c.JSON(map[string]interface{}{
		"success": true,
		"result":  ToFlowView(layer),
	})
}
