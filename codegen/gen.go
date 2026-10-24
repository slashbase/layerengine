package codegen

import (
	"errors"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v3"
	"github.com/slashbase/layerengine/modules"
)

const requestTimeout = 60 * time.Second

type CodeGen struct {
	openAIClient    *openai.Client
	anthropicClient *anthropic.Client
	modelInfo       modelInfo
}

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
		cg.openAIClient = newOpenAIClient(apiKey)
	case ProviderAnthropic:
		cg.anthropicClient = newAnthropicClient(apiKey)
	}
	return cg, nil
}

func (cg *CodeGen) GenerateLayerFunction(fnName, description string, inputs []Input, outputs []Output, mods []modules.Module) (string, error) {
	switch cg.modelInfo.provider {
	case ProviderOpenAI:
		return generateLuaFunctionCode(cg.openAIClient, cg.modelInfo.modelName, fnName, description, inputs, outputs, mods)
	case ProviderAnthropic:
		return generateLuaFunctionCodeAnthropic(cg.anthropicClient, cg.modelInfo.modelName, fnName, description, inputs, outputs, mods)
	}
	return "", errors.New("model not supported")
}
