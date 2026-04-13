package profile

import "loadcalc/internal/config"

// CustomProfileBuilder generates profiles from explicit user-defined step lists.
type CustomProfileBuilder struct{}

func (b *CustomProfileBuilder) BuildSteps(profile config.TestProfile) ([]Step, error) {
	steps := make([]Step, len(profile.Steps))
	for i, ps := range profile.Steps {
		step := Step{
			StepNumber:      i + 1,
			PercentOfTarget: ps.PercentOfTarget,
			RampupSec:       profile.DefaultRampupSec,
			ImpactSec:       profile.DefaultImpactSec,
			StabilitySec:    profile.DefaultStabilitySec,
			RampdownSec:     profile.DefaultRampdownSec,
		}
		if ps.RampupSec != nil {
			step.RampupSec = *ps.RampupSec
		}
		if ps.ImpactSec != nil {
			step.ImpactSec = *ps.ImpactSec
		}
		if ps.StabilitySec != nil {
			step.StabilitySec = *ps.StabilitySec
		}
		if ps.RampdownSec != nil {
			step.RampdownSec = *ps.RampdownSec
		}
		steps[i] = step
	}
	return steps, nil
}
