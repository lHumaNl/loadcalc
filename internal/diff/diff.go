// Package diff provides configuration comparison.
package diff

import (
	"fmt"

	"loadcalc/internal/config"
)

const nilString = "<nil>"

// ChangeType describes how a scenario changed.
type ChangeType string

const (
	Added    ChangeType = "added"
	Removed  ChangeType = "removed"
	Modified ChangeType = "modified"
)

// FieldChange represents a single field difference.
type FieldChange struct {
	Field    string `json:"field"`
	OldValue string `json:"old_value"`
	NewValue string `json:"new_value"`
}

// ScenarioChange represents a scenario-level difference.
type ScenarioChange struct {
	Type     ChangeType    `json:"type"`
	Name     string        `json:"name"`
	Fields   []FieldChange `json:"fields,omitempty"`
	OldIndex int           `json:"old_index"`
	NewIndex int           `json:"new_index"`
}

// Result holds all differences between two test plans.
type Result struct {
	GlobalChanges   []FieldChange    `json:"global_changes"`
	ProfileChanges  []FieldChange    `json:"profile_changes"`
	ScenarioChanges []ScenarioChange `json:"scenario_changes"`
}

// ComparePlans compares two TestPlans and returns the differences.
func ComparePlans(old, newPlan *config.TestPlan) Result {
	var result Result
	result.GlobalChanges = compareGlobals(old.GlobalDefaults, newPlan.GlobalDefaults)
	result.ProfileChanges = compareProfiles(old.Profile, newPlan.Profile)
	result.ScenarioChanges = compareScenarios(old.Scenarios, newPlan.Scenarios)
	return result
}

func compareGlobals(old, newGlobals config.GlobalDefaults) []FieldChange {
	var changes []FieldChange
	addIfDiff := func(field, oldVal, newVal string) {
		if oldVal != newVal {
			changes = append(changes, FieldChange{Field: field, OldValue: oldVal, NewValue: newVal})
		}
	}
	addIfDiff("tool", string(old.Tool), string(newGlobals.Tool))
	addIfDiff("load_model", string(old.LoadModel), string(newGlobals.LoadModel))
	addIfDiff("pacing_multiplier", fmt.Sprintf("%g", old.PacingMultiplier), fmt.Sprintf("%g", newGlobals.PacingMultiplier))
	addIfDiff("deviation_tolerance", fmt.Sprintf("%g", old.DeviationTolerance), fmt.Sprintf("%g", newGlobals.DeviationTolerance))
	addIfDiff("spike_participate", fmt.Sprintf("%t", old.SpikeParticipate), fmt.Sprintf("%t", newGlobals.SpikeParticipate))
	addIfDiff("generators_count", fmt.Sprintf("%d", old.GeneratorsCount), fmt.Sprintf("%d", newGlobals.GeneratorsCount))
	return changes
}

func compareProfiles(old, newProfile config.TestProfile) []FieldChange {
	var changes []FieldChange
	addIfDiff := func(field, oldVal, newVal string) {
		if oldVal != newVal {
			changes = append(changes, FieldChange{Field: field, OldValue: oldVal, NewValue: newVal})
		}
	}
	addIfDiff("type", string(old.Type), string(newProfile.Type))
	addIfDiff("default_rampup_sec", fmt.Sprintf("%d", old.DefaultRampupSec), fmt.Sprintf("%d", newProfile.DefaultRampupSec))
	addIfDiff("default_impact_sec", fmt.Sprintf("%d", old.DefaultImpactSec), fmt.Sprintf("%d", newProfile.DefaultImpactSec))
	addIfDiff("default_stability_sec", fmt.Sprintf("%d", old.DefaultStabilitySec), fmt.Sprintf("%d", newProfile.DefaultStabilitySec))
	addIfDiff("default_rampdown_sec", fmt.Sprintf("%d", old.DefaultRampdownSec), fmt.Sprintf("%d", newProfile.DefaultRampdownSec))
	addIfDiff("start_percent", fmt.Sprintf("%g", old.StartPercent), fmt.Sprintf("%g", newProfile.StartPercent))
	addIfDiff("step_increment", fmt.Sprintf("%g", old.StepIncrement), fmt.Sprintf("%g", newProfile.StepIncrement))
	addIfDiff("num_steps", fmt.Sprintf("%d", old.NumSteps), fmt.Sprintf("%d", newProfile.NumSteps))
	addIfDiff("percent", fmt.Sprintf("%g", old.Percent), fmt.Sprintf("%g", newProfile.Percent))
	addIfDiff("base_percent", fmt.Sprintf("%g", old.BasePercent), fmt.Sprintf("%g", newProfile.BasePercent))
	addIfDiff("num_spikes", fmt.Sprintf("%d", old.NumSpikes), fmt.Sprintf("%d", newProfile.NumSpikes))
	addIfDiff("spiketime_sec", fmt.Sprintf("%d", old.SpiketimeSec), fmt.Sprintf("%d", newProfile.SpiketimeSec))
	addIfDiff("cooldown_sec", fmt.Sprintf("%d", old.CooldownSec), fmt.Sprintf("%d", newProfile.CooldownSec))

	// Compare fine_tune
	oldFT := old.FineTune
	newFT := newProfile.FineTune
	switch {
	case oldFT != nil && newFT != nil:
		addIfDiff("fine_tune.after_percent", fmt.Sprintf("%g", oldFT.AfterPercent), fmt.Sprintf("%g", newFT.AfterPercent))
		addIfDiff("fine_tune.step_increment", fmt.Sprintf("%g", oldFT.StepIncrement), fmt.Sprintf("%g", newFT.StepIncrement))
		addIfDiff("fine_tune.num_steps", fmt.Sprintf("%d", oldFT.NumSteps), fmt.Sprintf("%d", newFT.NumSteps))
	case oldFT != nil:
		changes = append(changes, FieldChange{Field: "fine_tune", OldValue: "present", NewValue: "absent"})
	case newFT != nil:
		changes = append(changes, FieldChange{Field: "fine_tune", OldValue: "absent", NewValue: "present"})
	}

	// Compare steps count
	if len(old.Steps) != len(newProfile.Steps) {
		addIfDiff("steps_count", fmt.Sprintf("%d", len(old.Steps)), fmt.Sprintf("%d", len(newProfile.Steps)))
	}

	return changes
}

func compareScenarios(old, newScenarios []config.Scenario) []ScenarioChange {
	var changes []ScenarioChange

	// Index old scenarios by name
	oldByName := make(map[string]int)
	for i, s := range old {
		oldByName[s.Name] = i
	}
	newByName := make(map[string]int)
	for i, s := range newScenarios {
		newByName[s.Name] = i
	}

	// Check for modified and removed
	for i, oldS := range old {
		if j, ok := newByName[oldS.Name]; ok {
			fields := compareScenarioFields(oldS, newScenarios[j])
			if len(fields) > 0 {
				changes = append(changes, ScenarioChange{
					Type:     Modified,
					Name:     oldS.Name,
					OldIndex: i,
					NewIndex: j,
					Fields:   fields,
				})
			}
		} else {
			changes = append(changes, ScenarioChange{
				Type:     Removed,
				Name:     oldS.Name,
				OldIndex: i,
				NewIndex: -1,
			})
		}
	}

	// Check for added
	for j, newS := range newScenarios {
		if _, ok := oldByName[newS.Name]; !ok {
			changes = append(changes, ScenarioChange{
				Type:     Added,
				Name:     newS.Name,
				OldIndex: -1,
				NewIndex: j,
			})
		}
	}

	return changes
}

func compareScenarioFields(old, newScenario config.Scenario) []FieldChange {
	var changes []FieldChange
	addIfDiff := func(field, oldVal, newVal string) {
		if oldVal != newVal {
			changes = append(changes, FieldChange{Field: field, OldValue: oldVal, NewValue: newVal})
		}
	}
	addIfDiff("script_id", fmt.Sprintf("%d", old.ScriptID), fmt.Sprintf("%d", newScenario.ScriptID))
	addIfDiff("target_intensity", fmt.Sprintf("%g", old.TargetIntensity), fmt.Sprintf("%g", newScenario.TargetIntensity))
	addIfDiff("intensity_unit", string(old.IntensityUnit), string(newScenario.IntensityUnit))
	addIfDiff("max_script_time_ms", fmt.Sprintf("%d", old.MaxScriptTimeMs), fmt.Sprintf("%d", newScenario.MaxScriptTimeMs))
	addIfDiff("background", fmt.Sprintf("%t", old.Background), fmt.Sprintf("%t", newScenario.Background))
	addIfDiff("background_percent", fmt.Sprintf("%g", old.BackgroundPercent), fmt.Sprintf("%g", newScenario.BackgroundPercent))

	// Compare optional overrides
	oldLM := ptrStr(old.LoadModel)
	newLM := ptrStr(newScenario.LoadModel)
	addIfDiff("load_model", oldLM, newLM)

	oldPM := ptrFloat(old.PacingMultiplier)
	newPM := ptrFloat(newScenario.PacingMultiplier)
	addIfDiff("pacing_multiplier", oldPM, newPM)

	oldDT := ptrFloat(old.DeviationTolerance)
	newDT := ptrFloat(newScenario.DeviationTolerance)
	addIfDiff("deviation_tolerance", oldDT, newDT)

	oldSP := ptrBool(old.SpikeParticipate)
	newSP := ptrBool(newScenario.SpikeParticipate)
	addIfDiff("spike_participate", oldSP, newSP)

	return changes
}

func ptrStr[T ~string](p *T) string {
	if p == nil {
		return nilString
	}
	return string(*p)
}

func ptrFloat(p *float64) string {
	if p == nil {
		return nilString
	}
	return fmt.Sprintf("%g", *p)
}

func ptrBool(p *bool) string {
	if p == nil {
		return nilString
	}
	return fmt.Sprintf("%t", *p)
}
