package modules

import (
	"fmt"

	"github.com/slashbase/layerengine/modules/manifest"
	"github.com/slashbase/layerengine/modules/system"
	lua "github.com/yuin/gopher-lua"
)

// Module is a capability published on the layer's Lua state.
//
// Init returns the table the module publishes. The engine binds that table as a
// global named after the module, so a module called "http" is reached as
// http.<fn>(...) in generated code.
//
// Manifest is part of the contract: a module declares the functions it exposes
// as data, and codegen renders the prompt from that declaration. A module with
// nothing to declare returns the zero Manifest, and renders from Usage() alone.
type Module interface {
	Init(l *lua.LState) *lua.LTable
	Name() string
	Usage() string
	Manifest() Manifest
}

// Validate checks a module set for the problems that would produce a broken or
// ambiguous prompt: empty, duplicate or unusable module names, and malformed
// manifests. A module that returns the zero Manifest is accepted — it renders
// from Usage().
//
// A module name is a global in the layer sandbox and the first word of every
// call line, so it has to be an identifier. Names that collide with the standard
// library or with a loader the sandbox removed are caught later, against a live
// sandbox; see checkModuleNamesAreFree in the root package.
func Validate(mods []Module) error {
	seen := make(map[string]bool, len(mods))
	for _, m := range mods {
		name := m.Name()
		if name == "" {
			return fmt.Errorf("module with empty name")
		}
		if seen[name] {
			return fmt.Errorf("duplicate module %q", name)
		}
		seen[name] = true

		if !manifest.IsLuaIdentifier(name) || manifest.IsLuaKeyword(name) {
			return fmt.Errorf("module %q: a module name must be a Lua identifier that is not a reserved word", name)
		}

		man := m.Manifest()
		if man.IsZero() {
			continue
		}
		if err := man.Validate(); err != nil {
			return fmt.Errorf("module %q: %w", m.Name(), err)
		}
	}
	return nil
}

func AllModules() []Module {
	return []Module{
		system.System{},
	}
}

// Aliases so module authors can name manifest types through this package
// without importing modules/manifest as well. The types live in the leaf
// package to keep modules/system free of an import cycle.
type (
	Manifest = manifest.Manifest
	Function = manifest.Function
	Param    = manifest.Param
	Return   = manifest.Return
)

const APIVersion = manifest.APIVersion
