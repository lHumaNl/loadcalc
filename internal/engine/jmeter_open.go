package engine

import (
	"loadcalc/internal/config"
)

const (
	opsPerSecUnit = "ops/sec"
	opsPerMinUnit = "ops/min"
)

// jmeterOpenCalculator implements Calculator for JMeter open model.
type jmeterOpenCalculator struct {
	generatorsCount int
}

func (c *jmeterOpenCalculator) Calculate(scenario config.Scenario, targetRPS float64) (Result, error) {
	_ = scenario // interface requires this parameter
	generators := c.generatorsCount
	if generators < 1 {
		generators = 1
	}

	perGenRPS := targetRPS / float64(generators)

	var outputUnit string
	var outputValue float64

	if perGenRPS >= 0.01 {
		outputUnit = opsPerSecUnit
		outputValue = perGenRPS
	} else {
		outputUnit = opsPerMinUnit
		outputValue = perGenRPS * 60
	}

	return Result{
		TargetRPS:   perGenRPS,
		ActualRPS:   perGenRPS,
		OutputUnit:  outputUnit,
		OutputValue: outputValue,
	}, nil
}
