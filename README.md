# layerengine

Simple AI generated code running engine based on LLMs & Lua written in Golang

## 🚧 WIP experimental project.

![maintenance-status: experimental](https://img.shields.io/badge/maintenance--status-experimental-palevioletred)

This project is under development and has experimental status and is not intended for serious or production use. Expect frequent changes and occasional breaking updates.

## How it works

You write a **flow** — a YAML file that breaks your task into a few small steps called **layers**. Each layer simply states what it takes in, what it does and what it gives out.

The engine then does three things:

1. **Reads your flow.** Validates the YAML and registers the flow by name.
2. **Gets an LLM to write the code.** For each layer, it sends the layer's description and its input/output contract to an LLM (OpenAI or Anthropic). The LLM replies with a small Lua function, which is compiled and stored with the layer.
3. **Runs the flow.** You pass in your inputs, the engine runs each layer in order, and feeds each layer's output or earlier inputs into the next layer as inputs — like an assembly line. The last layer's output is the flow's result.

For safety, every layer runs in its own private Lua environment with a 15-second timeout. A layer can't hang the engine, and its variables can't leak into other layers — when a layer finishes, its environment is closed before the next one starts.

## Quickstart

```bash
# 1. Clone
git clone https://github.com/slashbase/layerengine.git
cd layerengine

# 2. Set a provider key (either one works)
export OPENAI_API_KEY="sk-..."
#   or
export ANTHROPIC_API_KEY="sk-ant-..."

# 3. Run the arithmetic example
go run ./examples/arithmetic
```

Expected output:

```
---OUTPUT---
map[amount_string:Your final amount is ₹800]
```

## Writing a flow spec

A flow is a YAML document with a `name`, `description`, typed `input`, and an ordered`layers` list. Each layer declares the inputs it consumes and the typed outputs it produces. A layer's input is either a flow-level input or the output of an earlier layer.

```yaml
name: arithmetic
description: takes two numbers, adds them, multiplies the result by 100 or given multiplier, and returns a formatted rupee amount string.
input:
  - number_a:
      type: number
      description: first number
  - number_b:
      type: number
      description: second number
  - multipier:
      type: number
      description: multipier number
      optional: true
layers:
  - name: add_numbers
    description: adds number_a and number_b together
    input:
      - number_a
      - number_b
    output:
      - sum_result:
          type: number
          description: the sum of number_a and number_b
  - name: multiply_numbers
    description: multiplies the sum by multipier if given or 100.
    input:
      - sum_result
      - multipier
    output:
      - multiplied_result:
          type: number
          description: the product of sum_result and multipier (or 100 if multipier is absent)
  - name: format_amount
    description: formats the multiplied result into the string "Your final amount is ₹{number}"
    input:
      - multiplied_result
    output:
      - amount_string:
          type: string
          description: the formatted rupee amount string
```

## Using it as a library

```bash
go get github.com/slashbase/layerengine
```

```go
package main

import (
	"fmt"
	"os"

	"github.com/slashbase/layerengine"
	"github.com/slashbase/layerengine/codegen"
)

func main() {
	// Provide Provider Model API key and pick a model.
	generator, err := codegen.NewCodeGen(os.Getenv("OPENAI_API_KEY"), codegen.GPT4o)
	if err != nil {
		panic(err)
	}

	engine := layerengine.NewLayerEngine(generator)

	spec, _ := os.ReadFile("flow.yaml")
	if err := engine.LoadSpec(string(spec)); err != nil {
		panic(err)
	}

	out, err := engine.RunFlow("flow_name", map[string]any{
		"input_1": 10,
		"input_2": "string value",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(out)
}
```

## Execution model & safety

- **Per-layer Lua isolation.** Each `runLayer` call spins up a fresh `*lua.LState` and `Close()`s it (and cancels its context) before returning. Globals set by one layer cannot reach the next.
- **Per-layer timeout.** `layerExecutionTimeout` is 15s; a layer that loops forever is killed, not allowed to hang the flow.
- **Codegen timeout.** LLM requests use a 60s timeout.
- **Concurrency safety.** The engine guards its layer/flow registries with an `RWMutex`; `RunFlow` takes defensive copies of the layer slice so a concurrent `LoadSpec` can't mutate a running flow's view.

What is *not* sandboxed today: the Lua layers run with the full gopher-lua runtime (no capability restrictions, no I/O syscall filtering, no memory cap beyond the timeout). This is fine for trusted local use and demos; it is the main reason for the "not for production" warning above.

## Examples

| Example        | What it shows                                             |
| -------------- | --------------------------------------------------------- |
| `hello-world`  | smallest single-layer hello world flow                    |
| `arithmetic`   | multi-layer pipeline with an optional input               |
| `fizzbuzz`     | branching-ish logic inside generated Lua                  |
| `manual-run`   | engine loads the flows manually without LoadSpec method   |

Each example lives under `examples/<name>/` with its own `flow.yaml` and `main.go`.

## Contributing

This is an experiment — contributions that keep the core small and the contract clear are very welcome. Good first patches:

- Add a `ModelID` for a model that isn't in the registry yet (one line in `codegen/models.go`).
- Write a new example flow under `examples/` that shows off something the current toys don't.
- Add a module under `modules/` and document what it exposes to generated Lua. Modules that are custom and not system level are not welcome in the main layerengine. Custom modules can be published as seperate repositories.
- Add a test for a codegen prompt or a value-conversion edge case.

Before opening a PR: `go test ./...` should pass. Keep generated Lua correct and the spec
format backward-compatible, or call out a breaking change explicitly.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for license rights and limitations.