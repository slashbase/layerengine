package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/paraswaykole/layerdotrun/internal/global"
	"github.com/paraswaykole/layerdotrun/internal/layer"
	"github.com/paraswaykole/layerdotrun/pkg/layerengine"
)

type App struct {
	Global struct {
		Controller *global.GlobalController
	}
	Layer struct {
		Controller *layer.LayerController
		Service    *layer.LayerService
		Dao        *layer.LayerDAO
	}
	LayerEngine *layerengine.LayerEngine
}

func NewApp() *App {
	layerDao := layer.NewLayerDAO()
	layerEngine := layerengine.NewLayerEngine()
	layerService := layer.NewLayerService(layerDao)
	layerController := layer.NewLayerController(layerService)

	globalController := global.NewGlobalController(layerService, layerEngine)

	app := App{
		Global: struct {
			Controller *global.GlobalController
		}{
			Controller: globalController,
		},
		Layer: struct {
			Controller *layer.LayerController
			Service    *layer.LayerService
			Dao        *layer.LayerDAO
		}{
			Controller: layerController,
			Service:    layerService,
			Dao:        layerDao,
		},
		LayerEngine: layerEngine,
	}
	return &app
}

func (app App) StartApp() {
	server := fiber.New()
	// server.Use(cors.New(cors.Config{
	// 	AllowOrigins:     "*",
	// 	AllowCredentials: true,
	// 	AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
	// }))
	app.setupRoutes(server)
	server.Listen(":3000")
}
