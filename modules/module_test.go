package modules_test

import (
	"testing"

	"github.com/slashbase/layerengine/modules"
	lua "github.com/yuin/gopher-lua"
)

// These tests use the modules package aliases on purpose: that is the path a
// module author takes, so it should stay the only import they need.

// stubModule is a Module with no manifest, used to check set-level validation.
type stubModule struct{}

func (stubModule) Name() string { return "stub" }

func (stubModule) Usage() string { return "stub()" }

func (stubModule) Init(l *lua.LState) *lua.LTable { return l.NewTable() }

func (stubModule) Manifest() modules.Manifest { return modules.Manifest{} }

func validManifest() modules.Manifest {
	return modules.Manifest{
		APIVersion: modules.APIVersion,
		Notes:      "units are metric",
		Functions: []modules.Function{
			{
				Name:    "current",
				Doc:     "Current reading.",
				Params:  []modules.Param{{Name: "latitude", Type: "number"}},
				Returns: []modules.Return{{Name: "reading", Type: "table{value:number}"}},
			},
		},
	}
}

func TestManifestValidateAcceptsValid(t *testing.T) {
	if err := validManifest().Validate(); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
}

func TestManifestValidateRejects(t *testing.T) {
	cases := map[string]func(*modules.Manifest){
		"no functions":        func(m *modules.Manifest) { m.Functions = nil },
		"unsupported version": func(m *modules.Manifest) { m.APIVersion = 99 },
		"unnamed function":    func(m *modules.Manifest) { m.Functions[0].Name = "" },
		"qualified function":  func(m *modules.Manifest) { m.Functions[0].Name = "layer.current" },
		"duplicate function": func(m *modules.Manifest) {
			m.Functions = append(m.Functions, m.Functions[0])
		},
		"unnamed param":  func(m *modules.Manifest) { m.Functions[0].Params[0].Name = "" },
		"untyped return": func(m *modules.Manifest) { m.Functions[0].Returns[0].Type = "" },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			m := validManifest()
			mutate(&m)
			if err := m.Validate(); err == nil {
				t.Fatalf("%s: expected an error, got nil", name)
			}
		})
	}
}

func TestModulesValidateRejectsDuplicateModuleNames(t *testing.T) {
	if err := modules.Validate([]modules.Module{stubModule{}, stubModule{}}); err == nil {
		t.Fatal("expected duplicate module names to be rejected")
	}
}

func TestModulesValidateAcceptsZeroManifest(t *testing.T) {
	if err := modules.Validate([]modules.Module{stubModule{}}); err != nil {
		t.Fatalf("zero manifest should be accepted: %v", err)
	}
}

func TestModulesValidateAcceptsBuiltin(t *testing.T) {
	if err := modules.Validate(modules.AllModules()); err != nil {
		t.Fatalf("built-in modules rejected: %v", err)
	}
}
