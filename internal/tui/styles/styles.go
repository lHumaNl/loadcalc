// Package styles defines shared color schemes and styling constants for the TUI.
package styles

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Deviation thresholds.
const (
	DeviationGreenMax  = 1.0
	DeviationYellowMax = 2.5
)

// Colors for deviation indicators.
var (
	ColorGreen  = lipgloss.Color("#00CC00")
	ColorYellow = lipgloss.Color("#CCCC00")
	ColorRed    = lipgloss.Color("#CC0000")
	ColorWhite  = lipgloss.Color("#FFFFFF")
	ColorGray   = lipgloss.Color("#888888")
	ColorCyan   = lipgloss.Color("#00CCCC")
)

// Symbols for accessibility.
const (
	SymbolOK       = "✓"
	SymbolWarning  = "⚠"
	SymbolExceeded = "✗"
)

// DeviationLevel classifies a deviation percentage.
type DeviationLevel int

const (
	DeviationOK DeviationLevel = iota
	DeviationWarn
	DeviationExceeded
)

// ClassifyDeviation returns the level for a given deviation and tolerance.
func ClassifyDeviation(deviationPct, tolerance float64) DeviationLevel {
	abs := deviationPct
	if abs < 0 {
		abs = -abs
	}
	if abs < DeviationGreenMax {
		return DeviationOK
	}
	if abs <= DeviationYellowMax && abs <= tolerance {
		return DeviationWarn
	}
	return DeviationExceeded
}

// DeviationSymbol returns the accessibility symbol for a deviation level.
func DeviationSymbol(level DeviationLevel) string {
	switch level {
	case DeviationOK:
		return SymbolOK
	case DeviationWarn:
		return SymbolWarning
	case DeviationExceeded:
		return SymbolExceeded
	default:
		return ""
	}
}

// DeviationColor returns the lipgloss color for a deviation level.
func DeviationColor(level DeviationLevel) lipgloss.Color {
	switch level {
	case DeviationOK:
		return ColorGreen
	case DeviationWarn:
		return ColorYellow
	case DeviationExceeded:
		return ColorRed
	default:
		return ""
	}
}

// Shared styles.
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorCyan).
			MarginBottom(1)

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite)

	SelectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#333333")).
				Foreground(ColorWhite)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorGray)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorGray).
			MarginTop(1)
)

// FormatDeviation renders deviation with color and symbol as a styled string.
func FormatDeviation(deviationPct, tolerance float64) string {
	level := ClassifyDeviation(deviationPct, tolerance)
	sym := DeviationSymbol(level)
	color := DeviationColor(level)
	style := lipgloss.NewStyle().Foreground(color)
	return style.Render(fmt.Sprintf("%s %.2f%%", sym, deviationPct))
}
