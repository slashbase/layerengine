package prompts

import (
	"fmt"
	"strings"
)

// Input describes a named layer input for code generation.
type Input struct {
	Name        string `key:"name"`
	Type        string `key:"type"`
	Description string `key:"description"`
	Optional    bool   `key:"optional"`
}

// Output describes a named layer output for code generation.
type Output struct {
	Name        string `key:"name"`
	Type        string `key:"type"`
	Description string `key:"description"`
}

func generateCodePromptFormat(fnName, description string, inputs []Input, outputs []Output) string {

	var inputStr strings.Builder
	for _, in := range inputs {
		fmt.Fprintf(&inputStr, "\n\t- name: %s; type: %s; description: %s; optional: %t", in.Name, in.Type, in.Description, in.Optional)
	}

	var outputStr strings.Builder
	for _, out := range outputs {
		fmt.Fprintf(&outputStr, "\n\t- name: %s; type: %s; description: %s", out.Name, out.Type, out.Description)
	}

	var returnStr strings.Builder
	for i, out := range outputs {
		if i > 0 {
			returnStr.WriteString(", ")
		}
		fmt.Fprintf(&returnStr, "%s (%s)", out.Name, out.Type)
	}
	noun := "values"
	if len(outputs) == 1 {
		noun = "value"
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
- Lua function output names must exactly match the output names above — no renaming, abbreviating, or restructuring.
- Return exactly %d %s, in this order: %s.
- Return computed expressions or locals you defined, never an output name as a bare identifier, and do not wrap the results in a table.`, fnName, description, inputStr.String(), outputStr.String(), len(outputs), noun, returnStr.String())
}
