package prompts

import (
	"fmt"
	"sort"
	"strings"

	"github.com/slashbase/layerengine/modules"
)

const (
	modulesUsageInfo = `Each module below is a global table of its own, holding exactly the functions listed for it, with exactly the arguments shown. Calling anything not listed here, or reaching for any other global, is an error.
Use:`

	environmentFacts = `This function runs in a restricted Lua 5.1 environment.
Available: the base library, string, table, math, and the module tables listed below.
Not available: os, io, debug, package, require, loadfile, module, dofile, load,
loadstring, getfenv, setfenv, collectgarbage, newproxy, channel, coroutine.

Lua 5.1 syntax only. Not available: continue, goto, ::label::, // integer division,
bitwise operators (& | ~ << >>), utf8, math.type, string.pack, table.unpack (use unpack).
Write an "if" guard inside the loop instead of continue.
A for loop's control variable does not exist after the loop ends. To use an index
afterwards, store it in a variable declared before the loop.`
)

const (
	promptGuide = `Write a Lua function that takes input parameters, processes the input according to given description and returns output.
Every function body must contain the statements that compute each declared output from the inputs according to "description".
Write code in json field "code".`
)

// Describer is the slice of a module the prompt builder needs: what it is
// called, the surface it declares, and its free-text fallback.
//
// It is deliberately narrower than modules.Module. A module has to implement
// Init — which drags gopher-lua into anything that only wants to render a
// prompt — but the prompt never calls it. Naming the three methods it does use
// keeps the dependency where it belongs and leaves modules.Module free to grow
// without breaking this package.
type Describer interface {
	Name() string
	Manifest() modules.Manifest
	Usage() string
}

// Prompt is the message pair sent to a provider: the system messages, in the
// order they must be sent, and the user message.
//
// It is exported so a caller can see or keep what codegen would send — the
// starting point for caching, replaying, or diffing prompts without reaching
// into the package.
type Prompt struct {
	System []string
	User   string
}

// BuildPrompt assembles the whole prompt for one layer function. It is the one
// place the system/user split and the message order are decided.
func BuildPrompt[T Describer](fnName, description string, inputs []Input, outputs []Output, mods []T) Prompt {
	return Prompt{
		System: []string{GetModuleUsagePrompt(mods), promptGuide},
		User:   generateCodePromptFormat(fnName, description, inputs, outputs),
	}
}

// GetModuleUsagePrompt renders the module surface into the codegen prompt.
//
// Each declared function renders as a single call-shaped line, the way a module
// author would write it by hand:
//
//	versionString:string = system.version() - get system version
//
// A module that returns the zero manifest — or one that does not validate — has
// nothing declared, so it falls back to its Usage() text under a <name>
// heading. Modules and functions are sorted, so the same module set always
// produces the same prompt.
func GetModuleUsagePrompt[T Describer](mods []T) string {
	sorted := make([]Describer, len(mods))
	for i := range mods {
		sorted[i] = mods[i]
	}
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Name() < sorted[j].Name() })

	lines := make([]string, 0, len(sorted))
	for _, m := range sorted {
		lines = append(lines, renderModuleUsage(m)...)
	}

	return fmt.Sprintf("%s\n\n%s\n%s",
		environmentFacts,
		modulesUsageInfo,
		strings.Join(lines, "\n"),
	)
}

// renderModuleUsage renders one module: one line per declared function, plus a
// single line for module-level notes.
func renderModuleUsage(m Describer) []string {
	man := m.Manifest()
	if man.Validate() != nil {
		return renderUsageFallback(m)
	}

	fns := make([]modules.Function, len(man.Functions))
	copy(fns, man.Functions)
	// Sorting by the full name keeps one group's operations adjacent: "." sorts
	// below every character an identifier can contain, so a dotted name orders
	// the way its segments do.
	sort.SliceStable(fns, func(i, j int) bool { return fns[i].Name < fns[j].Name })

	lines := make([]string, 0, len(fns)+1)
	for _, fn := range fns {
		lines = append(lines, renderCallLine(m.Name(), fn))
	}
	if notes := oneLine(man.Notes); notes != "" {
		lines = append(lines, m.Name()+" - "+notes)
	}

	return lines
}

// renderCallLine renders one function the way it is called:
//
//	<returns> = <module>.<fn>(<params>) - <doc>
//
// The module name is the global the module table is bound to, and it is what
// the call is made on: it is not optional, or the line names a function with no
// table to find it in. The assignment is omitted when no return is named. Param
// and return types are inline annotations, so arity and shape survive in a
// single line.
func renderCallLine(module string, fn modules.Function) string {
	var b strings.Builder

	if named := namedReturns(fn.Returns); named != "" {
		b.WriteString(named)
		b.WriteString(" = ")
	}

	b.WriteString(module)
	b.WriteString(".")
	b.WriteString(fn.Name)
	b.WriteString("(")
	for i, p := range fn.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(p.Name)
		if omittable(p) {
			b.WriteString("?")
		}
		if p.Type != "" {
			b.WriteString(":")
			b.WriteString(p.Type)
		}
		if p.Default != "" {
			b.WriteString("=")
			b.WriteString(p.Default)
		}
	}
	b.WriteString(")")

	if doc := oneLine(fn.Doc); doc != "" {
		b.WriteString(" - ")
		b.WriteString(doc)
	}

	return b.String()
}

// omittable reports whether the caller may leave the argument out: it is marked
// Optional, or it declares a Default. Both render the same way, so a default
// always shows up as an optional argument.
func omittable(p modules.Param) bool {
	return p.Optional || p.Default != ""
}

// namedReturns renders "a:type, b:type" for the returns that carry a name, or ""
// when none do.
func namedReturns(returns []modules.Return) string {
	named := false
	for _, r := range returns {
		if r.Name != "" {
			named = true
			break
		}
	}
	if !named {
		return ""
	}

	var b strings.Builder
	for i, r := range returns {
		if i > 0 {
			b.WriteString(", ")
		}
		if r.Name == "" {
			b.WriteString(r.Type)
			continue
		}
		b.WriteString(r.Name)
		if r.Type != "" {
			b.WriteString(":")
			b.WriteString(r.Type)
		}
	}
	return b.String()
}

// renderUsageFallback renders a module that declares no surface: its Usage() text
// under a <name> heading, unchanged from the free-text days.
func renderUsageFallback(m Describer) []string {
	var b strings.Builder
	b.WriteString(m.Name())
	writeCommentBlock(&b, "  ", strings.TrimSpace(m.Usage()))
	return strings.Split(b.String(), "\n")
}

// writeCommentBlock writes text as Lua comment lines, each prefixed by indent.
// Lines that are already comments keep their own marker instead of getting a
// second one, so Usage() text written in Lua style renders cleanly.
func writeCommentBlock(b *strings.Builder, indent, text string) {
	if text == "" {
		return
	}
	for _, line := range strings.Split(text, "\n") {
		b.WriteString("\n")
		b.WriteString(indent)
		line = strings.TrimRight(line, " \t")
		switch {
		case line == "":
			b.WriteString("--")
		case strings.HasPrefix(line, "--"):
			b.WriteString(line)
		default:
			b.WriteString("-- ")
			b.WriteString(line)
		}
	}
}

// oneLine collapses whitespace so a doc string can sit on a single prompt line.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
