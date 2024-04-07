package app

import (
	"github.com/gofiber/fiber/v2"
)

func (app App) setupRoutes(server *fiber.App) {
	server.Get("/health", healthCheck)
	server.Post("/run", app.Global.Controller.Run)

	api := server.Group("/api/v1")
	{
		layerGroup := api.Group("layer")
		{
			layerGroup.Get("/:name", app.Layer.Controller.Get)
			layerGroup.Post("/update", app.Layer.Controller.Update)
		}
		flowGroup := api.Group("flow")
		{
			flowGroup.Get("/:name", app.Flow.Controller.Get)
			flowGroup.Post("/update", app.Flow.Controller.Update)
		}
	}
}

func healthCheck(c *fiber.Ctx) error {
	return c.JSON(map[string]interface{}{
		"success": true,
	})
}
