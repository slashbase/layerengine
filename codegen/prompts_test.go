package codegen

import (
	"strings"
	"testing"

	"github.com/slashbase/layerengine/modules"
	lua "github.com/yuin/gopher-lua"
)

// demoModule declares a manifest: each function must render as a single
// call-shaped line, with its Usage() text left out (the manifest supersedes it).
type demoModule struct{}

func (demoModule) Name() string { return "demo" }

func (demoModule) Usage() string {
	return "x = demo.get(\"k\")\n-- free text notes are superseded by the manifest."
}

func (demoModule) Init(l *lua.LState) *lua.LTable {
	t := l.NewTable()
	t.RawSetString("get", l.NewFunction(func(*lua.LState) int { return 0 }))
	return t
}

func (demoModule) Manifest() modules.Manifest {
	return modules.Manifest{
		APIVersion: modules.APIVersion,
		Notes:      "values are cached for 60s",
		Functions: []modules.Function{
			{
				Name:    "set",
				Doc:     "Store a value.",
				Params:  []modules.Param{{Name: "key", Type: "string"}, {Name: "value", Type: "any", Optional: true}},
				Returns: []modules.Return{{Name: "ok", Type: "boolean"}},
			},
			{
				Name:    "get",
				Doc:     "Read a value.",
				Params:  []modules.Param{{Name: "key", Type: "string"}},
				Returns: []modules.Return{{Name: "value", Type: "table{city:string}"}},
			},
		},
	}
}

// legacyModule declares nothing: it must render from Usage() alone, under a
// <name> heading.
type legacyModule struct{}

func (legacyModule) Name() string { return "legacy" }

func (legacyModule) Usage() string { return "v = legacy.ping()" }

func (legacyModule) Init(l *lua.LState) *lua.LTable { return l.NewTable() }

// Manifest returns the zero value, the documented way to declare no surface.
func (legacyModule) Manifest() modules.Manifest { return modules.Manifest{} }

// brokenModule has a manifest that fails validation (duplicate function names),
// so it must fall back to Usage() rather than render a bad contract.
type brokenModule struct{}

func (brokenModule) Name() string { return "broken" }

func (brokenModule) Usage() string { return "v = broken.ping()" }

func (brokenModule) Init(l *lua.LState) *lua.LTable { return l.NewTable() }

func (brokenModule) Manifest() modules.Manifest {
	return modules.Manifest{
		Functions: []modules.Function{{Name: "ping"}, {Name: "ping"}},
	}
}

func TestGetModuleUsagePromptRendersManifest(t *testing.T) {
	got := GetModuleUsagePrompt([]modules.Module{demoModule{}})

	for _, want := range []string{
		"value:table{city:string} = demo.get(key:string) - Read a value.",
		"ok:boolean = demo.set(key:string, value?:any) - Store a value.",
		"demo - values are cached for 60s",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("prompt is missing %q\ngot:\n%s", want, got)
		}
	}

	// The manifest supersedes Usage(): no notes block, no separate signature block.
	for _, unwanted := range []string{"usage notes:", "  get(key:string)", "  -- x = demo.get"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("prompt should not contain %q\ngot:\n%s", unwanted, got)
		}
	}

	// Functions render in name order, not author order.
	if strings.Index(got, "demo.get(") > strings.Index(got, "demo.set(") {
		t.Errorf("functions are not sorted by name:\n%s", got)
	}
}

func TestGetModuleUsagePromptFallsBackToUsage(t *testing.T) {
	got := GetModuleUsagePrompt([]modules.Module{legacyModule{}})

	for _, want := range []string{"legacy\n  -- v = legacy.ping()"} {
		if !strings.Contains(got, want) {
			t.Errorf("prompt is missing %q\ngot:\n%s", want, got)
		}
	}
}

func TestGetModuleUsagePromptIgnoresInvalidManifest(t *testing.T) {
	got := GetModuleUsagePrompt([]modules.Module{brokenModule{}})

	if !strings.Contains(got, "  -- v = broken.ping()") {
		t.Errorf("invalid manifest should fall back to Usage():\n%s", got)
	}
	if strings.Contains(got, "broken.ping() - ") {
		t.Errorf("invalid manifest should not render a call line:\n%s", got)
	}
}

func TestGetModuleUsagePromptIsDeterministic(t *testing.T) {
	forward := GetModuleUsagePrompt([]modules.Module{demoModule{}, legacyModule{}})
	reverse := GetModuleUsagePrompt([]modules.Module{legacyModule{}, demoModule{}})

	if forward != reverse {
		t.Errorf("prompt depends on module order\nforward:\n%s\nreverse:\n%s", forward, reverse)
	}
	if strings.Index(forward, "demo.get(") > strings.Index(forward, "\nlegacy") {
		t.Errorf("modules are not sorted by name:\n%s", forward)
	}
}

// defaultsModule exercises argument defaults: a default must render, and must
// make the argument omittable even when Optional is unset.
type defaultsModule struct{}

func (defaultsModule) Name() string { return "defaults" }

func (defaultsModule) Usage() string { return "" }

func (defaultsModule) Init(l *lua.LState) *lua.LTable { return l.NewTable() }

func (defaultsModule) Manifest() modules.Manifest {
	return modules.Manifest{
		APIVersion: modules.APIVersion,
		Functions: []modules.Function{{
			Name:    "list",
			Doc:     "List items.",
			Params:  []modules.Param{{Name: "limit", Type: "number", Default: "10"}},
			Returns: []modules.Return{{Name: "items", Type: "table"}},
		}},
	}
}

func TestGetModuleUsagePromptRendersDefault(t *testing.T) {
	got := GetModuleUsagePrompt([]modules.Module{defaultsModule{}})

	want := "items:table = defaults.list(limit?:number=10) - List items."
	if !strings.Contains(got, want) {
		t.Fatalf("prompt is missing %q\ngot:\n%s", want, got)
	}
}
