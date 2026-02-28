package layerengine

import (
	"github.com/slashbase/layerengine/pkg/modules"
	lua "github.com/yuin/gopher-lua"
)

type LayerRunner struct {
	luaRunner *lua.LState
}

func NewLayerRunner() *LayerRunner {
	layerRunner := LayerRunner{}
	layerRunner.init()
	return &layerRunner
}

func (cr *LayerRunner) init() {
	cr.luaRunner = lua.NewState()
	modules.Init(cr.luaRunner)
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
	for i := 0; i < outputLen; i++ {
		ret := cr.luaRunner.Get(i - outputLen)
		values = append(values, ret)
	}
	return values, nil
}
