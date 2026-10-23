package modules

import (
	"github.com/slashbase/layerengine/modules/system"
	lua "github.com/yuin/gopher-lua"
)

type Module interface {
	Init(l *lua.LState) *lua.LTable
	Name() string
	Usage() string
}

func AllModules() []Module {
	return []Module{
		system.System{},
	}
}
