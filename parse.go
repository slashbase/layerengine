package layerengine

import (
	"strings"

	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
)

func ParseAndCompileLuaCode(codeStr string) (*lua.FunctionProto, error) {
	chunk, err := parse.Parse(strings.NewReader(codeStr), "<string>")
	if err != nil {
		return nil, err
	}
	proto, err := lua.Compile(chunk, "<string>")
	if err != nil {
		return nil, err
	}
	return proto, nil
}
