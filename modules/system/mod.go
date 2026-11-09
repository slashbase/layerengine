package system

import (
	"github.com/slashbase/layerengine/modules/manifest"
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

func (System) Usage() string {
	return "versionString = layer.system.version() - get system version"
}

// Manifest declares the module's surface as data. codegen renders the signature
// lines from this and keeps Usage() underneath as notes.
func (System) Manifest() manifest.Manifest {
	return manifest.Manifest{
		APIVersion: manifest.APIVersion,
		Functions: []manifest.Function{
			{
				Name:    "version",
				Doc:     "get system version",
				Returns: []manifest.Return{{Name: "versionString", Type: "string"}},
			},
		},
	}
}

func (System) printV(l *lua.LState) int {
	l.Push(lua.LString("v0.0.0"))
	return 1
}
