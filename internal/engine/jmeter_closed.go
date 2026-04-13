package engine

import (
	"math"

	"loadcalc/internal/config"
)

// jmeterClosedCalculator implements Calculator for JMeter closed model.
type jmeterClosedCalculator struct {
	generatorsCount int
}

func (c *jmeterClosedCalculator) Calculate(scenario config.Scenario, targetRPS float64) (Result, error) {
	generators := c.generatorsCount
	if generators < 1 {
		generators = 1
	}

	// Step 7: per-generator split — calculate with per-generator target
	perGenRPS := targetRPS / float64(generators)

	multiplier := 3.0
	if scenario.PacingMultiplier != nil {
		multiplier = *scenario.PacingMultiplier
	}

	// Step 1: base pacing
	pacingMS := float64(scenario.MaxScriptTimeMs) * multiplier

	// Step 2: ideal threads
	idealThreads := perGenRPS * (pacingMS / 1000)

	// Step 3: ceil, minimum 1
	threads := int(math.Ceil(idealThreads))
	if threads < 1 {
		threads = 1
	}

	// Step 4: corrected pacing
	correctedPacingMS := (float64(threads) / perGenRPS) * 1000

	// Step 5: actual RPS and deviation
	actualRPS := float64(threads) / (correctedPacingMS / 1000)
	deviationPercent := math.Abs(actualRPS-perGenRPS) / perGenRPS * 100

	// Step 6: ops/min per thread
	opsPerMinPerThread := (perGenRPS / float64(threads)) * 60

	return Result{
		Threads:            threads,
		PacingMS:           correctedPacingMS,
		OpsPerMinPerThread: opsPerMinPerThread,
		ActualRPS:          actualRPS,
		DeviationPercent:   deviationPercent,
		TargetRPS:          perGenRPS,
	}, nil
}
