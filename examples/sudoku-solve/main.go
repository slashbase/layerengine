package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/slashbase/layerengine"
	"github.com/slashbase/layerengine/config"
)

// puzzle is a 9x9 grid with 30 givens and one solution. It is written as nine
// rows for the reader; the parse layer strips the whitespace before counting
// cells, so the layout on screen is not the layout the solver sees.
const puzzle = `530070000
600195000
098000060
800060003
400803001
700020006
060000280
000419005
000080079
`

// brokenPuzzle repeats a 5 in the first row, so the puzzle is illegal before a
// single cell is searched. It is here to show that the check layer runs first
// and the solver is never asked to guess.
const brokenPuzzle = `550070000
600195000
098000060
800060003
400803001
700020006
060000280
000419005
000080079
`

func main() {

	cfg, err := config.FromEnv()
	if err != nil {
		fmt.Println("Error reading config:", err)
		return
	}
	codegenerater, err := cfg.CodeGen()
	if err != nil {
		fmt.Println("Error initializing code generator:", err)
		return
	}
	engine := layerengine.NewLayerEngine(codegenerater)

	data, err := os.ReadFile("./examples/sudoku-solve/flow.yaml")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	err = engine.LoadSpec(string(data))
	if err != nil {
		fmt.Println(err)
		return
	}

	// This example exists to find the ceiling: solve_puzzle has to carry a
	// recursive search through a prose description, which is the hardest thing
	// asked of the model anywhere in examples/. Print each layer's code so the
	// cost of that is visible.
	for _, name := range []string{
		"parse_puzzle",
		"check_puzzle",
		"solve_puzzle",
		"verify_solution",
		"render_board",
	} {
		layer, ok := engine.Layer(name)
		if !ok {
			continue
		}
		lines := strings.Count(strings.TrimSpace(layer.Code), "\n") + 1
		fmt.Printf("---GENERATED LAYER %s (%d lines)---\n%s\n", name, lines, layer.Code)
	}

	fmt.Println("---OUTPUT (legal puzzle, 30 givens)---")
	result, err := engine.RunFlowResult("sudoku_solve", map[string]any{
		"puzzle": puzzle,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	if board, ok := result.Outputs["board"].(string); ok {
		fmt.Println(board)
	}

	// The same grid with a repeated 5 in row 1: check_puzzle reports the
	// conflict, solve_puzzle returns the input unchanged with solved = false,
	// and verify_solution refuses the result rather than passing it through.
	fmt.Println("---OUTPUT (illegal puzzle, repeated 5 in row 1)---")
	result, err = engine.RunFlowResult("sudoku_solve", map[string]any{
		"puzzle": brokenPuzzle,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	if board, ok := result.Outputs["board"].(string); ok {
		fmt.Println(board)
	}
}
