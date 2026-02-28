package validator

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ──────────────────────────────────────────────
// Schema types
// ──────────────────────────────────────────────

type Layer struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Input       []string `yaml:"input"`
	Output      []string `yaml:"output"`
}

type Flow struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Input       []string `yaml:"input"`
	Layers      []Layer  `yaml:"layers"`
}

// ──────────────────────────────────────────────
// Allowed keys (strict key enforcement)
// ──────────────────────────────────────────────

var allowedTopKeys = map[string]bool{
	"name": true, "description": true, "input": true, "layers": true,
}

var allowedLayerKeys = map[string]bool{
	"name": true, "description": true, "input": true, "output": true,
}

// ──────────────────────────────────────────────
// Compiler
// ──────────────────────────────────────────────

type CompileError struct {
	Line    int
	Message string
}

func (e CompileError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("line %d: %s", e.Line, e.Message)
	}
	return e.Message
}

// Compile parses and validates the YAML source, returning the compiled
// Flow or a list of errors.
func Compile(src []byte) (*Flow, []error) {
	var errs []error

	// ── Step 1: parse into a raw node to validate keys ──────────────────
	var root yaml.Node
	if err := yaml.Unmarshal(src, &root); err != nil {
		return nil, []error{fmt.Errorf("yaml parse error: %w", err)}
	}
	if len(root.Content) == 0 {
		return nil, []error{errors.New("empty document")}
	}

	doc := root.Content[0] // document node
	if doc.Kind != yaml.MappingNode {
		return nil, []error{errors.New("top-level must be a mapping")}
	}

	// Validate top-level keys
	errs = append(errs, validateKeys(doc, allowedTopKeys, "top-level")...)

	// Find layers node and validate its children keys
	for i := 0; i+1 < len(doc.Content); i += 2 {
		if doc.Content[i].Value == "layers" {
			seq := doc.Content[i+1]
			if seq.Kind != yaml.SequenceNode {
				errs = append(errs, CompileError{seq.Line, "'layers' must be a sequence"})
				break
			}
			for idx, item := range seq.Content {
				if item.Kind != yaml.MappingNode {
					errs = append(errs, CompileError{item.Line, fmt.Sprintf("layer[%d] must be a mapping", idx)})
					continue
				}
				errs = append(errs, validateKeys(item, allowedLayerKeys,
					fmt.Sprintf("layer[%d]", idx))...)
			}
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}

	// ── Step 2: unmarshal into typed struct ──────────────────────────────
	var flow Flow
	if err := yaml.Unmarshal(src, &flow); err != nil {
		return nil, []error{fmt.Errorf("unmarshal error: %w", err)}
	}

	// ── Step 3: required-field checks ───────────────────────────────────
	if strings.TrimSpace(flow.Name) == "" {
		errs = append(errs, errors.New("top-level 'name' is required"))
	}
	for i, l := range flow.Layers {
		if strings.TrimSpace(l.Name) == "" {
			errs = append(errs, fmt.Errorf("layer[%d]: 'name' is required", i))
		}
	}

	// ── Step 4: input/output name collision check (per layer) ──────────
	//
	// Within a single layer, the same variable name must not appear in
	// both its input list and its output list.  Across layers it is fine
	// (and expected) for an earlier layer's output to feed a later layer's
	// input — that is the pipeline flow validated in Step 5.
	// The top-level flow inputs are also checked against every layer's
	// outputs: an externally-supplied value should never be re-produced
	// internally.

	topInputSet := make(map[string]bool)
	for _, v := range flow.Input {
		topInputSet[strings.TrimSpace(v)] = true
	}

	for _, l := range flow.Layers {
		layerInputSet := make(map[string]bool)
		for _, v := range l.Input {
			layerInputSet[strings.TrimSpace(v)] = true
		}
		loc := fmt.Sprintf("layer %q", l.Name)
		for _, v := range l.Output {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			// Collision within the same layer
			if layerInputSet[v] {
				errs = append(errs, fmt.Errorf(
					"%s: variable %q appears in both input and output of the same layer",
					loc, v,
				))
			}
			// Top-level input re-produced by a layer
			if topInputSet[v] {
				errs = append(errs, fmt.Errorf(
					"%s: variable %q collides with a top-level input — "+
						"externally supplied values must not be re-produced by a layer",
					loc, v,
				))
			}
		}
	}

	// ── Step 5: layer-by-layer input availability (pipeline flow) ────────
	//
	// The pool starts with the top-level inputs.
	// At each layer (top-to-bottom):
	//   1. Every input the layer requests must already be in the pool.
	//   2. The layer's outputs are then added to the pool for subsequent layers.

	pool := make(map[string]string) // name -> "where it became available"
	for _, v := range flow.Input {
		v = strings.TrimSpace(v)
		if v != "" {
			pool[v] = "top-level input"
		}
	}

	for i, l := range flow.Layers {
		layerID := fmt.Sprintf("layer[%d] %q", i+1, l.Name)
		for _, v := range l.Input {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			if src, ok := pool[v]; ok {
				_ = src // available — fine
			} else {
				// Build a hint: what IS in the pool right now?
				available := poolNames(pool)
				errs = append(errs, fmt.Errorf(
					"%s requires input %q but it is not available at this point\n"+
						"        available pool: [%s]",
					layerID, v, available,
				))
			}
		}
		// After checking, publish this layer's outputs into the pool.
		for _, v := range l.Output {
			v = strings.TrimSpace(v)
			if v != "" {
				pool[v] = layerID + " output"
			}
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}
	return &flow, nil
}

// validateKeys checks that every key in a mapping node is in the allowed set.
func validateKeys(node *yaml.Node, allowed map[string]bool, context string) []error {
	var errs []error
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		if !allowed[key.Value] {
			errs = append(errs, CompileError{
				Line: key.Line,
				Message: fmt.Sprintf(
					"%s has unknown key %q (allowed: %s)",
					context, key.Value, allowedKeyList(allowed),
				),
			})
		}
	}
	return errs
}

func allowedKeyList(m map[string]bool) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

// poolNames returns a deterministic, comma-joined list of current pool keys
// used to build helpful "available variables" hints in error messages.
func poolNames(pool map[string]string) string {
	names := make([]string, 0, len(pool))
	for k := range pool {
		names = append(names, k)
	}
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j] < names[j-1]; j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}
	if len(names) == 0 {
		return "(empty)"
	}
	return strings.Join(names, ", ")
}

// Run compiles YAML source supplied as a byte slice.
// On success it returns success and a nil error.
// On failure it returns false and a combined error whose message lists every
// individual compile error, one per line.
func Run(src []byte) (*Flow, error) {
	flow, errs := Compile(src)
	if len(errs) > 0 {
		msgs := make([]string, 0, len(errs)+1)
		msgs = append(msgs, fmt.Sprintf("compilation failed with %d error(s):", len(errs)))
		for _, e := range errs {
			msgs = append(msgs, "  ✗ "+e.Error())
		}
		return nil, errors.New(strings.Join(msgs, "\n"))
	}

	return flow, nil
}
