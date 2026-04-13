// Package merge provides multi-config merging for loadcalc test plans.
package merge

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"loadcalc/internal/config"
)

// Warning describes a conflict detected during merge.
type Warning struct {
	Field   string
	Message string
}

// Result holds the merged plan and any warnings.
type Result struct {
	Plan     *config.TestPlan
	Warnings []Warning
}

// checkStringConflict returns a warning if two non-empty string values differ.
func checkStringConflict(field, otherVal, baseVal string, configIndex int) *Warning {
	if otherVal != "" && otherVal != baseVal && baseVal != "" {
		return &Warning{
			Field:   field,
			Message: fmt.Sprintf("config %d has %s=%s, keeping %s from first config", configIndex, field, otherVal, baseVal),
		}
	}
	return nil
}

// mergeGlobals checks for conflicts between a subsequent plan's globals and the base globals.
func mergeGlobals(base, other config.GlobalDefaults, configIndex int) []Warning {
	var warnings []Warning
	if w := checkStringConflict("global.tool", string(other.Tool), string(base.Tool), configIndex); w != nil {
		warnings = append(warnings, *w)
	}
	if w := checkStringConflict("global.load_model", string(other.LoadModel), string(base.LoadModel), configIndex); w != nil {
		warnings = append(warnings, *w)
	}
	if other.PacingMultiplier != 0 && other.PacingMultiplier != base.PacingMultiplier && base.PacingMultiplier != 0 {
		warnings = append(warnings, Warning{
			Field:   "global.pacing_multiplier",
			Message: fmt.Sprintf("config %d has pacing_multiplier=%.1f, keeping %.1f from first config", configIndex, other.PacingMultiplier, base.PacingMultiplier),
		})
	}
	if other.DeviationTolerance != 0 && other.DeviationTolerance != base.DeviationTolerance && base.DeviationTolerance != 0 {
		warnings = append(warnings, Warning{
			Field:   "global.deviation_tolerance",
			Message: fmt.Sprintf("config %d has deviation_tolerance=%.1f, keeping %.1f from first config", configIndex, other.DeviationTolerance, base.DeviationTolerance),
		})
	}
	if other.GeneratorsCount != 0 && other.GeneratorsCount != base.GeneratorsCount && base.GeneratorsCount != 0 {
		warnings = append(warnings, Warning{
			Field:   "global.generators_count",
			Message: fmt.Sprintf("config %d has generators_count=%d, keeping %d from first config", configIndex, other.GeneratorsCount, base.GeneratorsCount),
		})
	}
	return warnings
}

// mergeProfile checks for profile type conflicts between a subsequent plan and the merged plan.
func mergeProfile(baseType, otherType config.ProfileType, configIndex int) []Warning {
	var warnings []Warning
	if otherType != "" && otherType != baseType && baseType != "" {
		warnings = append(warnings, Warning{
			Field:   "profile.type",
			Message: fmt.Sprintf("config %d has profile type=%s, keeping %s from first config", configIndex, otherType, baseType),
		})
	}
	return warnings
}

// Plans merges multiple TestPlans into one.
// Globals and Profile come from the first plan. Scenarios are concatenated from all plans in order.
// If subsequent plans have different non-zero global values or profile type, a warning is emitted and the first wins.
func Plans(plans []*config.TestPlan) Result {
	if len(plans) == 0 {
		return Result{Plan: &config.TestPlan{}}
	}

	merged := &config.TestPlan{
		Version:        plans[0].Version,
		GlobalDefaults: plans[0].GlobalDefaults,
		Profile:        plans[0].Profile,
		OutputFormat:   plans[0].OutputFormat,
	}

	var warnings []Warning

	for _, p := range plans {
		merged.Scenarios = append(merged.Scenarios, p.Scenarios...)
	}

	for i := 1; i < len(plans); i++ {
		warnings = append(warnings, mergeGlobals(merged.GlobalDefaults, plans[i].GlobalDefaults, i+1)...)
		warnings = append(warnings, mergeProfile(merged.Profile.Type, plans[i].Profile.Type, i+1)...)
	}

	return Result{
		Plan:     merged,
		Warnings: warnings,
	}
}

// LoadPlansFromDir loads all .yaml/.yml files from a directory, sorted by name.
func LoadPlansFromDir(dir string) ([]*config.TestPlan, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".yaml" || ext == ".yml" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	var plans []*config.TestPlan
	for _, name := range files {
		path := filepath.Join(dir, name)
		plan, err := config.LoadFromYAML(path)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", name, err)
		}
		plans = append(plans, plan)
	}

	return plans, nil
}
