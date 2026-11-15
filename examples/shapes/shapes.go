package main

import (
	"math"

	"github.com/slashbase/layerengine/modules"
	lua "github.com/yuin/gopher-lua"
)

// Shapes publishes one group per shape, each holding the same two operations.
//
// The grouping is the point of the example: one module answers
// shapes.circle.area(radius) and shapes.rect.area(width, height), so a shape
// is a table in the layer sandbox rather than a module of its own. Generated
// code calls the path; nothing has to know how many shapes exist.
type Shapes struct{}

func (Shapes) Name() string { return "shapes" }

func (Shapes) Init(l *lua.LState) *lua.LTable {
	t := l.NewTable()
	t.RawSetString("circle", circleGroup(l))
	t.RawSetString("rect", rectGroup(l))
	return t
}

// circleGroup builds the "circle" sub-table. One builder per group keeps a
// group's functions together, which is the same shape the manifest declares.
func circleGroup(l *lua.LState) *lua.LTable {
	g := l.NewTable()
	g.RawSetString("area", l.NewFunction(circleArea))
	g.RawSetString("perimeter", l.NewFunction(circlePerimeter))
	return g
}

func rectGroup(l *lua.LState) *lua.LTable {
	g := l.NewTable()
	g.RawSetString("area", l.NewFunction(rectArea))
	g.RawSetString("perimeter", l.NewFunction(rectPerimeter))
	return g
}

// ── circle ───────────────────────────────────────────────────────────────────

// circleArea returns the area of a circle, πr².
func circleArea(l *lua.LState) int {
	r := float64(l.CheckNumber(1))
	l.Push(lua.LNumber(math.Pi * r * r))
	return 1
}

// circlePerimeter returns the circumference of a circle, 2πr.
func circlePerimeter(l *lua.LState) int {
	r := float64(l.CheckNumber(1))
	l.Push(lua.LNumber(2 * math.Pi * r))
	return 1
}

// ── rect ─────────────────────────────────────────────────────────────────────

// rectArea returns the area of a rectangle, w×h.
func rectArea(l *lua.LState) int {
	w := float64(l.CheckNumber(1))
	h := float64(l.CheckNumber(2))
	l.Push(lua.LNumber(w * h))
	return 1
}

// rectPerimeter returns the perimeter of a rectangle, 2(w+h).
func rectPerimeter(l *lua.LState) int {
	w := float64(l.CheckNumber(1))
	h := float64(l.CheckNumber(2))
	l.Push(lua.LNumber(2 * (w + h)))
	return 1
}

// Usage is the free-text fallback, rendered only when the manifest is zero.
// It says the same thing as the manifest, in the shape a caller writes.
func (Shapes) Usage() string {
	return `area = shapes.circle.area(radius)
perimeter = shapes.circle.perimeter(radius)
area = shapes.rect.area(width, height)
perimeter = shapes.rect.perimeter(width, height)
-- Areas are in squared units, perimeters in units.`
}

// Manifest declares the dotted surface: group, then operation. Each name is a
// path into the tables Init builds.
func (Shapes) Manifest() modules.Manifest {
	return modules.Manifest{
		APIVersion: modules.APIVersion,
		Notes:      "Lengths are in whatever unit you pass in: results are in that unit, squared for area.",
		Functions: []modules.Function{
			{
				Name:    "circle.area",
				Doc:     "Area of a circle from its radius.",
				Params:  []modules.Param{{Name: "radius", Type: "number"}},
				Returns: []modules.Return{{Name: "area", Type: "number"}},
			},
			{
				Name:    "circle.perimeter",
				Doc:     "Perimeter of a circle from its radius.",
				Params:  []modules.Param{{Name: "radius", Type: "number"}},
				Returns: []modules.Return{{Name: "perimeter", Type: "number"}},
			},
			{
				Name:    "rect.area",
				Doc:     "Area of a rectangle.",
				Params:  []modules.Param{{Name: "width", Type: "number"}, {Name: "height", Type: "number"}},
				Returns: []modules.Return{{Name: "area", Type: "number"}},
			},
			{
				Name:    "rect.perimeter",
				Doc:     "Perimeter of a rectangle.",
				Params:  []modules.Param{{Name: "width", Type: "number"}, {Name: "height", Type: "number"}},
				Returns: []modules.Return{{Name: "perimeter", Type: "number"}},
			},
		},
	}
}
