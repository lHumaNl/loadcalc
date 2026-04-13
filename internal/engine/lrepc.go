package engine

import (
	"math"

	"loadcalc/internal/config"
)

// lrepcCalculator implements Calculator for LRE Performance Center (closed model).
type lrepcCalculator struct{}

func (c *lrepcCalculator) Calculate(scenario config.Scenario, targetRPS float64) (Result, error) {
	multiplier := 3.0
	if scenario.PacingMultiplier != nil {
		multiplier = *scenario.PacingMultiplier
	}

	// Step 1: base pacing
	pacingMS := float64(scenario.MaxScriptTimeMs) * multiplier

	// Step 2: ideal threads
	idealThreads := targetRPS * (pacingMS / 1000)

	// Step 3: ceil, minimum 1
	threads := int(math.Ceil(idealThreads))
	if threads < 1 {
		threads = 1
	}

	// Step 4: corrected pacing
	correctedPacingMS := (float64(threads) / targetRPS) * 1000

	// Step 5: actual RPS and deviation
	actualRPS := float64(threads) / (correctedPacingMS / 1000)
	deviationPercent := math.Abs(actualRPS-targetRPS) / targetRPS * 100

	return Result{
		Threads:          threads,
		PacingMS:         correctedPacingMS,
		ActualRPS:        actualRPS,
		DeviationPercent: deviationPercent,
		TargetRPS:        targetRPS,
	}, nil
}
