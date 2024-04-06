package app

import (
	"github.com/gofiber/fiber/v2"
)

func (app App) setupRoutes(server *fiber.App) {
	server.Get("/health", healthCheck)
	server.Post("/run", app.Global.Controller.Run)

	api := server.Group("/api/v1")
	{
		api.Get("/layer/:name", app.Layer.Controller.Get)
		api.Post("/layer/update", app.Layer.Controller.Update)
	}
}

func healthCheck(c *fiber.Ctx) error {
	return c.JSON(map[string]interface{}{
		"success": true,
	})
}
