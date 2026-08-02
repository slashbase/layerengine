package layerengine

import (
	"github.com/slashbase/layerengine/validator"
	lua "github.com/yuin/gopher-lua"
)

type Layer struct {
	validator.Layer
	FnProto *lua.FunctionProto `yaml:"-"`
	Code    string             `yaml:"-"`
}

type FlowInput struct {
	validator.FlowInput
}
