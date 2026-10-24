package main

import (
	"fmt"
	"os"

	"github.com/slashbase/layerengine"
	"github.com/slashbase/layerengine/codegen"
	"github.com/slashbase/layerengine/modules"
)

func main() {

	OPENAI_API_KEY := os.Getenv("OPENAI_API_KEY")
	ANTHROPIC_API_KEY := os.Getenv("ANTHROPIC_API_KEY")

	var apiKey string
	var modelId codegen.ModelID
	if OPENAI_API_KEY != "" {
		apiKey = OPENAI_API_KEY
		modelId = codegen.GPT6Luna
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
