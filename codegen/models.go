package codegen

import (
	"github.com/anthropics/anthropic-sdk-go"
	openai "github.com/sashabaranov/go-openai"
)

// Provider identifies an LLM backend.
type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
)

// ModelID is a stable identifier for a supported model. Always pass one of the
// exported constants below; constructing a ModelID from a raw string literal
// will not compile without an explicit cast.
type ModelID string

const (
	// OpenAI
	GPT3Dot5Turbo ModelID = "gpt-3.5-turbo"
	GPT4Turbo     ModelID = "gpt-4-turbo"
	GPT4o         ModelID = "gpt-4o"
	GPT4oMini     ModelID = "gpt-4o-mini"
	O1            ModelID = "o1"
	O3Mini        ModelID = "o3-mini"
	GPT5          ModelID = "gpt-5"

	// Anthropic
	ClaudeSonnet4     ModelID = "claude-sonnet-4"
	ClaudeSonnet4Dot5 ModelID = "claude-sonnet-4.5"
	ClaudeOpus4       ModelID = "claude-opus-4"
	ClaudeOpus4Dot6   ModelID = "claude-opus-4.6"
	ClaudeSonnet4Dot6 ModelID = "claude-sonnet-4.6"
)

type modelInfo struct {
	provider  Provider
	modelName string // the string the vendor SDK expects
}

// models is the single source of truth mapping a ModelID to its provider and
// SDK name. Add a model by adding one entry here.
var models = map[ModelID]modelInfo{
	GPT3Dot5Turbo: {ProviderOpenAI, openai.GPT3Dot5Turbo},
	GPT4Turbo:     {ProviderOpenAI, openai.GPT4Turbo},
	GPT4o:         {ProviderOpenAI, openai.GPT4o},
	GPT4oMini:     {ProviderOpenAI, openai.GPT4oMini},
	O1:            {ProviderOpenAI, openai.O1},
	O3Mini:        {ProviderOpenAI, openai.O3Mini},
	GPT5:          {ProviderOpenAI, openai.GPT5},

	ClaudeSonnet4:     {ProviderAnthropic, string(anthropic.ModelClaudeSonnet4_0)},
	ClaudeSonnet4Dot5: {ProviderAnthropic, string(anthropic.ModelClaudeSonnet4_5)},
	ClaudeOpus4:       {ProviderAnthropic, string(anthropic.ModelClaudeOpus4_0)},
	ClaudeOpus4Dot6:   {ProviderAnthropic, string(anthropic.ModelClaudeOpus4_6)},
	ClaudeSonnet4Dot6: {ProviderAnthropic, string(anthropic.ModelClaudeSonnet4_6)},
}
