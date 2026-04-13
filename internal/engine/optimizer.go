package engine

import (
	"fmt"
	"math"

	"loadcalc/internal/config"
	"loadcalc/internal/profile"
	"loadcalc/pkg/units"
)

// StepResult holds optimization results for a single step.
type StepResult struct {
	Step         profile.Step
	TargetRPS    float64
	Threads      int
	ActualRPS    float64
	DeviationPct float64
}

// OptimizeResult holds the overall pacing optimization result.
type OptimizeResult struct {
	Warning                string
	StepResults            []StepResult
	BestPacingMS           float64
	BestOpsPerMinPerThread float64
	MaxDeviationPct        float64
	AllWithinTolerance     bool
}

// Optimizer performs brute-force pacing optimization across all steps.
type Optimizer struct {
	MultiplierRangeDown float64 // how much below base multiplier to search (e.g., 0.2)
	MultiplierRangeUp   float64 // how much above base multiplier to search (e.g., 0.5)
}

// bestDeviationForStep computes the minimum deviation achievable for a step
// by trying ceil/floor/round thread counts.
func bestDeviationForStep(stepTargetRPS, candidatePacing float64) float64 {
	ideal := stepTargetRPS * candidatePacing / 1000
	pacingSec := candidatePacing / 1000

	candidates := [3]int{
		int(math.Ceil(ideal)),
		int(math.Floor(ideal)),
		int(math.Round(ideal)),
	}

	bestDev := math.MaxFloat64
	for _, threads := range candidates {
		if threads < 1 {
			threads = 1
		}
		actualRPS := float64(threads) / pacingSec
		dev := math.Abs(actualRPS-stepTargetRPS) / stepTargetRPS * 100
		if dev < bestDev {
			bestDev = dev
		}
	}
	return bestDev
}

// evaluateCandidate scores a candidate pacing across all steps, returning the
// worst-case deviation and whether all steps are within tolerance.
func evaluateCandidate(candidatePacing, baseTargetRPS, tolerance float64, steps []profile.Step, isJMeter bool, generators int) (score float64, withinTolerance bool) {
	withinTolerance = true
	for _, step := range steps {
		stepTargetRPS := baseTargetRPS * step.PercentOfTarget / 100
		if isJMeter {
			stepTargetRPS /= float64(generators)
		}
		dev := bestDeviationForStep(stepTargetRPS, candidatePacing)
		if dev > score {
			score = dev
		}
		if dev > tolerance {
			withinTolerance = false
		}
	}
	return score, withinTolerance
}

// bestThreadsForStep picks the thread count (from ceil/floor/round) that
// minimizes deviation, returning threads, actualRPS, and deviation.
func bestThreadsForStep(stepTargetRPS, pacingMS float64) (threads int, actualRPS, deviationPct float64) {
	ideal := stepTargetRPS * pacingMS / 1000
	pacingSec := pacingMS / 1000

	candidates := [3]int{
		int(math.Ceil(ideal)),
		int(math.Floor(ideal)),
		int(math.Round(ideal)),
	}

	threads = 1
	bestDev := math.MaxFloat64
	for _, t := range candidates {
		if t < 1 {
			t = 1
		}
		actual := float64(t) / pacingSec
		dev := math.Abs(actual-stepTargetRPS) / stepTargetRPS * 100
		if dev < bestDev {
			bestDev = dev
			threads = t
			actualRPS = actual
		}
	}
	deviationPct = bestDev
	return threads, actualRPS, deviationPct
}

type optimizeParams struct {
	multiplier    float64
	tolerance     float64
	isJMeter      bool
	generators    int
	baseTargetRPS float64
	scriptTimeMs  int
}

func (o *Optimizer) resolveParams(scenario config.Scenario, tool config.Tool, generatorsCount int) (optimizeParams, error) {
	p := optimizeParams{
		multiplier:   3.0,
		tolerance:    2.5,
		isJMeter:     tool == config.ToolJMeter,
		generators:   generatorsCount,
		scriptTimeMs: scenario.MaxScriptTimeMs,
	}
	if scenario.PacingMultiplier != nil {
		p.multiplier = *scenario.PacingMultiplier
	}
	if scenario.DeviationTolerance != nil {
		p.tolerance = *scenario.DeviationTolerance
	}
	if p.generators < 1 {
		p.generators = 1
	}
	p.baseTargetRPS = scenario.TargetIntensity
	switch scenario.IntensityUnit {
	case units.OpsPerHour:
		p.baseTargetRPS /= 3600
	case units.OpsPerMinute:
		p.baseTargetRPS /= 60
	case units.OpsPerSecond:
		// already ops/s.
	default:
		return p, fmt.Errorf("unknown intensity unit: %s", scenario.IntensityUnit)
	}
	return p, nil
}

func (o *Optimizer) searchBounds(p optimizeParams) (low, high float64) {
	basePacing := float64(p.scriptTimeMs) * p.multiplier
	if o.MultiplierRangeDown > 0 || o.MultiplierRangeUp > 0 {
		low = float64(p.scriptTimeMs) * (p.multiplier - o.MultiplierRangeDown)
		high = float64(p.scriptTimeMs) * (p.multiplier + o.MultiplierRangeUp)
	} else {
		delta := basePacing * 0.25
		low = basePacing - delta
		high = basePacing + delta
	}
	if low < 1 {
		low = 1
	}
	return low, high
}

// Optimize finds the pacing (or ops/min/thread) that minimizes the worst-case
// deviation across all steps.
func (o *Optimizer) Optimize(scenario config.Scenario, steps []profile.Step, tool config.Tool, loadModel config.LoadModel, generatorsCount int) (OptimizeResult, error) {
	if loadModel == config.LoadModelOpen {
		return OptimizeResult{}, fmt.Errorf("optimizer is not applicable for open load model")
	}

	params, err := o.resolveParams(scenario, tool, generatorsCount)
	if err != nil {
		return OptimizeResult{}, err
	}

	low, high := o.searchBounds(params)
	basePacing := float64(scenario.MaxScriptTimeMs) * params.multiplier

	bestScore := math.MaxFloat64
	bestPacing := basePacing
	bestWithinTolerance := false

	for candidatePacing := low; candidatePacing <= high; candidatePacing += 1.0 {
		score, allWithin := evaluateCandidate(candidatePacing, params.baseTargetRPS, params.tolerance, steps, params.isJMeter, params.generators)
		if score < bestScore {
			bestScore = score
			bestPacing = candidatePacing
			bestWithinTolerance = allWithin
		}
	}

	// Build final step results with best pacing.
	stepResults := make([]StepResult, len(steps))
	for i, step := range steps {
		stepTargetRPS := params.baseTargetRPS * step.PercentOfTarget / 100
		if params.isJMeter {
			stepTargetRPS /= float64(params.generators)
		}
		threads, actualRPS, dev := bestThreadsForStep(stepTargetRPS, bestPacing)
		stepResults[i] = StepResult{
			Step:         step,
			TargetRPS:    stepTargetRPS,
			Threads:      threads,
			ActualRPS:    actualRPS,
			DeviationPct: dev,
		}
	}

	result := OptimizeResult{
		BestPacingMS:       bestPacing,
		StepResults:        stepResults,
		MaxDeviationPct:    bestScore,
		AllWithinTolerance: bestWithinTolerance,
	}

	result.BestOpsPerMinPerThread = 60.0 / (bestPacing / 1000)

	if !bestWithinTolerance {
		result.Warning = fmt.Sprintf("no pacing found within %.2f%% tolerance; best-effort max deviation: %.4f%%", params.tolerance, bestScore)
	}

	return result, nil
}
