package modules

import (
	"github.com/paraswaykole/layerdotrun/pkg/modules/database"
	"github.com/paraswaykole/layerdotrun/pkg/modules/system"
	lua "github.com/yuin/gopher-lua"
)

type Module interface {
	Init(l *lua.LState) *lua.LTable
	Name() string
}

func allModules() []Module {
	return []Module{
		system.System{},
		database.Database{},
	}
}
