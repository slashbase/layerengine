package codegen

import (
	"fmt"
	"strings"

	"github.com/slashbase/layerengine/modules"
)

const (
	modulesUsageInfo = "Use:"

	environmentFacts = `This function runs in a restricted Lua 5.1 environment.
Available: the base library, string, table, math, and the layer module table.
Not available: os, io, debug, package, require, dofile, load, loadstring,
getfenv, setfenv.`
)

const (
	promptGuide = "Write a Lua function that takes input parameters, processes the input according to given description and returns output.\nWrite code in json field \"code\"."
)

func GetModuleUsagePrompt(mods []modules.Module) string {
	usage := []string{}
	for _, m := range mods {
		usage = append(usage, m.Usage())
	}
	return fmt.Sprintf("%s\n%s\n%s", environmentFacts, modulesUsageInfo, strings.Join(usage, "\n"))
}
