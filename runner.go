package layerengine

import (
	"context"

	"github.com/slashbase/layerengine/modules"
	lua "github.com/yuin/gopher-lua"
)

type LayerRunner struct {
	luaRunner *lua.LState
}

func NewLayerRunner(ctx context.Context, mods []modules.Module) *LayerRunner {
	layerRunner := LayerRunner{}
	layerRunner.init(ctx)
	layerRunner.openLibsAndMods(mods)
	return &layerRunner
}

func (cr *LayerRunner) init(ctx context.Context) {
	cr.luaRunner = lua.NewState(lua.Options{
		SkipOpenLibs:    true,
		CallStackSize:   256,
		RegistrySize:    256 * 20, // 5120
		RegistryMaxSize: 256 * 20, // same → growth effectively capped here
	})
	cr.luaRunner.SetContext(ctx)
}

func (cr *LayerRunner) openLibsAndMods(mods []modules.Module) {
	lua.OpenBase(cr.luaRunner)
	lua.OpenTable(cr.luaRunner)
	lua.OpenString(cr.luaRunner)
	lua.OpenMath(cr.luaRunner)
	// deliberately NOT opened: os, io, debug, package, channel, coroutine
	for _, name := range []string{
		"load", "loadstring", "dofile", "getfenv", "setfenv", "collectgarbage", "newproxy",
	} {
		cr.luaRunner.SetGlobal(name, lua.LNil)
	}
	modules.Init(cr.luaRunner, mods)
}

func (cr *LayerRunner) Close() {
	cr.luaRunner.Close()
}

func (cr *LayerRunner) LoadFunction(fnProto *lua.FunctionProto) error {
	lfunc := lua.LFunction{
		IsG:       false,
		Env:       cr.luaRunner.Env,
		Proto:     fnProto,
		GFunction: nil,
		Upvalues:  make([]*lua.Upvalue, 0),
	}
	cr.luaRunner.Push(&lfunc)
	if err := cr.luaRunner.PCall(0, lua.MultRet, nil); err != nil {
		return err
	}
	return nil
}

func (cr *LayerRunner) RunFunction(funName string, arguments []lua.LValue, outputLen int) error {
	if err := cr.luaRunner.CallByParam(lua.P{
		Fn:      cr.luaRunner.GetGlobal(funName),
		NRet:    outputLen,
		Protect: true,
	}, arguments...); err != nil {
		return err
	}
	return nil
}

func (cr *LayerRunner) ReadResult(outputLen int) ([]lua.LValue, error) {
	values := []lua.LValue{}
	for i := range outputLen {
		ret := cr.luaRunner.Get(i - outputLen)
		values = append(values, ret)
	}
	return values, nil
}
