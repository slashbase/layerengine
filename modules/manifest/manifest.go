// Package manifest describes a module's function surface as data, so the codegen
// prompt can be rendered from a strict format instead of free text.
//
// It is a leaf package on purpose: package modules imports modules/system, so
// modules/system cannot import modules back. Keeping the manifest types here
// lets a module implementation (including the built-in system module) declare a
// manifest without an import cycle. Package modules re-exports these types as
// aliases, so module authors only ever import modules.
package manifest

import (
	"fmt"
	"strings"
)

// APIVersion is the manifest format version this build understands.
const APIVersion = 1

// Manifest is a module's declared surface: every function it publishes on the
// layer table, with argument and return shapes.
//
// Arguments are always positional: fn(a, b), one Lua argument per declared
// parameter. Other calling conventions (a single table argument, say) are not
// modelled yet.
type Manifest struct {
	// APIVersion is the manifest format version. Zero means "current".
	APIVersion int
	// Notes is module-level prose: units, coordinate systems, rate limits,
	// anything that applies to every function in this module.
	Notes string
	// Functions is the complete list of functions the module exposes.
	Functions []Function
}

// Function is one callable on the module table.
type Function struct {
	Name    string
	Doc     string
	Params  []Param
	Returns []Return
}

// Param is one argument. Type is a prompt-level type expression, not an
// enforced one: "string", "number", "boolean", "table", or a table shape such
// as "table{city:string, limit:number}".
//
// Default is rendered verbatim into the prompt as name?:type=default, and makes
// the argument omittable whether or not Optional is set. Per-argument prose has
// no home here — the one-line call format has no room for it — so describe
// arguments in Function.Doc instead.
type Param struct {
	Name     string
	Type     string
	Optional bool
	Default  string
}

// Return is one value the function hands back. Type carries the field names when
// the value is a table; say what it means in Function.Doc.
type Return struct {
	Name string
	Type string
}

// IsZero reports that the module declares no surface at all. A zero manifest is
// the escape hatch for a module that only wants to render its Usage() text.
func (m Manifest) IsZero() bool {
	return m.APIVersion == 0 && m.Notes == "" && len(m.Functions) == 0
}

// Validate reports whether a manifest is structurally sound. It does not check
// that the module actually implements what it declares — Init still owns that.
func (m Manifest) Validate() error {
	if m.APIVersion != 0 && m.APIVersion != APIVersion {
		return fmt.Errorf("manifest api version %d is not supported (want %d)", m.APIVersion, APIVersion)
	}

	if len(m.Functions) == 0 {
		return fmt.Errorf("manifest declares no functions")
	}

	seen := make(map[string]bool, len(m.Functions))
	for i, fn := range m.Functions {
		if fn.Name == "" {
			return fmt.Errorf("function %d has no name", i)
		}
		if seen[fn.Name] {
			return fmt.Errorf("duplicate function %q", fn.Name)
		}
		seen[fn.Name] = true
		if strings.HasPrefix(fn.Name, "layer.") {
			return fmt.Errorf("function %q must be a bare name; the module name supplies the layer.<module> prefix", fn.Name)
		}
		for j, p := range fn.Params {
			if p.Name == "" {
				return fmt.Errorf("function %q param %d has no name", fn.Name, j)
			}
		}
		for j, r := range fn.Returns {
			if r.Type == "" {
				return fmt.Errorf("function %q return %d has no type", fn.Name, j)
			}
		}
	}

	return nil
}
