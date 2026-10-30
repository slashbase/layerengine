package layerengine

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// CoerceOptions controls how CoerceInputs treats the arguments handed to a flow.
// The zero value is the lenient policy: values are converted where they can be,
// and arguments the flow does not declare are dropped.
type CoerceOptions struct {
	// Strict rejects a JSON-type mismatch instead of converting it. A caller
	// that passes the string "5" for a declared number input gets an error
	// rather than 5. A value that already has the declared type is always
	// accepted, so Strict only affects values that would have been converted.
	Strict bool

	// KeepUndeclared forwards arguments the flow does not declare, untouched,
	// instead of dropping them. This is the pre-coercion behaviour of RunFlow,
	// and it is the escape hatch for a flow whose inputs are not declared.
	KeepUndeclared bool

	// OnUndeclared is called once for each dropped argument, with the argument
	// name. The engine never writes to stdout, so reporting is the caller's job.
	// It is not called for arguments forwarded under KeepUndeclared.
	OnUndeclared func(name string)
}

// CoerceInputs maps the arguments for a flow onto the flow's declared inputs,
// converting each value to the type the input declares. It returns a new map and
// never modifies args.
//
// Only declared inputs appear in the result, unless KeepUndeclared is set. An
// input that is absent from args, or present as nil, is skipped when it is
// Optional and reported as missing otherwise.
//
// The type vocabulary is JSON Schema's. The recognised types are:
//
//	string   numbers and booleans are formatted; arrays and objects are rejected
//	number   a numeric string is parsed; other non-numbers are rejected
//	integer  as number, plus the value must be a whole number
//	boolean  a "true"/"false" string is parsed; other non-booleans are rejected
//
// Anything else — "array", "object", or a typo such as "str" — passes through
// untouched, because validator/flow.go validates the keys under an input but
// never the value of its type. A flow that says "str" therefore gets no
// coercion at all, silently.
//
// A value that already has the declared type is returned as-is and keeps its Go
// type: an int stays an int, a float64 stays a float64. Only converted values
// are re-typed (a numeric string becomes a float64). Every value this function
// can produce is one ConvertGoValueToLuaValue accepts.
func CoerceInputs(inputs []FlowInput, args map[string]any, opts CoerceOptions) (map[string]any, error) {
	values := make(map[string]any, len(inputs))
	declared := make(map[string]struct{}, len(inputs))

	for _, input := range inputs {
		declared[input.Name] = struct{}{}

		value, provided := args[input.Name]
		if !provided || value == nil {
			if input.Optional {
				continue
			}
			return nil, fmt.Errorf("required input %q not provided", input.Name)
		}

		coerced, err := coerceInputValue(input.Type, value, opts.Strict)
		if err != nil {
			return nil, fmt.Errorf("input %q: %w", input.Name, err)
		}
		values[input.Name] = coerced
	}

	undeclared := make([]string, 0, len(args))
	for name := range args {
		if _, ok := declared[name]; !ok {
			undeclared = append(undeclared, name)
		}
	}
	if len(undeclared) == 0 {
		return values, nil
	}

	sort.Strings(undeclared) // deterministic reporting order
	for _, name := range undeclared {
		if opts.KeepUndeclared {
			values[name] = args[name]
			continue
		}
		if opts.OnUndeclared != nil {
			opts.OnUndeclared(name)
		}
	}
	return values, nil
}

func coerceInputValue(declaredType string, value any, strict bool) (any, error) {
	switch strings.ToLower(strings.TrimSpace(declaredType)) {
	case "string":
		return coerceString(value, strict)
	case "number":
		return coerceNumber(value, strict, false)
	case "integer":
		return coerceNumber(value, strict, true)
	case "boolean":
		return coerceBoolean(value, strict)
	default:
		return value, nil
	}
}

func coerceString(value any, strict bool) (any, error) {
	if text, ok := value.(string); ok {
		return text, nil
	}
	if strict {
		return nil, mismatchError(value, "string")
	}

	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return nil, mismatchError(value, "string")
	}

	switch rv.Kind() {
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		bitSize := 64
		if rv.Kind() == reflect.Float32 {
			bitSize = 32
		}
		return strconv.FormatFloat(rv.Float(), 'g', -1, bitSize), nil
	}
	return nil, mismatchError(value, "string")
}

func coerceNumber(value any, strict bool, integer bool) (any, error) {
	want := "number"
	if integer {
		want = "integer"
	}

	if text, ok := value.(string); ok {
		if strict {
			return nil, mismatchError(value, want)
		}
		number, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil || math.IsNaN(number) || math.IsInf(number, 0) {
			return nil, fmt.Errorf("cannot convert string %q to %s", text, want)
		}
		value = number
	}

	number, ok := numericValue(value)
	if !ok {
		return nil, mismatchError(value, want)
	}
	if integer && number != math.Trunc(number) {
		return nil, fmt.Errorf("cannot convert number %v to integer: not a whole number", number)
	}
	return value, nil
}

func coerceBoolean(value any, strict bool) (any, error) {
	if boolean, ok := value.(bool); ok {
		return boolean, nil
	}

	text, ok := value.(string)
	if !ok || strict {
		return nil, mismatchError(value, "boolean")
	}

	switch strings.ToLower(strings.TrimSpace(text)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return nil, fmt.Errorf("cannot convert string %q to boolean", text)
}

// numericValue reports the value of any Go numeric kind. The returned float is
// for range and integrality checks; callers that hand the value on keep the
// original, so a large int64 does not lose precision on the way to Lua.
func numericValue(value any) (float64, bool) {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return 0, false
	}
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	}
	return 0, false
}

func mismatchError(value any, want string) error {
	return fmt.Errorf("cannot convert %s to %s", jsonKind(value), want)
}

// jsonKind names the JSON type a Go value would have been decoded from, so a
// caller can print something actionable about what it actually sent.
func jsonKind(value any) string {
	if value == nil {
		return "null"
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	}
	return rv.Kind().String()
}
