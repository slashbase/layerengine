package modules

import (
	lua "github.com/yuin/gopher-lua"
)

// Init installs every module on l.
//
// Each module becomes a global named after it, so layer code calls
// system.version(...) for a module named "system". There is no wrapper
// table: the module tables are the whole surface, and what a manifest does not
// declare is not there.
//
// Init writes those globals; it does not choose which names are legal. That is
// Validate's job (identifier, not a reserved word) plus the free-name check the
// engine runs against a sandbox with no modules in it.
func Init(l *lua.LState, mods []Module) {
	for _, mod := range mods {
		l.SetGlobal(mod.Name(), mod.Init(l))
	}
}
