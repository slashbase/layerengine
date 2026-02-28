package system

import (
	lua "github.com/yuin/gopher-lua"
)

type System struct{}

func (s System) Init(l *lua.LState) *lua.LTable {
	table := l.NewTable()
	table.RawSetString("version", l.NewFunction(s.printV))
	return table
}

func (System) Name() string {
	return "system"
}

func (System) printV(l *lua.LState) int {
	l.Push(lua.LString("v0.0.0"))
	return 1
}
