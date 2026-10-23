package codegen

import (
	"fmt"
	"strings"

	"github.com/slashbase/layerengine/modules"
)

const (
	modulesUsageInfo = "Use:"
)

const (
	promptGuide = "Write a Lua function that takes input parameters, processes the input according to given description and returns output.\nWrite code in json field \"code\"."
)

func GetModuleUsagePrompt(mods []modules.Module) string {
	usage := []string{}
	for _, m := range mods {
		usage = append(usage, m.Usage())
	}
	return fmt.Sprintf("%s\n%s", modulesUsageInfo, strings.Join(usage, "\n"))
}
