package modules

import (
	lua "github.com/yuin/gopher-lua"
)

func Init(l *lua.LState) {
	table := l.NewTable()
	mods := AllModules()
	for _, mod := range mods {
		table.RawSetString(mod.Name(), mod.Init(l))
	}
	l.SetGlobal("layer", table)
}
