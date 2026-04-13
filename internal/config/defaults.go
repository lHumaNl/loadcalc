package config

import "loadcalc/pkg/units"

// ResolveDefaults merges global defaults into scenarios.
// For each scenario, if a per-scenario override pointer is nil, the global default is used.
// Global fields not set in YAML are filled from DefaultGlobalDefaults().
func ResolveDefaults(plan *TestPlan) *TestPlan {
	defaults := DefaultGlobalDefaults()

	// Fill global fields that are zero-valued with spec defaults.
	if plan.GlobalDefaults.PacingMultiplier == 0 {
		plan.GlobalDefaults.PacingMultiplier = defaults.PacingMultiplier
	}
	if plan.GlobalDefaults.DeviationTolerance == 0 {
		plan.GlobalDefaults.DeviationTolerance = defaults.DeviationTolerance
	}
	if plan.GlobalDefaults.GeneratorsCount == 0 {
		plan.GlobalDefaults.GeneratorsCount = defaults.GeneratorsCount
	}
	if plan.GlobalDefaults.LoadModel == "" {
		plan.GlobalDefaults.LoadModel = LoadModelClosed
	}
	// SpikeParticipate default is true; since bool zero is false, we need a different approach.
	// We handle this by always ensuring the global is set and using it for nil scenario overrides.
	// The YAML explicitly sets it. If the entire global block is missing, we use DefaultGlobalDefaults
	// which has SpikeParticipate=true. But we can't distinguish "explicitly set false" from "not set"
	// for a plain bool. The spec default is true, so we only apply it if Tool is also empty (meaning
	// global block was likely empty/missing entirely).
	// For simplicity: if tool is empty (global block likely absent), use default spike_participate.
	// Otherwise trust the parsed value.
	if plan.GlobalDefaults.Tool == "" {
		plan.GlobalDefaults.SpikeParticipate = defaults.SpikeParticipate
	}

	// Merge global defaults into each scenario.
	for i := range plan.Scenarios {
		s := &plan.Scenarios[i]
		if s.IntensityUnit == "" {
			s.IntensityUnit = units.OpsPerHour
		}
		if s.LoadModel == nil {
			lm := plan.GlobalDefaults.LoadModel
			s.LoadModel = &lm
		}
		if s.PacingMultiplier == nil {
			pm := plan.GlobalDefaults.PacingMultiplier
			s.PacingMultiplier = &pm
		}
		if s.DeviationTolerance == nil {
			dt := plan.GlobalDefaults.DeviationTolerance
			s.DeviationTolerance = &dt
		}
		if s.SpikeParticipate == nil {
			sp := plan.GlobalDefaults.SpikeParticipate
			s.SpikeParticipate = &sp
		}
	}

	return plan
}
