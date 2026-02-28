package codegen

import (
	"errors"
)

const (
	OPENAI_GPT3DOT5_TURBO     int = 0
	OPENAI_GPT4_TURBO_PREVIEW int = 1
)

func GenerateLayerFunction(model int, fnName, description string, inputs, outputs []string) (string, error) {
	var codeStr string
	switch model {
	case OPENAI_GPT3DOT5_TURBO:
		var err error
		codeStr, err = generateLuaFunctionCode(fnName, description, inputs, outputs)
		if err != nil {
			return "", err
		}
	default:
		return "", errors.New("model not supported")
	}
	return codeStr, nil
}
