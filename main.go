package main

import (
	"os"

	"github.com/paraswaykole/layerdotrun/internal/app"
	"github.com/paraswaykole/layerdotrun/pkg/config"
	"github.com/paraswaykole/layerdotrun/pkg/database"
)

var version = "v0.0.0"

func main() {
	config.Init(version)
	app := app.NewApp()
	defer app.CloseApp()
	database.Init(map[string]string{
		"default": os.Getenv("DATABASE"), // temp - figure out database conn string storage or config
	})
	app.StartApp()
}
