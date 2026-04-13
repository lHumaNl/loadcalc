// Package units provides intensity unit conversion utilities (ops/h, ops/m, ops/s).
package units

import (
	"errors"
	"fmt"
)

// IntensityUnit represents the unit of measurement for load intensity.
type IntensityUnit string

const (
	OpsPerHour   IntensityUnit = "ops_h"
	OpsPerMinute IntensityUnit = "ops_m"
	OpsPerSecond IntensityUnit = "ops_s"
)

// NormalizeToOpsPerSec converts a value from the given unit to ops/sec.
// Returns an error for negative values or unknown units.
func NormalizeToOpsPerSec(value float64, unit IntensityUnit) (float64, error) {
	if value < 0 {
		return 0, errors.New("intensity value must be non-negative")
	}
	switch unit {
	case OpsPerHour:
		return value / 3600, nil
	case OpsPerMinute:
		return value / 60, nil
	case OpsPerSecond:
		return value, nil
	default:
		return 0, fmt.Errorf("unknown intensity unit: %s", unit)
	}
}

// ConvertFromOpsPerSec converts ops/sec back to the specified unit for display.
func ConvertFromOpsPerSec(opsPerSec float64, unit IntensityUnit) float64 {
	switch unit {
	case OpsPerHour:
		return opsPerSec * 3600
	case OpsPerMinute:
		return opsPerSec * 60
	case OpsPerSecond:
		return opsPerSec
	default:
		return opsPerSec
	}
}
