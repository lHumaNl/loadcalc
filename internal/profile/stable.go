package profile

import "loadcalc/internal/config"

// StableProfileBuilder generates a single-step stable load profile.
type StableProfileBuilder struct{}

func (b *StableProfileBuilder) BuildSteps(profile config.TestProfile) ([]Step, error) {
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
