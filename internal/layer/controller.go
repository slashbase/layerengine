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

func (ctrl LayerController) Get(c *fiber.Ctx) error {
	name := c.Params("name", "")
	if name == "" {
		return c.SendStatus(http.StatusBadRequest)
	}
	layer, err := ctrl.layerService.GetLayer(name)
	if err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}
	return c.JSON(map[string]interface{}{
		"success": true,
		"result":  ToLayerView(layer),
	})
}

func (ctrl LayerController) Update(c *fiber.Ctx) error {
	var body UpdateLayerRequest
	if err := c.BodyParser(&body); err != nil {
		return c.SendStatus(http.StatusBadRequest)
	}
	layer, err := ctrl.layerService.UpdateLayer(body)
	if err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}
	return c.JSON(map[string]interface{}{
		"success": true,
		"result":  ToLayerView(layer),
	})
}
