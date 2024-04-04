package codegen

import "fmt"

func generateCodePromptFormat(fnName, description string, inputs, outputs []string) string {

	inputStr := ""
	for _, in := range inputs {
		inputStr += "\n	- " + in
	}

	outputStr := ""
	for _, out := range outputs {
		outputStr += "\n	- " + out
	}

	return fmt.Sprintf(`Function Name: %s
Description: %s
Input:
%s
Output: 
%s`, fnName, description, inputStr, outputStr)
}
