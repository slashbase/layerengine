package app

import (
	"github.com/gofiber/fiber/v2"
	"github.com/paraswaykole/layerdotrun/internal/api"
	"github.com/paraswaykole/layerdotrun/internal/flow"
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
	Flow struct {
		Controller *flow.FlowController
		Service    *flow.FlowService
		Dao        *flow.FlowDAO
	}
	Api struct {
		Controller *api.ApiController
		Service    *api.ApiService
		Dao        *api.ApiDAO
	}
	LayerEngine *layerengine.LayerEngine
}

func NewApp() *App {
	store := store.NewStore([]string{layer.BucketName, flow.BucketName, api.BucketName})
	layerEngine := layerengine.NewLayerEngine()

	layerDao := layer.NewLayerDAO(store)
	layerService := layer.NewLayerService(layerDao, layerEngine)
	layerController := layer.NewLayerController(layerService)

	flowDao := flow.NewFlowDAO(store)
	flowService := flow.NewFlowService(flowDao, layerDao, layerEngine)
	flowController := flow.NewFlowController(flowService)

	apiDao := api.NewApiDAO(store)
	apiService := api.NewApiService(apiDao)
	apiController := api.NewApiController(apiService)

	globalService := global.NewGlobalService(apiService, layerEngine)
	globalController := global.NewGlobalController(globalService, layerService, layerEngine)

	layerService.LoadAllLayers()
	flowService.LoadAllFlows()

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
		Flow: struct {
			Controller *flow.FlowController
			Service    *flow.FlowService
			Dao        *flow.FlowDAO
		}{
			Controller: flowController,
			Service:    flowService,
			Dao:        flowDao,
		},
		Api: struct {
			Controller *api.ApiController
			Service    *api.ApiService
			Dao        *api.ApiDAO
		}{
			Controller: apiController,
			Service:    apiService,
			Dao:        apiDao,
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
