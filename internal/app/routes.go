package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/slashbase/layerengine/pkg/config"
)

func (app App) setupRoutes(server *fiber.App) {
	server.Get("/health", healthCheck)
	server.Post("/run", app.Global.Controller.Run)

	admin := server.Group("/admin")
	{
		layerGroup := admin.Group("layer")
		{
			layerGroup.Get("/:name", app.Layer.Controller.Get)
			layerGroup.Post("/update", app.Layer.Controller.Update)
		}
		flowGroup := admin.Group("flow")
		{
			flowGroup.Get("/:name", app.Flow.Controller.Get)
			flowGroup.Post("/update", app.Flow.Controller.Update)
		}
		apiGroup := admin.Group("api")
		{
			apiGroup.Get("/:name", app.Api.Controller.Get)
			apiGroup.Post("/update", app.Api.Controller.Update)
		}
	}

	server.All("/*", app.Global.Controller.RunApi)

}

func healthCheck(c *fiber.Ctx) error {
	return c.JSON(map[string]interface{}{
		"success": true,
		"version": config.Get().Version,
	})
}
