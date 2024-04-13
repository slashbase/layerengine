package api

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type ApiController struct {
	apiService *ApiService
}

func NewApiController(apiService *ApiService) *ApiController {
	return &ApiController{apiService: apiService}
}

func (ctrl ApiController) Get(c *fiber.Ctx) error {
	name := c.Params("name", "")
	if name == "" {
		return c.SendStatus(http.StatusBadRequest)
	}
	api, err := ctrl.apiService.GetApi(name)
	if err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}
	return c.JSON(map[string]interface{}{
		"success": true,
		"result":  ToApiView(api),
	})
}

func (ctrl ApiController) Update(c *fiber.Ctx) error {
	var body UpdateApiRequest
	if err := c.BodyParser(&body); err != nil {
		return c.SendStatus(http.StatusBadRequest)
	}
	api, err := ctrl.apiService.UpdateApi(body)
	if err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}
	return c.JSON(map[string]interface{}{
		"success": true,
		"result":  ToApiView(api),
	})
}
