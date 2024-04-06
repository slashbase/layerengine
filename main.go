package main

import "github.com/paraswaykole/layerdotrun/internal/app"

func main() {
	app := app.NewApp()
	defer app.CloseApp()
	app.StartApp()
}
