package layer

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type LayerController struct {
	layerService *LayerService
}

func NewLayerController(layerService *LayerService) *LayerController {
	return &LayerController{layerService: layerService}
}

func (ctrl LayerController) Update(c *fiber.Ctx) error {
	var body struct {
		Name  string             `json:"name"`
		Layer UpdateLayerRequest `json:"layer"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.SendStatus(http.StatusBadRequest)
	}

	return nil
}
