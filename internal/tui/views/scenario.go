package views

import (
	"fmt"
	"strings"

	"loadcalc/internal/engine"
	"loadcalc/internal/tui/styles"
)

// RenderScenario renders the detail view for a single scenario.
func RenderScenario(results engine.CalculationResults, scenarioIdx int) string {
	var b strings.Builder

	if scenarioIdx < 0 || scenarioIdx >= len(results.ScenarioResults) {
		return "No scenario selected."
	}

	sr := results.ScenarioResults[scenarioIdx]
	sc := sr.Scenario

	b.WriteString(styles.TitleStyle.Render(fmt.Sprintf("Scenario: %s", sc.Name)))
	b.WriteString("\n\n")

	loadModel := string(results.Plan.GlobalDefaults.LoadModel)
	if sc.LoadModel != nil {
		loadModel = string(*sc.LoadModel)
	}
	tolerance := results.Plan.GlobalDefaults.DeviationTolerance
	if sc.DeviationTolerance != nil {
		tolerance = *sc.DeviationTolerance
	}
	pacingMult := results.Plan.GlobalDefaults.PacingMultiplier
	if sc.PacingMultiplier != nil {
		pacingMult = *sc.PacingMultiplier
	}

	fmt.Fprintf(&b, "  %-25s %s\n", "Load Model:", loadModel)
	fmt.Fprintf(&b, "  %-25s %.1f %s\n", "Target Intensity:", sc.TargetIntensity, sc.IntensityUnit)
	fmt.Fprintf(&b, "  %-25s %d ms\n", "Max Script Time:", sc.MaxScriptTimeMs)
	fmt.Fprintf(&b, "  %-25s %.1f\n", "Pacing Multiplier:", pacingMult)
	fmt.Fprintf(&b, "  %-25s %.1f%%\n", "Deviation Tolerance:", tolerance)
	if sr.IsBackground {
		fmt.Fprintf(&b, "  %-25s %.0f%%\n", "Background Percent:", sc.BackgroundPercent)
	}

	b.WriteString("\n")
	b.WriteString(styles.HeaderStyle.Render("Optimization Results"))
	b.WriteString("\n")
	if sr.OptimizeResult.BestPacingMS > 0 {
		fmt.Fprintf(&b, "  %-25s %.1f ms\n", "Best Pacing:", sr.OptimizeResult.BestPacingMS)
	}
	if sr.OptimizeResult.BestOpsPerMinPerThread > 0 {
		fmt.Fprintf(&b, "  %-25s %.2f\n", "Best Ops/Min/Thread:", sr.OptimizeResult.BestOpsPerMinPerThread)
	}
	fmt.Fprintf(&b, "  %-25s %s\n", "Max Deviation:", styles.FormatDeviation(sr.OptimizeResult.MaxDeviationPct, tolerance))
	withinStr := "Yes"
	if !sr.OptimizeResult.AllWithinTolerance {
		withinStr = "No"
	}
	fmt.Fprintf(&b, "  %-25s %s\n", "All Within Tolerance:", withinStr)
	if sr.OptimizeResult.Warning != "" {
		fmt.Fprintf(&b, "  %-25s %s\n", "Warning:", sr.OptimizeResult.Warning)
	}

	// Step breakdown
	b.WriteString("\n")
	b.WriteString(styles.HeaderStyle.Render("Step Breakdown"))
	b.WriteString("\n")
	header := fmt.Sprintf("  %6s %8s %12s %8s %12s %10s",
		"Step", "Percent", "TargetRPS", "Threads", "ActualRPS", "Deviation")
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString("  " + strings.Repeat("─", 58))
	b.WriteString("\n")

	for _, step := range sr.OptimizeResult.StepResults {
		devStr := styles.FormatDeviation(step.DeviationPct, tolerance)
		fmt.Fprintf(&b, "  %6d %7.0f%% %12.4f %8d %12.4f %10s\n",
			step.Step.StepNumber, step.Step.PercentOfTarget,
			step.TargetRPS, step.Threads, step.ActualRPS, devStr)
	}

	b.WriteString("\n")
	b.WriteString(styles.StatusBarStyle.Render("[Esc] Back | [S] Steps | [Q] Quit"))
	return b.String()
}
