// Package engine implements the core load test parameter calculation logic.
package engine

import (
	"fmt"

	"loadcalc/internal/config"
)

// Calculator computes load test parameters for a single scenario at a given target RPS.
type Calculator interface {
	Calculate(scenario config.Scenario, targetRPS float64) (Result, error)
}

// NewCalculator creates the appropriate calculator based on tool and load model.
func NewCalculator(tool config.Tool, loadModel config.LoadModel, generatorsCount int) (Calculator, error) {
	switch tool {
	case config.ToolLREPC:
		if loadModel == config.LoadModelOpen {
			return nil, fmt.Errorf("open load model is not supported for LRE PC")
		}
		return &lrepcCalculator{}, nil
	case config.ToolJMeter:
		if loadModel == config.LoadModelOpen {
			return &jmeterOpenCalculator{generatorsCount: generatorsCount}, nil
		}
		return &jmeterClosedCalculator{generatorsCount: generatorsCount}, nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", tool)
	}
}
