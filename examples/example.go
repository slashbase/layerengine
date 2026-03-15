package examples

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

	engine := layerengine.NewLayerEngine()
	codegenerater, _ := codegen.NewCodeGen(OPENAI_API_KEY, ANTHROPIC_API_KEY, codegen.OPENAI_GPT3DOT5_TURBO)

	data, err := os.ReadFile("./template.yaml")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	flow, err := validator.Run(data)
	if err != nil {
		fmt.Println(err)
		return
	}

	layers := []layerengine.Layer{}
	layerNames := []string{}
	for _, layer := range flow.Layers {
		code, err := codegenerater.GenerateLayerFunction(layer.Name, layer.Description, layer.Input, layer.Output)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Code: ", code)

		layerInstance := layerengine.Layer{
			Name:    layer.Name,
			FnProto: nil,
			Input:   layer.Input,
			Output:  layer.Output,
			Code:    code,
		}
		layers = append(layers, layerInstance)
		layerNames = append(layerNames, layer.Name)
	}

	engine.LoadLayers(layers)
	engine.LoadFlow(map[string][]string{
		flow.Name: layerNames,
	})

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
