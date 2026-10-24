package codegen

import (
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v3"
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
	GPT6_1Sol   ModelID = "gpt-6.1-sol"
	GPT6Sol     ModelID = "gpt-6-sol"
	GPT6Luna    ModelID = "gpt-6-luna"
	GPT6Astra   ModelID = "gpt-6-astra"
	GPT5_6Sol   ModelID = "gpt-5.6-sol"
	GPT5_6Terra ModelID = "gpt-5.6-terra"
	GPT5_6Luna  ModelID = "gpt-5.6-luna"
	GPT5_5      ModelID = "gpt-5.5"
	GPT5_4Mini  ModelID = "gpt-5.4-mini"
	GPT5_2      ModelID = "gpt-5.2"

	// Anthropic
	ClaudeFable5_1  ModelID = "claude-fable-5-1"
	ClaudeOpus5_5   ModelID = "claude-opus-5-5"
	ClaudeSonnet5_5 ModelID = "claude-sonnet-5-5"
	ClaudeHaiku5_5  ModelID = "claude-haiku-5-5"
	ClaudeOpus4_6   ModelID = "claude-opus-4.6"
	ClaudeSonnet4_6 ModelID = "claude-sonnet-4.6"
)

type modelInfo struct {
	provider  Provider
	modelName string // the string the vendor SDK expects
}

// models is the single source of truth mapping a ModelID to its provider and
// SDK name. Add a model by adding one entry here.
var models = map[ModelID]modelInfo{
	GPT6_1Sol:   {ProviderOpenAI, string(openai.ChatModelGPT6_1Sol)},
	GPT6Sol:     {ProviderOpenAI, string(openai.ChatModelGPT6Sol)},
	GPT6Luna:    {ProviderOpenAI, string(openai.ChatModelGPT6Luna)},
	GPT6Astra:   {ProviderOpenAI, string(openai.ChatModelGPT6Astra)},
	GPT5_6Sol:   {ProviderOpenAI, string(openai.ChatModelGPT5_6Sol)},
	GPT5_6Terra: {ProviderOpenAI, string(openai.ChatModelGPT5_6Terra)},
	GPT5_6Luna:  {ProviderOpenAI, string(openai.ChatModelGPT5_6Luna)},
	GPT5_5:      {ProviderOpenAI, string(openai.ChatModelGPT5_5)},
	GPT5_4Mini:  {ProviderOpenAI, string(openai.ChatModelGPT5_4Mini)},
	GPT5_2:      {ProviderOpenAI, string(openai.ChatModelGPT5_2)},

	ClaudeFable5_1:  {ProviderAnthropic, string(anthropic.ModelClaudeFable5_1)},
	ClaudeOpus5_5:   {ProviderAnthropic, string(anthropic.ModelClaudeOpus5_5)},
	ClaudeSonnet5_5: {ProviderAnthropic, string(anthropic.ModelClaudeSonnet5_5)},
	ClaudeHaiku5_5:  {ProviderAnthropic, string(anthropic.ModelClaudeHaiku5_5)},
	ClaudeOpus4_6:   {ProviderAnthropic, string(anthropic.ModelClaudeOpus4_6)},
	ClaudeSonnet4_6: {ProviderAnthropic, string(anthropic.ModelClaudeSonnet4_6)},
}
