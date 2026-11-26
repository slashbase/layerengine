package provider

import (
	"context"
)

// ProviderName identifies an LLM backend.
type ProviderName string

const (
	ProviderOpenAI    ProviderName = "openai"
	ProviderAnthropic ProviderName = "anthropic"
	ProviderSarvam    ProviderName = "sarvam"
)

// Provider turns a finished prompt into a model answer.
//
// A provider owns exactly what differs between vendors: the SDK call, the
// request knobs (token budget, reasoning effort, JSON mode), and the stop-reason
// vocabulary. It does not build prompts and does not decode the {"code": ...}
// envelope — those are shared, so a fix to either lands once for every provider.
type Provider interface {
	Generate(ctx context.Context, system []string, user string) (Reply, error)
}

// providerFactory builds a provider for one model. The provider is constructed
// with its model, so generate does not have to be told what it is running.
type providerFactory func(apiKey, baseURL, model string) (Provider, error)

// providers is the extension point. Adding a vendor means adding one entry here
// and one file that implements provider; no existing branch is edited.
var Providers = map[ProviderName]providerFactory{
	ProviderOpenAI:    newOpenAIProvider,
	ProviderAnthropic: newAnthropicProvider,
	ProviderSarvam:    newSarvamProvider,
}

// reply is one model answer as a provider returns it: the raw text, whether the
// model stopped because it ran out of room, and the vendor's own word for why
// it stopped.
//
// Translating a vendor's stop vocabulary into Truncated is the provider's job,
// so the shared path reports truncation the same way for every provider instead
// of each one growing its own check.
type Reply struct {
	Text      string
	Truncated bool
	Stop      string
}
