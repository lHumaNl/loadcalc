package views

import (
	"fmt"
	"strings"

	"loadcalc/internal/engine"
	"loadcalc/internal/tui/styles"
)

// RenderTimeline renders a simple text timeline of phases.
func RenderTimeline(results engine.CalculationResults) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Test Timeline"))
	b.WriteString("\n\n")

	if len(results.Timeline) == 0 {
		b.WriteString("No timeline phases available.\n")
	} else {
		header := fmt.Sprintf("%-30s %10s %10s %10s %8s",
			"Phase", "Start(s)", "Duration", "End(s)", "Percent")
		b.WriteString(styles.HeaderStyle.Render(header))
		b.WriteString("\n")
		b.WriteString(strings.Repeat("─", 70))
		b.WriteString("\n")

		for _, phase := range results.Timeline {
			fmt.Fprintf(&b, "%-30s %10d %10d %10d %7.0f%%\n",
				phase.PhaseName, phase.StartTimeSec, phase.DurationSec,
				phase.EndTimeSec, phase.PercentOfTarget)
		}
	}

	b.WriteString("\n")
	b.WriteString(styles.StatusBarStyle.Render("[Esc] Back | [Q] Quit"))
	return b.String()
}
