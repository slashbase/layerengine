package main

import (
	"github.com/paraswaykole/layerdotrun/internal/app"
	"github.com/paraswaykole/layerdotrun/pkg/config"
)

var version = "v0.0.0"

func main() {
	config.Init(version)
	app := app.NewApp()
	defer app.CloseApp()
	app.StartApp()
}
