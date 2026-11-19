# layerengine

Simple LLM generated lua code running engine lib written in Golang.

## 🚧 WIP experimental project.

![maintenance-status: experimental](https://img.shields.io/badge/maintenance--status-experimental-palevioletred)

This project is under development and has experimental status and is not intended for serious or production use. Expect frequent changes and occasional breaking updates.

## How it works

You write a **flow** — a YAML file that breaks your task into a few small steps called **layers**. Each layer simply states what it takes in, what it does and what it gives out.

The engine then does three things:

1. **Reads your flow.** Validates the YAML and registers the flow by name.
2. **Gets an LLM to write the code.** For each layer, it sends the layer's description and its input/output contract to an LLM (OpenAI or Anthropic). The LLM replies with a small Lua function, which is compiled and stored with the layer.
3. **Runs the flow.** You pass in your inputs, the engine runs each layer in order, and feeds each layer's output or earlier inputs into the next layer as inputs — like an assembly line. The last layer's output is the flow's result.

For safety, every layer runs in its own private Lua environment with a whitelisted standard library and a 60-second timeout. A layer can't reach the filesystem, spawn a process, or hang the engine, and its variables can't leak into other layers — when a layer finishes, its environment is closed before the next one starts.

Generated code starts with pure computation. When a layer needs to do something real — call an API, need http client or database connection to read/write data — you register a **module**, whose functions the sandbox binds as a global named after the module (`weather.temperature(...)`); see [Modules](#modules).

## Quickstart

Requires Go 1.26 or newer — that is the floor in `go.mod`.

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

A flow is a YAML document with a `name`, `description`, typed `input`, and an ordered `layers` list. Each layer declares the inputs it consumes and the typed outputs it produces. A layer's input is either a flow-level input or the output of an earlier layer.

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
	"github.com/slashbase/layerengine/config"
)

func main() {
	// FromEnv picks a provider from OPENAI_API_KEY or ANTHROPIC_API_KEY.
	// For an explicit choice, build a config.Config{Model: ..., APIKey: ...}.
	cfg, err := config.FromEnv()
	if err != nil {
		panic(err)
	}
	generator, err := cfg.CodeGen()
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

### Engine API at a glance

| Method | What it does |
| --- | --- |
| `LoadSpec(spec string) error` | Validate a spec, generate every layer and load the flow — all or nothing |
| `GenerateLayers(flow *Flow) error` | Generate Lua for every layer and store it on the flow, without loading |
| `GenerateLayer(flow *Flow, index int) error` | Regenerate one layer, e.g. after a retry |
| `LoadLayers(layers []Layer) error` | Compile and load hand-written layers; no LLM involved |
| `LoadFlow(flow Flow) error` | Register a flow whose layers are already loaded |
| `Layer(name string) (*Layer, bool)` | Read back a loaded layer, including its generated `Code` |
| `RunFlow(name string, inputs map[string]any) (any, error)` | Run a flow with lenient input coercion |
| `RunFlowWithOptions(name string, inputs map[string]any, opts CoerceOptions) (any, error)` | The same, with a strict / keep-undeclared input policy |
| `RunLayer(name string, inputs []any) (any, error)` | Run one loaded layer on positional arguments |
| `GetModules() []modules.Module` | The module set every layer sandbox gets (a copy) |

`NewLayerEngine(generator)` needs an LLM; `NewBlankLayerEngine()` has none but loads the same built-in modules, so a hand-loaded layer runs against the sandbox the codegen prompt describes — `system.version()` resolves on both.

**Layer names are engine-global.** A flow's layers are keyed by name in one engine-wide map, so two flows cannot both own a layer called `format` with different code: loading the second fails and names the flow that already holds the name. Identical code is shared, which keeps re-loading the same spec idempotent. Layers loaded by hand through `LoadLayers` are unowned, and a later hand load may replace them.

## Modules

A layer's Lua starts with pure computation only — the standard library is deliberately narrow (see [Execution model & safety](#execution-model--safety)). Real work happens through a **module**: a small Go type that publishes a table of functions and describes itself so the LLM knows how to call it.

A module implements four methods:

```go
type Module interface {
	Name() string                    // the global it is bound to: "modname" → modname.<fn>(...)
	Init(l *lua.LState) *lua.LTable  // the functions to expose
	Usage() string                   // free-text; used by codegen as notes; rendered only when Manifest is zero
	Manifest() modules.Manifest      // used by codegen; the functions it exposes, as data
}
```

The manifest declares each function once, as data, instead of relying on prose the model has to parse:

```go
func (Weather) Manifest() modules.Manifest {
	return modules.Manifest{
		APIVersion: modules.APIVersion,
		Notes: "fetches live weather from live weather API",
		Functions: []modules.Function{{
			Name:    "temperature",
			Doc:     "Temperature in degrees Celsius for a city.",
			Params:  []modules.Param{{Name: "city", Type: "string"}},
			Returns: []modules.Return{{Name: "celsius", Type: "number"}},
		}},
	}
}
```

With a manifest, each function renders as a single call-shaped line, the way a module author would write it by hand — return names on the left, types inline, the doc after a dash:

```
celsius:number = weather.temperature(city:string) - Temperature in degrees Celsius for a city.
```

An argument with a `Default` renders as `limit?:number=10` and may be omitted by the generated code, whether or not `Optional` is set.

`Notes` renders once per module in the same shape: `weather - fetches live weather from live weather API.` A module's `Usage()` is only rendered when it declares nothing — the zero `modules.Manifest{}` — and then it appears under a `<name>` heading. Modules and functions are sorted, so the same module set always produces the same prompt. `modules.Validate(mods)` reports duplicate or unusable module names and malformed manifests.

Register modules on the engine before loading a flow:

```go
engine := layerengine.NewLayerEngine(generator)
if err := engine.LoadModules([]modules.Module{Module1{}, Module2{}}); err != nil {
	log.Fatal(err)
}
```

Two things happen. Each module table is bound as a **global named after the module** in every layer's sandbox — so `modname` is `modname.fnname(...)`, and each module's surface is rendered into the prompt sent to the LLM, so the generated code knows the calls exist, their arity, and what they return. `LoadModules` adds new modules to the engine alongside the built-in `system` module.

Because a module name is a global, it has to be free in the sandbox, so `LoadModules` checks it and can fail: a module called `string` would replace the standard library's string table, and one called `print` would replace a base-library function the prompt says is available — both rejected with the name in the error, and a rejected set leaves the engine unchanged. The reserved set is read off a real sandbox rather than hardcoded, so opening another library later cannot leave the check behind.

`LoadModules` also holds the manifest to what `Init` actually published. Every declared path has to be a function on the table the module returned, so a manifest that names `shapes.circle.area` while `Init` builds a flat table fails at registration with the full path in the error, instead of as a nil index inside a layer that names neither the module nor the manifest. A module with the zero manifest is skipped: there is no declaration to hold it to.

The [`external-modules`](examples/external-modules) example is a complete worked version: two modules over two live APIs, wired through three layers.

The [`shapes`](examples/shapes) example is the smallest one that shows grouping: a single module whose manifest names are paths, so one module answers `shapes.circle.area(radius)` and `shapes.rect.area(width, height)`. It also prints the rendered prompt, so you can see the four lines the model is given without spending a token on generation.

Keep a module's surface tight. It should own its endpoint and take only the arguments the layer supplies, so generated code can never reach a host you didn't choose — that is the difference between exposing a capability and exposing the network.

## Execution model & safety

- **Per-layer Lua isolation.** Each `runLayer` call spins up a fresh `*lua.LState` and `Close()`s it (and cancels its context) before returning. Globals set by one layer cannot reach the next.
- **Per-layer timeout.** `layerExecutionTimeout` is 60s; a layer that loops forever is killed, not allowed to hang the flow.
- **Codegen timeout.** LLM requests use a 60s timeout.
- **Concurrency safety.** The engine guards its layer/flow registries with an `RWMutex`; `RunFlow` takes defensive copies of the layer slice so a concurrent `LoadSpec` can't mutate a running flow's view.

**What the layer sandbox covers, and what it doesn't.** Each layer runs with only base, string, table, math and the module tables you register. os, io, debug and package are not opened, and the dynamic loaders (load, loadstring, dofile, getfenv, setfenv, collectgarbage) are removed. Generated code therefore cannot read files, spawn processes, read the environment, or exit the process — host capability is reachable only through a module you expose deliberately as a global in the sandbox.

What is still not covered: there is no hard memory cap. A watchdog samples the process heap every 50 ms and cancels a layer that grows past ~256 MiB, but it is approximate and process-wide, and it is only meaningful for sequential execution — a single very large allocation can overshoot before the poller observes it. True isolation (a kernel-enforced address-space limit) would mean running layers in a child process, which the engine does not do yet. That, together with the experimental status, is why this is still not for production use.

## Examples

| Example            | What it shows                                             |
| ------------------ | --------------------------------------------------------- |
| `hello-world`      | smallest single-layer hello world flow                    |
| `arithmetic`       | multi-layer pipeline with an optional input               |
| `fizzbuzz`         | branching-ish logic inside generated Lua                  |
| `manual-run`       | engine loads the flows manually without LoadSpec method   |
| `external-modules` | two custom modules calling live HTTP APIs across three layers |
| `typed-inputs`     | input coercion: lenient conversion, Strict rejection, undeclared arguments |
| `shapes`           | one module publishing several groups: `shapes.circle.area(...)` and `shapes.rect.area(...)` |

Each example lives under `examples/<name>/`. Most are a `flow.yaml` and a `main.go`; `external-modules` and `shapes` also carry the module implementations they register.

## Contributing

This is an experiment — contributions that keep the core small and the contract clear are very welcome. Good first patches:

- Add a `ModelID` for a model that isn't in the registry yet (one line in `codegen/models.go`).
- Write a new example flow under `examples/` that shows off something the current toys don't.
- Add a **system-level** module under `modules/` and document what it exposes to generated Lua. Example-specific modules don't belong in the core — register them from your own code with `LoadModules` (see [`external-modules`](examples/external-modules)) and publish them as separate repositories.
- Add a test for a codegen prompt or a value-conversion edge case.

Before opening a PR: `go test ./...` should pass. Keep generated Lua correct and the spec
format backward-compatible, or call out a breaking change explicitly.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for license rights and limitations.
