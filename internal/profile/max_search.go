package profile

import "loadcalc/internal/config"

// MaxSearchProfileBuilder generates incrementing step sequences for capacity search.
type MaxSearchProfileBuilder struct{}

func (b *MaxSearchProfileBuilder) BuildSteps(profile config.TestProfile) ([]Step, error) {
	// Generate percents from start + increment * N
	var percents []float64
	for i := 0; i < profile.NumSteps; i++ {
		percents = append(percents, profile.StartPercent+profile.StepIncrement*float64(i))
	}

	// Fine tune: additional steps after a threshold with different increment
	if profile.FineTune != nil {
		ft := profile.FineTune
		for i := 0; i < ft.NumSteps; i++ {
			percents = append(percents, ft.AfterPercent+ft.StepIncrement*float64(i+1))
		}
	}

	// Build step override lookup by percent
	overrides := make(map[float64]config.ProfileStep)
	for _, s := range profile.Steps {
		overrides[s.PercentOfTarget] = s
	}

	steps := make([]Step, len(percents))
	for i, pct := range percents {
		step := Step{
			StepNumber:      i + 1,
			PercentOfTarget: pct,
			RampupSec:       profile.DefaultRampupSec,
			ImpactSec:       profile.DefaultImpactSec,
			StabilitySec:    profile.DefaultStabilitySec,
			RampdownSec:     profile.DefaultRampdownSec,
		}
		if ov, ok := overrides[pct]; ok {
			if ov.RampupSec != nil {
				step.RampupSec = *ov.RampupSec
			}
			if ov.ImpactSec != nil {
				step.ImpactSec = *ov.ImpactSec
			}
			if ov.StabilitySec != nil {
				step.StabilitySec = *ov.StabilitySec
			}
			if ov.RampdownSec != nil {
				step.RampdownSec = *ov.RampdownSec
			}
		}
		steps[i] = step
	}
	return steps, nil
}
