package codegen

import (
	"errors"

	"github.com/anthropics/anthropic-sdk-go"
	openai "github.com/sashabaranov/go-openai"
)

type CodeGen struct {
	openAIClient    *openai.Client
	anthropicClient *anthropic.Client
	modelInfo       modelInfo
}

// Input describes a named layer input for code generation.
type Input struct {
	Name     string
	Optional bool
}

func NewCodeGen(apiKey string, model ModelID) (*CodeGen, error) {
	info, ok := models[model]
	if !ok {
		return nil, errors.New("model not supported: " + string(model))
	}

	cg := &CodeGen{
		modelInfo: info,
	}

	switch info.provider {
	case ProviderOpenAI:
		cg.openAIClient = openai.NewClient(apiKey)
	case ProviderAnthropic:
		cg.anthropicClient = newAnthropicClient(apiKey)
	}
	return cg, nil
}

func (cg *CodeGen) GenerateLayerFunction(fnName, description string, inputs []Input, outputs []string) (string, error) {
	switch cg.modelInfo.provider {
	case ProviderOpenAI:
		return generateLuaFunctionCode(cg.openAIClient, cg.modelInfo.modelName, fnName, description, inputs, outputs)
	case ProviderAnthropic:
		return generateLuaFunctionCodeAnthropic(cg.anthropicClient, cg.modelInfo.modelName, fnName, description, inputs, outputs)
	}
	return "", errors.New("model not supported")
}
