package engine

import (
	"loadcalc/internal/config"
	"loadcalc/internal/profile"
)

// Result holds the calculation output for a single scenario at a single intensity step.
type Result struct {
	OutputUnit         string
	Threads            int
	PacingMS           float64
	OpsPerMinPerThread float64
	ActualRPS          float64
	DeviationPercent   float64
	TargetRPS          float64
	OutputValue        float64
}

// OptimizedResult holds the result of multi-step pacing optimization (Phase 5).
type OptimizedResult struct {
	OutputUnit             string
	Threads                int
	PacingMS               float64
	OpsPerMinPerThread     float64
	ActualRPS              float64
	DeviationPercent       float64
	TargetRPS              float64
	OutputValue            float64
	BestPacingMS           float64
	BestOpsPerMinPerThread float64
}

// ScenarioResult holds the full calculation result for one scenario.
type ScenarioResult struct {
	Scenario       config.Scenario
	OptimizeResult OptimizeResult // for closed model (multi-step)
	SingleResult   Result         // for open model or single-step
	IsBackground   bool
	IsOpenModel    bool
}

// CalculationResults is the top-level container consumed by output writers.
type CalculationResults struct {
	Plan            *config.TestPlan
	ScenarioResults []ScenarioResult
	Steps           []profile.Step
	Timeline        []profile.TimelinePhase
}
