package main

import (
	"fmt"
	"os"

	"github.com/slashbase/layerengine"
	"github.com/slashbase/layerengine/codegen"
	"github.com/slashbase/layerengine/validator"
)

func main() {

	OPENAI_API_KEY := os.Getenv("OPENAI_API_KEY")
	ANTHROPIC_API_KEY := os.Getenv("ANTHROPIC_API_KEY")

	var apiKey string
	var modelId codegen.ModelID
	if OPENAI_API_KEY != "" {
		apiKey = OPENAI_API_KEY
		modelId = codegen.GPT3Dot5Turbo
	}
	if ANTHROPIC_API_KEY != "" {
		apiKey = ANTHROPIC_API_KEY
		modelId = codegen.ClaudeSonnet4Dot5
	}

	codegenerater, err := codegen.NewCodeGen(apiKey, modelId)
	if err != nil {
		fmt.Println("Error initializing code generator:", err)
		return
	}
	engine := layerengine.NewLayerEngine(codegenerater)

	data, err := os.ReadFile("./examples/manual-run/flow.yaml")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	f, err := validator.Run(data)
	if err != nil {
		return
	}

	var flow layerengine.Flow
	layerengine.MapToStruct(validator.StructToMap(f), &flow)

	err = engine.GenerateLayers(&flow)
	if err != nil {
		return
	}

	layerNames := make([]string, len(flow.Layers))
	for i, layer := range flow.Layers {
		layerNames[i] = layer.Name
	}

	if err := engine.LoadLayers(flow.Layers); err != nil {
		fmt.Println("Error loading layers:", err)
		return
	}
	if err := engine.LoadFlow(flow); err != nil {
		fmt.Println("Error loading flow:", err)
		return
	}

	inputValues := map[string]any{
		"text": "layer-engine ",
	}
	output, err := engine.RunFlow("string_pipeline", inputValues)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("---OUTPUT---")
	fmt.Println(output)
}
