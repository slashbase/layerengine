package codegen

import (
	"errors"

	"github.com/anthropics/anthropic-sdk-go"
	openai "github.com/sashabaranov/go-openai"
)

const (
	// OpenAI models
	OPENAI_GPT3DOT5_TURBO int = 0
	OPENAI_GPT4_TURBO     int = 1
	OPENAI_GPT4O          int = 2
	OPENAI_GPT4O_MINI     int = 3
	OPENAI_O1             int = 4
	OPENAI_O3_MINI        int = 5
	OPENAI_GPT5           int = 6

	// Anthropic models
	ANTHROPIC_CLAUDE_SONNET_4   int = 100
	ANTHROPIC_CLAUDE_SONNET_4_5 int = 101
	ANTHROPIC_CLAUDE_OPUS_4     int = 102
	ANTHROPIC_CLAUDE_OPUS_4_6   int = 103
	ANTHROPIC_CLAUDE_SONNET_4_6 int = 104
)

var openAIModelStringMap = map[int]string{
	OPENAI_GPT3DOT5_TURBO: openai.GPT3Dot5Turbo,
	OPENAI_GPT4_TURBO:     openai.GPT4Turbo,
	OPENAI_GPT4O:          openai.GPT4o,
	OPENAI_GPT4O_MINI:     openai.GPT4oMini,
	OPENAI_O1:             openai.O1,
	OPENAI_O3_MINI:        openai.O3Mini,
	OPENAI_GPT5:           openai.GPT5,
}

var anthropicModelStringMap = map[int]string{
	ANTHROPIC_CLAUDE_SONNET_4:   string(anthropic.ModelClaudeSonnet4_0),
	ANTHROPIC_CLAUDE_SONNET_4_5: string(anthropic.ModelClaudeSonnet4_5),
	ANTHROPIC_CLAUDE_OPUS_4:     string(anthropic.ModelClaudeOpus4_0),
	ANTHROPIC_CLAUDE_OPUS_4_6:   string(anthropic.ModelClaudeOpus4_6),
	ANTHROPIC_CLAUDE_SONNET_4_6: string(anthropic.ModelClaudeSonnet4_6),
}

type CodeGen struct {
	openAIClient    *openai.Client
	anthropicClient *anthropic.Client
	model           int
}

func NewCodeGen(openAISecretKey, anthropicSecretKey string, model int) (*CodeGen, error) {
	_, isOpenAI := openAIModelStringMap[model]
	_, isAnthropic := anthropicModelStringMap[model]

	if !isOpenAI && !isAnthropic {
		return nil, errors.New("model not supported")
	}

	return &CodeGen{
		openAIClient:    openai.NewClient(openAISecretKey),
		anthropicClient: newAnthropicClient(anthropicSecretKey),
		model:           model,
	}, nil
}

func (cg *CodeGen) GenerateLayerFunction(fnName, description string, inputs, outputs []string) (string, error) {
	if modelStr, ok := openAIModelStringMap[cg.model]; ok {
		return generateLuaFunctionCode(cg.openAIClient, modelStr, fnName, description, inputs, outputs)
	}
	if modelStr, ok := anthropicModelStringMap[cg.model]; ok {
		return generateLuaFunctionCodeAnthropic(cg.anthropicClient, modelStr, fnName, description, inputs, outputs)
	}
	return "", errors.New("model not supported")
}
