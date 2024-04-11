package global

import (
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/paraswaykole/layerdotrun/internal/layer"
	"github.com/paraswaykole/layerdotrun/pkg/layerengine"
)

type GlobalController struct {
	layerService *layer.LayerService
	layerEngine  *layerengine.LayerEngine
}

func NewGlobalController(layerService *layer.LayerService, layerEngine *layerengine.LayerEngine) *GlobalController {
	return &GlobalController{layerService: layerService, layerEngine: layerEngine}
}

func (ctrl GlobalController) Run(c *fiber.Ctx) error {
	var body struct {
		Name        string              `json:"name"`
		Type        layerengine.RunType `json:"type"`
		InputValues []interface{}       `json:"inputValues"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.SendStatus(http.StatusBadRequest)
	}

	result, err := ctrl.layerEngine.Run(body.Type, body.Name, body.InputValues)
	if err != nil {
		log.Printf("Run error: %v\n", err)
		return c.SendStatus(http.StatusInternalServerError)
	}

	return c.JSON(map[string]interface{}{
		"success": true,
		"result":  result,
	})
}
