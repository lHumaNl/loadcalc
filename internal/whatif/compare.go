package whatif

import (
	"fmt"
	"strings"

	"loadcalc/internal/engine"
)

// ScenarioComparison holds the before/after comparison for one scenario.
type ScenarioComparison struct {
	Name            string
	OldThreads      int
	NewThreads      int
	OldPacingMS     float64
	NewPacingMS     float64
	OldMaxDeviation float64
	NewMaxDeviation float64
	Improved        bool // true if new deviation < old deviation
}

// Result holds the full what-if comparison output.
type Result struct {
	Overrides   map[string]string
	Comparisons []ScenarioComparison
}

// CompareResults builds a Result from baseline and modified calculation results.
func CompareResults(baseline, modified engine.CalculationResults, overrides map[string]string) Result {
	result := Result{Overrides: overrides}

	// Build map of modified results by name for matching
	modMap := make(map[string]engine.ScenarioResult)
	for _, sr := range modified.ScenarioResults {
		modMap[sr.Scenario.Name] = sr
	}

	for _, bsr := range baseline.ScenarioResults {
		msr, ok := modMap[bsr.Scenario.Name]
		if !ok {
			continue
		}

		oldThreads := 0
		newThreads := 0
		if len(bsr.OptimizeResult.StepResults) > 0 {
			oldThreads = bsr.OptimizeResult.StepResults[0].Threads
		}
		if len(msr.OptimizeResult.StepResults) > 0 {
			newThreads = msr.OptimizeResult.StepResults[0].Threads
		}

		c := ScenarioComparison{
			Name:            bsr.Scenario.Name,
			OldThreads:      oldThreads,
			NewThreads:      newThreads,
			OldPacingMS:     bsr.OptimizeResult.BestPacingMS,
			NewPacingMS:     msr.OptimizeResult.BestPacingMS,
			OldMaxDeviation: bsr.OptimizeResult.MaxDeviationPct,
			NewMaxDeviation: msr.OptimizeResult.MaxDeviationPct,
			Improved:        msr.OptimizeResult.MaxDeviationPct < bsr.OptimizeResult.MaxDeviationPct,
		}
		result.Comparisons = append(result.Comparisons, c)
	}

	return result
}

// FormatComparison returns a formatted table comparing old vs new results.
func FormatComparison(result Result) string {
	var sb strings.Builder

	sb.WriteString("What-If Comparison\n")
	sb.WriteString("Overrides:\n")
	for k, v := range result.Overrides {
		fmt.Fprintf(&sb, "  %s = %s\n", k, v)
	}
	sb.WriteString("\n")

	fmt.Fprintf(&sb, "%-20s %10s %10s %12s %12s %10s %10s %8s\n",
		"Scenario", "OldThr", "NewThr", "OldPacing", "NewPacing", "OldDev%", "NewDev%", "Status")
	sb.WriteString(strings.Repeat("-", 94) + "\n")

	for _, c := range result.Comparisons {
		status := "\033[31mWorse\033[0m"
		if c.Improved {
			status = "\033[32mBetter\033[0m"
		}
		fmt.Fprintf(&sb, "%-20s %10d %10d %12.1f %12.1f %9.2f%% %9.2f%% %s\n",
			c.Name, c.OldThreads, c.NewThreads,
			c.OldPacingMS, c.NewPacingMS,
			c.OldMaxDeviation, c.NewMaxDeviation, status)
	}

	return sb.String()
}
