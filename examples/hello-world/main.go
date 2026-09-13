package main

import (
	"fmt"
	"os"

	"github.com/slashbase/layerengine"
	"github.com/slashbase/layerengine/codegen"
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

	codegenerater, _ := codegen.NewCodeGen(apiKey, modelId)
	engine := layerengine.NewLayerEngine(codegenerater)

	data, err := os.ReadFile("./examples/hello-world/flow.yaml")
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
		"name": "Paras",
	}
	output, err := engine.RunFlow("hello_world", inputValues)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("---OUTPUT---")
	fmt.Println(output)
}
