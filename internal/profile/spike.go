package profile

import (
	"fmt"

	"loadcalc/internal/config"
)

// SpikeProfileBuilder generates spike test profiles with alternating base and spike levels.
type SpikeProfileBuilder struct{}

func (b *SpikeProfileBuilder) BuildSteps(profile config.TestProfile) ([]Step, error) {
	if profile.NumSpikes <= 0 {
		return nil, fmt.Errorf("num_spikes must be > 0")
	}

	var steps []Step
	stepNum := 1

	// Step 1: rampup to base, impact at base, stability at base
	steps = append(steps, Step{
		StepNumber:      stepNum,
		PercentOfTarget: profile.BasePercent,
		RampupSec:       profile.DefaultRampupSec,
		ImpactSec:       profile.DefaultImpactSec,
		StabilitySec:    profile.DefaultStabilitySec,
		RampdownSec:     0,
	})
	stepNum++

	// For each spike
	for i := 0; i < profile.NumSpikes; i++ {
		spikePercent := profile.BasePercent + profile.SpikeStartIncrement + float64(i)*profile.SpikeIncrementGrowth

		// Spike step: rampup to spike level, impact, stability, rampdown to base
		steps = append(steps, Step{
			StepNumber:      stepNum,
			PercentOfTarget: spikePercent,
			RampupSec:       profile.DefaultRampupSec,
			ImpactSec:       profile.SpiketimeSec,
			StabilitySec:    profile.DefaultStabilitySec,
			RampdownSec:     profile.DefaultRampdownSec,
		})
		stepNum++

		// Cooldown step: back at base percent
		steps = append(steps, Step{
			StepNumber:      stepNum,
			PercentOfTarget: profile.BasePercent,
			RampupSec:       0,
			ImpactSec:       profile.CooldownSec,
			StabilitySec:    0,
			RampdownSec:     0,
		})
		stepNum++
	}

	return steps, nil
}
