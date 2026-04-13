// Package profile builds test profiles (step sequences) from configuration.
package profile

import (
	"fmt"

	"loadcalc/internal/config"
)

// Step represents a single load step in the test profile.
type Step struct {
	StepNumber      int
	PercentOfTarget float64
	RampupSec       int
	ImpactSec       int
	StabilitySec    int
	RampdownSec     int // 0 = no rampdown
}

// TimelinePhase represents a phase in the absolute timeline.
type TimelinePhase struct {
	PhaseName       string
	StartTimeSec    int
	DurationSec     int
	EndTimeSec      int
	PercentOfTarget float64
}

// Builder builds steps from a TestProfile.
type Builder interface {
	BuildSteps(profile config.TestProfile) ([]Step, error)
}

// NewProfileBuilder returns the appropriate builder for the given profile type.
func NewProfileBuilder(profileType config.ProfileType) (Builder, error) {
	switch profileType {
	case config.ProfileStability:
		return &StabilityProfileBuilder{}, nil
	case config.ProfileCapacity:
		return &CapacityProfileBuilder{}, nil
	case config.ProfileCustom:
		return &CustomProfileBuilder{}, nil
	case config.ProfileSpike:
		return &SpikeProfileBuilder{}, nil
	default:
		return nil, fmt.Errorf("unknown profile type: %q", profileType)
	}
}
