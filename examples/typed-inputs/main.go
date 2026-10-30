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
		modelId = codegen.GPT6Luna
	}
	if ANTHROPIC_API_KEY != "" {
		apiKey = ANTHROPIC_API_KEY
		modelId = codegen.ClaudeSonnet5_5
	}

	codegenerater, err := codegen.NewCodeGen(apiKey, modelId)
	if err != nil {
		fmt.Println("Error initializing code generator:", err)
		return
	}
	engine := layerengine.NewLayerEngine(codegenerater)

	data, err := os.ReadFile("./examples/typed-inputs/flow.yaml")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	err = engine.LoadSpec(string(data))
	if err != nil {
		fmt.Println(err)
		return
	}

	// The generated Lua is what the flow's declared types are written for. The
	// layer compares count against 10 and branches on enabled, so it needs a
	// number and a boolean — not the strings a JSON caller would send.
	if layer, ok := engine.Layer("describe_inputs"); ok {
		fmt.Println("---GENERATED LAYER---")
		fmt.Println(layer.Code)
	}

	// Every value here is what an LLM tool call delivers: numbers and booleans
	// arrive as strings, and label arrives as a number. Without coercion the
	// comparison on count raises "attempt to compare string with number", and
	// the string "false" is truthy, so the branch silently flips.
	stringlyTyped := map[string]any{
		"count":   "7",
		"enabled": "false",
		"label":   42,
	}

	fmt.Println("---LENIENT (RunFlow)---")
	output, err := engine.RunFlow("typed_inputs", stringlyTyped)
	if err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Println(output)
	}

	// The same arguments under Strict: a Go caller usually wants the mismatch
	// reported rather than converted.
	fmt.Println("---STRICT (RunFlowWithOptions)---")
	output, err = engine.RunFlowWithOptions("typed_inputs", stringlyTyped, layerengine.CoerceOptions{Strict: true})
	if err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Println(output)
	}

	// Arguments the flow does not declare are dropped, and reported through
	// OnUndeclared, so a caller can log what it sent that went nowhere.
	fmt.Println("---UNDECLARED ARGUMENT---")
	withExtra := map[string]any{
		"count":   "7",
		"enabled": "false",
		"label":   "demo",
		"extra":   "this is not a declared input",
	}
	output, err = engine.RunFlowWithOptions("typed_inputs", withExtra, layerengine.CoerceOptions{
		OnUndeclared: func(name string) {
			fmt.Printf("dropped undeclared argument: %q\n", name)
		},
	})
	if err != nil {
		fmt.Println("error:", err)
	} else {
		fmt.Println(output)
	}
}
