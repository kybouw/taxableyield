package main

import (
	"fmt"
	"math"
)

// Inputs that in JS came from the form
type Inputs struct {
	// Yields entered by the user (as percentages, e.g., 4.5 for 4.5%)
	FullyTaxable   float64
	Treasury       float64
	NatlTaxExempt  float64
	NatlAmTPct     float64 // AMT-affected portion (%) for national tax-exempt
	StateTaxExempt float64
	StateAmTPct    float64 // AMT-affected portion (%) for state tax-exempt
	AMTFree        float64 // already "after-tax" yield in the original JS

	// Tax settings
	FedBracket   float64 // e.g., 24 for 24%
	StateBracket float64 // e.g., 9.3 for 9.3%
	Itemize      bool    // itemize deductions?
	AMT          bool    // subject to AMT?

	// AMT bracket (radio group in JS). Use 0..4 to match original logic:
	// 0 or 1 => 26%; 2 => 32.5%; 3 => 35%; 4 => 28%
	AMTBracketIndex int
}

// calcAfterTaxYield replicates JS calcAfterTaxYield(yield, fedtaxable, statetaxable, amtpct)
func calcAfterTaxYield(yield float64, fedTaxable, stateTaxable bool, amtPct float64, in Inputs) float64 {
	fed := in.FedBracket
	state := in.StateBracket
	itemize := in.Itemize
	amt := in.AMT

	// AMT logic from the JS
	if amt {
		itemize = false
		switch in.AMTBracketIndex {
		case 0, 1:
			fed = 26
		case 2:
			fed = 32.5
		case 3:
			fed = 35
		case 4:
			fed = 28
		default:
			// fall back to 26 if out of range
			fed = 26
		}
	}

	tax := 0.0

	if fedTaxable {
		tax += fed
	} else if amt {
		// not federally taxable, but a portion is AMT-includable
		tax += (amtPct / 100.0) * fed
	}

	if stateTaxable {
		tax += state
		if itemize {
			// federal deduction for state taxes (reduce fed by state * fed)
			tax -= (state / 100.0) * fed
		}
	}

	return yield * (1.0 - tax/100.0)
}

type Result struct {
	FullyTaxableAfterTax float64
	FullyTaxableTEY      float64
	TreasuryAfterTax     float64
	TreasuryTEY          float64
	NatlAfterTax         float64
	NatlTEY              float64
	StateAfterTax        float64
	StateTEY             float64
	AMTFreeAfterTax      float64
	AMTFreeTEY           float64

	// Pretty, multiline string like the original .result.value
	Text string
}

// Compute does what the JS compute() did
func Compute(in Inputs) Result {
	// After-tax yields
	fullyAT := calcAfterTaxYield(in.FullyTaxable, true, true, 0, in)
	treasuryAT := calcAfterTaxYield(in.Treasury, true, false, 0, in)
	natlAT := calcAfterTaxYield(in.NatlTaxExempt, false, true, in.NatlAmTPct, in)
	stateAT := calcAfterTaxYield(in.StateTaxExempt, false, false, in.StateAmTPct, in)

	// Gross-up factor:
	// If FullyTaxable is NaN in JS, they used 1.0% as a temp; replicate that.
	var grossup float64
	if math.IsNaN(in.FullyTaxable) {
		tmp := 1.0
		tmpAT := calcAfterTaxYield(tmp, true, true, 0, in)
		grossup = tmp / tmpAT
	} else {
		// Avoid divide-by-zero if someone passes a case with fullyAT==0
		if fullyAT == 0 {
			// Fall back to tmp method if needed
			tmp := 1.0
			tmpAT := calcAfterTaxYield(tmp, true, true, 0, in)
			grossup = tmp / tmpAT
		} else {
			grossup = in.FullyTaxable / fullyAT
		}
	}

	// Build display text (3 decimals, with %)
	line := func(label string, afterTax, tey float64) string {
		return fmt.Sprintf("%-18s %6.3f%% after tax, %6.3f%% tax equivalent", label+":", afterTax, tey)
	}

	res := Result{
		FullyTaxableAfterTax: fullyAT,
		FullyTaxableTEY:      in.FullyTaxable, // same as original
		TreasuryAfterTax:     treasuryAT,
		TreasuryTEY:          treasuryAT * grossup,
		NatlAfterTax:         natlAT,
		NatlTEY:              natlAT * grossup,
		StateAfterTax:        stateAT,
		StateTEY:             stateAT * grossup,
		AMTFreeAfterTax:      in.AMTFree, // original JS treated AMT Free as already after-tax
		AMTFreeTEY:           in.AMTFree * grossup,
	}

	res.Text = line("Fully Taxable", res.FullyTaxableAfterTax, res.FullyTaxableTEY) + "\n" +
		line("Treasury", res.TreasuryAfterTax, res.TreasuryTEY) + "\n" +
		line("Nat'l Tax-Exempt", res.NatlAfterTax, res.NatlTEY) + "\n" +
		line("State Tax-Exempt", res.StateAfterTax, res.StateTEY) + "\n" +
		line("AMT Free", res.AMTFreeAfterTax, res.AMTFreeTEY)

	return res
}

func main() {
	// Example usage
	in := Inputs{
		FullyTaxable:   5.000,
		Treasury:       4.500,
		NatlTaxExempt:  3.800,
		NatlAmTPct:     20.0,
		StateTaxExempt: 3.400,
		StateAmTPct:    10.0,
		AMTFree:        3.700,

		FedBracket:      24.0,
		StateBracket:    9.3,
		Itemize:         true,
		AMT:             false,
		AMTBracketIndex: 0, // ignored unless AMT=true
	}

	res := Compute(in)
	fmt.Println(res.Text)
}
