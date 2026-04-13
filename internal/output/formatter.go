// Package output provides result output writers.
package output

import (
	"fmt"

	"loadcalc/internal/engine"
	"loadcalc/pkg/units"
)

// Writer defines the interface for writing calculation results.
type Writer interface {
	Write(results engine.CalculationResults, dest string) error
}

// FormatIntensity formats an intensity value with its unit label.
func FormatIntensity(value float64, unit units.IntensityUnit) string {
	var label string
	switch unit {
	case units.OpsPerHour:
		label = "ops/h"
	case units.OpsPerMinute:
		label = "ops/m"
	case units.OpsPerSecond:
		label = "ops/s"
	default:
		label = string(unit)
	}
	return fmt.Sprintf("%.2f %s", value, label)
}

// FormatDuration formats seconds into a human-readable duration string.
func FormatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	m := seconds / 60
	s := seconds % 60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm%ds", m, s)
}

// FormatPercent formats a percentage value with 2 decimal places.
func FormatPercent(value float64) string {
	return fmt.Sprintf("%.2f%%", value)
}
