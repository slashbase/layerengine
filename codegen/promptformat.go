package codegen

import (
	"fmt"
	"strings"
)

func generateCodePromptFormat(fnName, description string, inputs []Input, outputs []string) string {

	var inputStr strings.Builder
	for _, in := range inputs {
		fmt.Fprintf(&inputStr, "\n\t- name: %s; optional: %t", in.Name, in.Optional)
	}

	var outputStr strings.Builder
	for _, out := range outputs {
		outputStr.WriteString("\n	- " + out)
	}

	return fmt.Sprintf(`Function Name: %s
Description: %s
Input:
%s
Input requirements:
- Lua function parameter names must exactly match the input names above.
Output: 
%s`, fnName, description, inputStr.String(), outputStr.String())
}
