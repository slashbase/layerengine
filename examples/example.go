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

	codegenerater, _ := codegen.NewCodeGen(OPENAI_API_KEY, ANTHROPIC_API_KEY, codegen.OPENAI_GPT3DOT5_TURBO)
	engine := layerengine.NewLayerEngine(codegenerater)

	data, err := os.ReadFile("./template.yaml")
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
		"number_a": 3,
		"number_b": 5,
	}
	output, err := engine.RunFlow("template_test", inputValues)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("---OUTPUT---")
	fmt.Println(output)
}
