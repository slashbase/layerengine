package codegen

import (
	"fmt"
	"strings"
)

func generateCodePromptFormat(fnName, description string, inputs []Input, outputs []Output) string {

	var inputStr strings.Builder
	for _, in := range inputs {
		fmt.Fprintf(&inputStr, "\n\t- name: %s; type: %s; description: %s; optional: %t", in.Name, in.Type, in.Description, in.Optional)
	}

	var outputStr strings.Builder
	for _, out := range outputs {
		fmt.Fprintf(&outputStr, "\n\t- name: %s; type: %s; description: %s", out.Name, out.Type, out.Description)
	}

	return fmt.Sprintf(`Function Name: %s
Description: %s
Input:
%s
Input requirements:
- Lua function parameter names must exactly match the input names above.
Output: 
%s
Output requirements:
- Lua function output names must exactly match the output names above,  — no renaming, abbreviating, or restructuring.
- The function must return all the values directly (e.g. 'return output_name1, output_name2') — do NOT wrap it in a table.`, fnName, description, inputStr.String(), outputStr.String())
}
