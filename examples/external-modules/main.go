package main

import (
	"fmt"
	"os"

	"github.com/slashbase/layerengine"
	"github.com/slashbase/layerengine/config"
	"github.com/slashbase/layerengine/modules"
)

func main() {

	cfg, err := config.FromEnv()
	if err != nil {
		fmt.Println("Error reading config:", err)
		return
	}
	codegenerater, err := cfg.CodeGen()
	if err != nil {
		fmt.Println("Error initializing code generator:", err)
		return
	}
	engine := layerengine.NewLayerEngine(codegenerater)

	// Register this example's modules. They become layer.ipinfo and
	// layer.weather, and their Usage() is rendered into the codegen prompt,
	// so the model knows the calls exist and what they return.
	engine.LoadModules([]modules.Module{IPInfo{}, Weather{}})

	data, err := os.ReadFile("./examples/external-modules/flow.yaml")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	if err := engine.LoadSpec(string(data)); err != nil {
		fmt.Println(err)
		return
	}

	// The flow takes no inputs — the location comes from the IP lookup.
	output, err := engine.RunFlow("weather_report", map[string]any{})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("---OUTPUT---")
	fmt.Println(output)
}
