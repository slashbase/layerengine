package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/slashbase/layerengine"
	"github.com/slashbase/layerengine/config"
)

// ledger is one month of personal spend for someone in India:
// rent and maintenance, groceries across three shops, a metro card and petrol,
// a few Swiggy orders, the usual bills, a SIP and PPF contribution, and one line
// with a missing field. Amounts are in rupees, categories are not consistently
// cased, and the SIP is counted as an outflow like any other — the flow reports
// money leaving the account, not money invested.
const ledger = `# October 2026 expenses
rent, Rent, 14500
society maintenance, Rent, 2500
big basket, Groceries, 6320
dmart run, groceries, 2840
kirana and vegetables, Groceries, 3110
metro card recharge, Transport, 1000
petrol, Transport, 2400
ola to office, transport, 860
swiggy, Eating Out, 1980
chai and samosa, Eating Out, 640
dinner with friends, Eating Out, 2350
electricity bill, Utilities, 1780
wifi, Utilities, 999
mobile recharge, Utilities, 749
gym membership, Health, 1500
medical, Health, 860
myntra, Shopping, 2999
sip index fund, Investments, 10000
ppf, Investments, 4000
cousin wedding gift, Other, 3100
bad line, Misc
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

	data, err := os.ReadFile("./examples/personal-expenses/flow.yaml")
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	err = engine.LoadSpec(string(data))
	if err != nil {
		fmt.Println(err)
		return
	}

	// Five layers of real logic: parsing, normalising, cap checking, savings
	// arithmetic and formatting. Printing each one shows what the description
	// actually bought before a token goes to a run.
	for _, name := range []string{
		"parse_ledger",
		"apply_category_rules",
		"check_budget_caps",
		"savings_check",
		"render_report",
	} {
		layer, ok := engine.Layer(name)
		if !ok {
			continue
		}
		lines := strings.Count(strings.TrimSpace(layer.Code), "\n") + 1
		fmt.Printf("---GENERATED LAYER %s (%d lines)---\n%s\n", name, lines, layer.Code)
	}

	fmt.Println("---OUTPUT (income 95000, target 30%)---")
	result, err := engine.RunFlowResult("personal_expenses", map[string]any{
		"ledger_text":    ledger,
		"monthly_income": 95000,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	if report, ok := result.Outputs["report"].(string); ok {
		fmt.Println(report)
	}

	// The same month on a smaller salary: the spend is identical, so only the
	// savings branch changes and the verdict drops from "on track" to
	// "overspending".
	fmt.Println("---OUTPUT (income 70000, target 30%)---")
	result, err = engine.RunFlowResult("personal_expenses", map[string]any{
		"ledger_text":    ledger,
		"monthly_income": 70000,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	if report, ok := result.Outputs["report"].(string); ok {
		fmt.Println(report)
	}

	// savings_target is optional and defaults to 30. Asking for 40 on the same
	// 95000 salary is enough to move the verdict off "on track".
	fmt.Println("---OUTPUT (income 95000, target 40%)---")
	result, err = engine.RunFlowResult("personal_expenses", map[string]any{
		"ledger_text":    ledger,
		"monthly_income": 95000,
		"savings_target": 40,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	if report, ok := result.Outputs["report"].(string); ok {
		fmt.Println(report)
	}
}
