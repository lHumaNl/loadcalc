package profile

import "fmt"

// BuildTimeline converts steps into a flat timeline of phases with absolute times.
func BuildTimeline(steps []Step) []TimelinePhase {
	var phases []TimelinePhase
	currentTime := 0

	for _, step := range steps {
		prefix := fmt.Sprintf("Step %d", step.StepNumber)

		if step.RampupSec > 0 {
			phases = append(phases, TimelinePhase{
				PhaseName:       prefix + " Rampup",
				StartTimeSec:    currentTime,
				DurationSec:     step.RampupSec,
				EndTimeSec:      currentTime + step.RampupSec,
				PercentOfTarget: step.PercentOfTarget,
			})
			currentTime += step.RampupSec
		}

		if step.ImpactSec > 0 {
			phases = append(phases, TimelinePhase{
				PhaseName:       prefix + " Impact",
				StartTimeSec:    currentTime,
				DurationSec:     step.ImpactSec,
				EndTimeSec:      currentTime + step.ImpactSec,
				PercentOfTarget: step.PercentOfTarget,
			})
			currentTime += step.ImpactSec
		}

		if step.StabilitySec > 0 {
			phases = append(phases, TimelinePhase{
				PhaseName:       prefix + " Stability",
				StartTimeSec:    currentTime,
				DurationSec:     step.StabilitySec,
				EndTimeSec:      currentTime + step.StabilitySec,
				PercentOfTarget: step.PercentOfTarget,
			})
			currentTime += step.StabilitySec
		}

		if step.RampdownSec > 0 {
			phases = append(phases, TimelinePhase{
				PhaseName:       prefix + " Rampdown",
				StartTimeSec:    currentTime,
				DurationSec:     step.RampdownSec,
				EndTimeSec:      currentTime + step.RampdownSec,
				PercentOfTarget: step.PercentOfTarget,
			})
			currentTime += step.RampdownSec
		}
	}

	return phases
}
