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

// NewCodeGen builds a code generator that talks to the provider's default
// endpoint. Use NewCodeGenWithBaseURL to override it.
func NewCodeGen(apiKey string, model ModelID) (*CodeGen, error) {
	return NewCodeGenWithBaseURL(apiKey, model, "")
}

// NewCodeGenWithBaseURL is NewCodeGen with the provider's API endpoint
// overridden. An empty baseURL means the SDK default; set it to point at a
// proxy, a compatible gateway, or an httptest server in tests.
func NewCodeGenWithBaseURL(apiKey string, model ModelID, baseURL string) (*CodeGen, error) {
	info, ok := models[model]
	if !ok {
		return nil, errors.New("model not supported: " + string(model))
	}

	cg := &CodeGen{
		modelInfo: info,
	}

	switch info.provider {
	case ProviderOpenAI:
		cg.openAIClient = newOpenAIClient(apiKey, baseURL)
	case ProviderAnthropic:
		cg.anthropicClient = newAnthropicClient(apiKey, baseURL)
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
