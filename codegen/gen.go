package codegen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/slashbase/layerengine/codegen/prompts"
	"github.com/slashbase/layerengine/codegen/provider"
	"github.com/slashbase/layerengine/modules"
)

// defaultTimeout bounds one generation request when Options.Timeout is zero.
const defaultTimeout = 60 * time.Second

// Options tunes a CodeGen. The zero value is usable: the provider's default
// endpoint, and defaultTimeout.
type Options struct {
	// BaseURL overrides the provider's API endpoint. Empty means the SDK
	// default; set it to point at a proxy, a compatible gateway, or an httptest
	// server.
	BaseURL string

	// Timeout bounds one generation request. Zero means defaultTimeout.
	Timeout time.Duration
}

type CodeGen struct {
	gen       provider.Provider
	modelInfo provider.ModelInfo
	timeout   time.Duration
}

type Input = prompts.Input
type Output = prompts.Output
type ModelID = provider.ModelID

// NewCodeGen builds a code generator that talks to the provider's default
// endpoint. Use NewCodeGenWithOptions to override the endpoint or the timeout.
func NewCodeGen(apiKey string, model ModelID) (*CodeGen, error) {
	return NewCodeGenWithOptions(apiKey, model, Options{})
}

// NewCodeGenWithBaseURL is NewCodeGen with the provider's API endpoint
// overridden. An empty baseURL means the SDK default; set it to point at a
// proxy, a compatible gateway, or an httptest server in tests.
//
// Deprecated: use NewCodeGenWithOptions(apiKey, model, Options{BaseURL: baseURL}).
func NewCodeGenWithBaseURL(apiKey string, model ModelID, baseURL string) (*CodeGen, error) {
	return NewCodeGenWithOptions(apiKey, model, Options{BaseURL: baseURL})
}

// NewCodeGenWithOptions builds a code generator with every knob set explicitly.
//
// The model chooses the provider, so the two are always consistent: an unknown
// model is rejected here, and GenerateLayerFunction cannot be reached with one.
func NewCodeGenWithOptions(apiKey string, model ModelID, opts Options) (*CodeGen, error) {
	info, ok := provider.Models[model]
	if !ok {
		return nil, errors.New("model not supported: " + string(model))
	}

	factory, ok := provider.Providers[info.Provider]
	if !ok {
		return nil, errors.New("provider not supported: " + string(info.Provider))
	}

	gen, err := factory(apiKey, opts.BaseURL, info.ModelName)
	if err != nil {
		return nil, err
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	return &CodeGen{gen: gen, modelInfo: info, timeout: timeout}, nil
}

// GenerateLayerFunction asks the model for the Lua source of one layer function.
//
// It builds the prompt, calls the provider, and decodes the answer, in that
// order and each in one place, so the per-vendor code is only the call itself.
func (cg *CodeGen) GenerateLayerFunction(fnName, description string, inputs []Input, outputs []Output, mods []modules.Module) (string, error) {
	prompt := prompts.BuildPrompt(fnName, description, inputs, outputs, mods)

	ctx, cancel := context.WithTimeout(context.Background(), cg.timeout)
	defer cancel()

	answer, err := cg.gen.Generate(ctx, prompt.System, prompt.User)
	if err != nil {
		return "", err
	}

	return cg.decode(answer)
}

// decode unwraps the {"code": ...} envelope every provider is asked for.
//
// The truncation and empty-answer checks live here rather than in each provider
// because they are properties of the answer, not of the vendor — only the word
// the vendor uses to describe them differs, and reply has already translated it.
func (cg *CodeGen) decode(answer provider.Reply) (string, error) {
	model := cg.modelInfo.ModelName

	if answer.Truncated {
		return "", fmt.Errorf("%s: stopped at the token limit (stop reason %q); the answer is truncated", model, answer.Stop)
	}
	if strings.TrimSpace(answer.Text) == "" {
		return "", fmt.Errorf("%s: empty answer (stop reason %q)", model, answer.Stop)
	}

	var response struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal([]byte(answer.Text), &response); err != nil {
		return "", fmt.Errorf("%s: answer is not the expected {\"code\": ...} JSON: %w", model, err)
	}

	return response.Code, nil
}
