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

// TestManifestValidateAcceptsGroupedNames pins the supported shape: one module,
// several groups, each operation a path into the table Init builds.
func TestManifestValidateAcceptsGroupedNames(t *testing.T) {
	m := modules.Manifest{
		APIVersion: modules.APIVersion,
		Notes:      "lengths in, lengths out",
		Functions: []modules.Function{
			{
				Name:    "circle.area",
				Doc:     "Area of a circle from its radius.",
				Params:  []modules.Param{{Name: "radius", Type: "number"}},
				Returns: []modules.Return{{Name: "area", Type: "number"}},
			},
			{
				Name:    "circle.perimeter",
				Doc:     "Perimeter of a circle from its radius.",
				Params:  []modules.Param{{Name: "radius", Type: "number"}},
				Returns: []modules.Return{{Name: "perimeter", Type: "number"}},
			},
			{
				Name:    "rect.area",
				Doc:     "Area of a rectangle.",
				Params:  []modules.Param{{Name: "width", Type: "number"}, {Name: "height", Type: "number"}},
				Returns: []modules.Return{{Name: "area", Type: "number"}},
			},
		},
	}

	if err := m.Validate(); err != nil {
		t.Fatalf("grouped function names rejected: %v", err)
	}
}

// TestManifestValidateAcceptsLayerSegment pins that "layer" is an ordinary
// segment: the module name is the first segment at the call site, so a group
// called layer is a group like any other.
func TestManifestValidateAcceptsLayerSegment(t *testing.T) {
	m := validManifest()
	m.Functions[0].Name = "layer.current"

	if err := m.Validate(); err != nil {
		t.Fatalf("layer as a group name rejected: %v", err)
	}
}

func TestManifestValidateRejects(t *testing.T) {
	cases := map[string]func(*modules.Manifest){
		"no functions":        func(m *modules.Manifest) { m.Functions = nil },
		"unsupported version": func(m *modules.Manifest) { m.APIVersion = 99 },
		"unnamed function":    func(m *modules.Manifest) { m.Functions[0].Name = "" },
		"duplicate function": func(m *modules.Manifest) {
			m.Functions = append(m.Functions, m.Functions[0])
		},
		"unnamed param":  func(m *modules.Manifest) { m.Functions[0].Params[0].Name = "" },
		"untyped return": func(m *modules.Manifest) { m.Functions[0].Returns[0].Type = "" },

		// A name is a dotted path, but only as paths of Lua identifiers.
		"empty segment":  func(m *modules.Manifest) { m.Functions[0].Name = "a..b" },
		"trailing dot":   func(m *modules.Manifest) { m.Functions[0].Name = "a." },
		"leading dot":    func(m *modules.Manifest) { m.Functions[0].Name = ".a" },
		"hyphen segment": func(m *modules.Manifest) { m.Functions[0].Name = "a.b-c" },
		"space segment":  func(m *modules.Manifest) { m.Functions[0].Name = "a b" },
		"digit segment":  func(m *modules.Manifest) { m.Functions[0].Name = "1a" },
		"keyword segment": func(m *modules.Manifest) {
			m.Functions[0].Name = "a.end"
		},
		"prefix collision": func(m *modules.Manifest) {
			m.Functions[0].Name = "circle"
			m.Functions = append(m.Functions, modules.Function{
				Name:    "circle.area",
				Params:  []modules.Param{{Name: "radius", Type: "number"}},
				Returns: []modules.Return{{Name: "area", Type: "number"}},
			})
		},
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

// namedModule is a Module with a fixed name and no manifest, for checking the
// rules a module name has to satisfy now that it is a global.
type namedModule string

func (n namedModule) Name() string { return string(n) }

func (namedModule) Usage() string { return "" }

func (namedModule) Init(l *lua.LState) *lua.LTable { return l.NewTable() }

func (namedModule) Manifest() modules.Manifest { return modules.Manifest{} }

// TestModulesValidateRejectsUnusableNames covers the names a module cannot
// have: not an identifier, or a reserved word. A name that merely collides with
// the standard library is caught later, against a live sandbox, not here.
func TestModulesValidateRejectsUnusableNames(t *testing.T) {
	for name, moduleName := range map[string]string{
		"hyphen":        "my-http",
		"dot":           "http.github",
		"space":         "my http",
		"digit-leading": "1http",
		"reserved word": "end",
	} {
		t.Run(name, func(t *testing.T) {
			if err := modules.Validate([]modules.Module{namedModule(moduleName)}); err == nil {
				t.Fatalf("expected module name %q to be rejected", moduleName)
			}
		})
	}
}

func TestModulesValidateAcceptsUsableNames(t *testing.T) {
	for _, name := range []string{"http", "system", "_internal", "db2", "layer"} {
		if err := modules.Validate([]modules.Module{namedModule(name)}); err != nil {
			t.Fatalf("module name %q rejected: %v", name, err)
		}
	}
}
