package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/slashbase/layerengine/modules"
)

const (
	modulesUsageInfo = `The only global table is 'layer'. It contains exactly the modules and functions listed below, with exactly the arguments shown. Calling anything not listed here, or reaching for any other global, is an error.
Use:`

	environmentFacts = `This function runs in a restricted Lua 5.1 environment.
Available: the base library, string, table, math, and the layer module table.
Not available: os, io, debug, package, require, loadfile, module, dofile, load,
loadstring, getfenv, setfenv, collectgarbage, newproxy, channel, coroutine.`
)

const (
	promptGuide = "Write a Lua function that takes input parameters, processes the input according to given description and returns output.\nWrite code in json field \"code\"."
)

// GetModuleUsagePrompt renders the module surface into the codegen prompt.
//
// Each declared function renders as a single call-shaped line, the way a module
// author would write it by hand:
//
//	versionString:string = layer.system.version() - get system version
//
// A module that returns the zero manifest — or one that does not validate — has
// nothing declared, so it falls back to its Usage() text under a layer.<name>
// heading. Modules and functions are sorted, so the same module set always
// produces the same prompt.
func GetModuleUsagePrompt(mods []modules.Module) string {
	sorted := make([]modules.Module, len(mods))
	copy(sorted, mods)
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
func renderModuleUsage(m modules.Module) []string {
	man := m.Manifest()
	if man.Validate() != nil {
		return renderUsageFallback(m)
	}

	fns := make([]modules.Function, len(man.Functions))
	copy(fns, man.Functions)
	sort.SliceStable(fns, func(i, j int) bool { return fns[i].Name < fns[j].Name })

	lines := make([]string, 0, len(fns)+1)
	for _, fn := range fns {
		lines = append(lines, renderCallLine(m.Name(), fn))
	}
	if notes := oneLine(man.Notes); notes != "" {
		lines = append(lines, "layer."+m.Name()+" - "+notes)
	}

	return lines
}

// renderCallLine renders one function the way it is called:
//
//	<returns> = layer.<module>.<fn>(<params>) - <doc>
//
// The assignment is omitted when no return is named. Param and return types are
// inline annotations, so arity and shape survive in a single line.
func renderCallLine(module string, fn modules.Function) string {
	var b strings.Builder

	if named := namedReturns(fn.Returns); named != "" {
		b.WriteString(named)
		b.WriteString(" = ")
	}

	b.WriteString("layer.")
	b.WriteString(module)
	b.WriteString(".")
	b.WriteString(fn.Name)
	b.WriteString("(")
	for i, p := range fn.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(p.Name)
		if p.Optional {
			b.WriteString("?")
		}
		if p.Type != "" {
			b.WriteString(":")
			b.WriteString(p.Type)
		}
	}
	b.WriteString(")")

	if doc := oneLine(fn.Doc); doc != "" {
		b.WriteString(" - ")
		b.WriteString(doc)
	}

	return b.String()
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
// under a layer.<name> heading, unchanged from the free-text days.
func renderUsageFallback(m modules.Module) []string {
	var b strings.Builder
	b.WriteString("layer.")
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
