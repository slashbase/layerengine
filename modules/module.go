package modules

import (
	"fmt"

	"github.com/slashbase/layerengine/modules/manifest"
	"github.com/slashbase/layerengine/modules/system"
	lua "github.com/yuin/gopher-lua"
)

// Module is a capability published on the layer table.
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
// ambiguous prompt: empty or duplicate module names, and malformed manifests.
// A module that returns the zero Manifest is accepted — it renders from Usage().
func Validate(mods []Module) error {
	seen := make(map[string]bool, len(mods))
	for _, m := range mods {
		if m.Name() == "" {
			return fmt.Errorf("module with empty name")
		}
		if seen[m.Name()] {
			return fmt.Errorf("duplicate module %q", m.Name())
		}
		seen[m.Name()] = true

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
