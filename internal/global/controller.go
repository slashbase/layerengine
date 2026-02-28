package global

import (
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/slashbase/layerengine/internal/layer"
	"github.com/slashbase/layerengine/pkg/layerengine"
)

type GlobalController struct {
	globalService *GlobalService
	layerService  *layer.LayerService
	layerEngine   *layerengine.LayerEngine
}

func NewGlobalController(globalService *GlobalService, layerService *layer.LayerService, layerEngine *layerengine.LayerEngine) *GlobalController {
	return &GlobalController{globalService: globalService, layerService: layerService, layerEngine: layerEngine}
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

func (ctrl GlobalController) RunApi(c *fiber.Ctx) error {
	path := c.Path()
	method := c.Method()

	var body map[string]interface{}
	c.BodyParser(&body)

	result, err := ctrl.globalService.RunApi(path, method, body)
	if err != nil {
		log.Printf("RunApi error: %v\n", err)
		return c.SendStatus(http.StatusInternalServerError)
	}

	return c.JSON(result)
}
