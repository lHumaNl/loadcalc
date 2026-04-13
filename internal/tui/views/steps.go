package views

import (
	"fmt"
	"strings"

	"loadcalc/internal/engine"
	"loadcalc/internal/tui/styles"
)

// RenderSteps renders the step detail view for a given scenario index.
func RenderSteps(results engine.CalculationResults, scenarioIdx int) string {
	var b strings.Builder

	if scenarioIdx < 0 || scenarioIdx >= len(results.ScenarioResults) {
		return "No scenario selected."
	}

	sr := results.ScenarioResults[scenarioIdx]
	b.WriteString(styles.TitleStyle.Render(fmt.Sprintf("Steps: %s", sr.Scenario.Name)))
	b.WriteString("\n\n")

	tolerance := results.Plan.GlobalDefaults.DeviationTolerance
	if sr.Scenario.DeviationTolerance != nil {
		tolerance = *sr.Scenario.DeviationTolerance
	}

	header := fmt.Sprintf("%6s %8s %12s %8s %12s %10s",
		"Step", "Percent", "TargetRPS", "Threads", "ActualRPS", "Deviation")
	b.WriteString(styles.HeaderStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n")

	for _, step := range sr.OptimizeResult.StepResults {
		devStr := styles.FormatDeviation(step.DeviationPct, tolerance)
		row := fmt.Sprintf("%6d %7.0f%% %12.4f %8d %12.4f %10s",
			step.Step.StepNumber, step.Step.PercentOfTarget,
			step.TargetRPS, step.Threads, step.ActualRPS, devStr)
		b.WriteString(row)
		b.WriteString("\n")
	}

	// Timing breakdown
	b.WriteString("\n")
	b.WriteString(styles.HeaderStyle.Render("Timing Breakdown"))
	b.WriteString("\n")
	for _, step := range sr.OptimizeResult.StepResults {
		s := step.Step
		total := s.RampupSec + s.ImpactSec + s.StabilitySec + s.RampdownSec
		fmt.Fprintf(&b, "  Step %d (%.0f%%): rampup=%ds impact=%ds stability=%ds rampdown=%ds total=%ds\n",
			s.StepNumber, s.PercentOfTarget, s.RampupSec, s.ImpactSec, s.StabilitySec, s.RampdownSec, total)
	}

	b.WriteString("\n")
	b.WriteString(styles.StatusBarStyle.Render("[←→] Switch scenario | [Esc] Back | [Q] Quit"))
	return b.String()
}
