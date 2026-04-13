// Package views contains individual TUI view components.
package views

import (
	"fmt"
	"strings"

	"loadcalc/internal/engine"
	"loadcalc/internal/tui/styles"
)

// ColumnGroup determines which set of columns to display.
type ColumnGroup int

const (
	ColumnGroupBase ColumnGroup = iota
	ColumnGroupTool
)

// RenderSummary renders the summary table view.
func RenderSummary(results engine.CalculationResults, selectedRow int, colGroup ColumnGroup) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Load Test Summary"))
	b.WriteString("\n\n")

	if colGroup == ColumnGroupBase {
		b.WriteString(renderBaseColumns(results, selectedRow))
	} else {
		b.WriteString(renderToolColumns(results, selectedRow))
	}

	b.WriteString("\n")
	groupLabel := "Base"
	if colGroup == ColumnGroupTool {
		groupLabel = "Tool-specific"
	}
	b.WriteString(styles.StatusBarStyle.Render(fmt.Sprintf("[Tab] Switch columns (showing: %s) | [Enter] Detail | [S] Steps | [T] Timeline | [Q] Quit", groupLabel)))
	return b.String()
}

func renderBaseColumns(results engine.CalculationResults, selectedRow int) string {
	var b strings.Builder
	header := fmt.Sprintf("%-20s %-8s %10s %12s %10s %10s",
		"Name", "Model", "Target", "Pacing/Ops", "Threads", "MaxDev")
	b.WriteString(styles.HeaderStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 72))
	b.WriteString("\n")

	for i, sr := range results.ScenarioResults {
		loadModel := string(results.Plan.GlobalDefaults.LoadModel)
		if sr.Scenario.LoadModel != nil {
			loadModel = string(*sr.Scenario.LoadModel)
		}

		target := fmt.Sprintf("%.1f", sr.Scenario.TargetIntensity)
		pacing := "-"
		threads := "-"
		maxDev := 0.0

		if len(sr.OptimizeResult.StepResults) > 0 {
			threads = fmt.Sprintf("%d", sr.OptimizeResult.StepResults[0].Threads)
			maxDev = sr.OptimizeResult.MaxDeviationPct
		}
		if sr.OptimizeResult.BestPacingMS > 0 {
			pacing = fmt.Sprintf("%.1f ms", sr.OptimizeResult.BestPacingMS)
		} else if sr.OptimizeResult.BestOpsPerMinPerThread > 0 {
			pacing = fmt.Sprintf("%.2f op/m", sr.OptimizeResult.BestOpsPerMinPerThread)
		}

		tolerance := results.Plan.GlobalDefaults.DeviationTolerance
		if sr.Scenario.DeviationTolerance != nil {
			tolerance = *sr.Scenario.DeviationTolerance
		}
		devStr := styles.FormatDeviation(maxDev, tolerance)

		name := sr.Scenario.Name
		if sr.IsBackground {
			name += " [BG]"
		}

		row := fmt.Sprintf("%-20s %-8s %10s %12s %10s %10s",
			truncate(name, 20), loadModel, target, pacing, threads, devStr)

		if i == selectedRow {
			row = styles.SelectedRowStyle.Render(row)
		}
		b.WriteString(row)
		b.WriteString("\n")
	}
	return b.String()
}

func renderToolColumns(results engine.CalculationResults, selectedRow int) string {
	var b strings.Builder
	header := fmt.Sprintf("%-20s %-8s %12s %12s %10s",
		"Name", "Tool", "PacingMS", "Ops/Min/Thr", "Generators")
	b.WriteString(styles.HeaderStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 64))
	b.WriteString("\n")

	for i, sr := range results.ScenarioResults {
		tool := string(results.Plan.GlobalDefaults.Tool)
		pacingMS := fmt.Sprintf("%.1f", sr.OptimizeResult.BestPacingMS)
		opsMin := fmt.Sprintf("%.2f", sr.OptimizeResult.BestOpsPerMinPerThread)
		gens := fmt.Sprintf("%d", results.Plan.GlobalDefaults.GeneratorsCount)

		name := sr.Scenario.Name
		if sr.IsBackground {
			name += " [BG]"
		}

		row := fmt.Sprintf("%-20s %-8s %12s %12s %10s",
			truncate(name, 20), tool, pacingMS, opsMin, gens)

		if i == selectedRow {
			row = styles.SelectedRowStyle.Render(row)
		}
		b.WriteString(row)
		b.WriteString("\n")
	}
	return b.String()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "…"
}
