package main

import (
	"fmt"
	"os"

	"github.com/slashbase/layerengine"
	"github.com/slashbase/layerengine/config"
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

	data, err := os.ReadFile("./examples/fizzbuzz/flow.yaml")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	err = engine.LoadSpec(string(data))
	if err != nil {
		fmt.Println(err)
		return
	}

	inputValues := map[string]any{
		"count": 15,
	}
	output, err := engine.RunFlow("fizzbuzz", inputValues)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("---OUTPUT---")
	fmt.Println(output)
}
