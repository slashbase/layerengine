package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/paraswaykole/layerdotrun/internal/global"
	"github.com/paraswaykole/layerdotrun/internal/layer"
	"github.com/paraswaykole/layerdotrun/internal/store"
	"github.com/paraswaykole/layerdotrun/pkg/layerengine"
)

type App struct {
	Store  *store.Store
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
	store := store.NewStore([]string{layer.BucketName})
	layerEngine := layerengine.NewLayerEngine()

	layerDao := layer.NewLayerDAO(store)
	layerService := layer.NewLayerService(layerDao, layerEngine)
	layerController := layer.NewLayerController(layerService)

	globalController := global.NewGlobalController(layerService, layerEngine)

	layerService.LoadAllLayers()

	app := App{
		Store: store,
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
func (app App) CloseApp() {
	app.Store.Close()
}
