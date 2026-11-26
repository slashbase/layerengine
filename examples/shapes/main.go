package main

import (
	"fmt"
	"os"

	"github.com/slashbase/layerengine"
	"github.com/slashbase/layerengine/codegen/prompts"
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

	// One module, two groups. Generated code calls shapes.circle.area(...) and
	// shapes.rect.area(...): the group is a table in the sandbox, so its name is
	// part of the path rather than a prefix glued onto the function name.
	if err := engine.LoadModules([]modules.Module{Shapes{}}); err != nil {
		fmt.Println("Error registering modules:", err)
		return
	}

	// What the model is shown. Four lines, contiguous per shape, because the
	// manifest names are paths and the prompt sorts by full name.
	fmt.Println("---MODULE SURFACE---")
	fmt.Println(prompts.GetModuleUsagePrompt([]modules.Module{Shapes{}}))

	data, err := os.ReadFile("./examples/shapes/flow.yaml")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	if err := engine.LoadSpec(string(data)); err != nil {
		fmt.Println(err)
		return
	}

	inputValues := map[string]any{
		"radius": 2.0,
		"width":  3.0,
		"height": 4.0,
	}
	output, err := engine.RunFlow("shapes_report", inputValues)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("---OUTPUT---")
	fmt.Println(output)
}
