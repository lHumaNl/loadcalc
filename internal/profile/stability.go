package profile

import "loadcalc/internal/config"

// StabilityProfileBuilder generates a single-step stability load profile.
type StabilityProfileBuilder struct{}

func (b *StabilityProfileBuilder) BuildSteps(profile config.TestProfile) ([]Step, error) {
	pct := profile.Percent
	if pct == 0 {
		pct = 100
	}
	return []Step{
		{
			StepNumber:      1,
			PercentOfTarget: pct,
			RampupSec:       profile.DefaultRampupSec,
			ImpactSec:       profile.DefaultImpactSec,
			StabilitySec:    profile.DefaultStabilitySec,
			RampdownSec:     profile.DefaultRampdownSec,
		},
	}, nil
}
