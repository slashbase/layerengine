// Package config turns a model choice and credentials into a code generator.
//
// Config is a plain data type plus one resolver. It never reads the environment
// or a file, so the caller decides where its values come from; FromEnv is the
// convenience for programs that want the environment to be the source.
package config

import (
	"fmt"
	"net/url"
	"os"

	"github.com/slashbase/layerengine/codegen"
)

// Config describes how to reach the LLM that generates layer code.
type Config struct {
	// Model selects both the provider and the model, so it must be one of the
	// exported codegen.ModelID constants. The provider is derived from it.
	Model codegen.ModelID

	// APIKey authenticates against the provider that Model belongs to.
	APIKey string

	// BaseURL optionally overrides the provider's API endpoint. Leave it empty
	// for the SDK default. Set it to point at a proxy, a compatible gateway, or
	// an httptest server in tests.
	BaseURL string
}

// CodeGen builds a code generator from the config.
func (c Config) CodeGen() (*codegen.CodeGen, error) {
	if c.APIKey == "" {
		return nil, fmt.Errorf("config: API key is empty")
	}
	if c.BaseURL != "" {
		u, err := url.Parse(c.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("config: invalid base URL %q: %w", c.BaseURL, err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return nil, fmt.Errorf("config: base URL %q must be an absolute http(s) URL", c.BaseURL)
		}
		if u.Host == "" {
			return nil, fmt.Errorf("config: base URL %q has no host", c.BaseURL)
		}
	}
	return codegen.NewCodeGenWithBaseURL(c.APIKey, c.Model, c.BaseURL)
}

// Environment variables FromEnv reads.
const (
	envOpenAIKey    = "OPENAI_API_KEY"
	envAnthropicKey = "ANTHROPIC_API_KEY"
	envModel        = "LAYERENGINE_MODEL"
	envModelBaseURL = "LAYERENGINE_MODEL_BASE_URL"
)

// Default models FromEnv picks when a provider key is present and no model is
// requested.
const (
	defaultOpenAIModel    = codegen.GPT6Luna
	defaultAnthropicModel = codegen.ClaudeSonnet5_5
)

// FromEnv builds a Config from the environment.
//
// It reads OPENAI_API_KEY or ANTHROPIC_API_KEY (OpenAI wins if both are set),
// plus optionally LAYERENGINE_MODEL to override the default model and
// LAYERENGINE_MODEL_BASE_URL to override the endpoint. An unknown model is not
// rejected here; CodeGen reports it.
func FromEnv() (Config, error) {
	var cfg Config

	switch {
	case os.Getenv(envOpenAIKey) != "":
		cfg.APIKey = os.Getenv(envOpenAIKey)
		cfg.Model = defaultOpenAIModel
	case os.Getenv(envAnthropicKey) != "":
		cfg.APIKey = os.Getenv(envAnthropicKey)
		cfg.Model = defaultAnthropicModel
	default:
		return Config{}, fmt.Errorf("config: set %s or %s", envOpenAIKey, envAnthropicKey)
	}

	if model := os.Getenv(envModel); model != "" {
		cfg.Model = codegen.ModelID(model)
	}
	cfg.BaseURL = os.Getenv(envModelBaseURL)
	return cfg, nil
}
