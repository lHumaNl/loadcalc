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
	MultiplierRange float64 // 0 = use default ±25% of base pacing
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

// Optimize finds the pacing (or ops/min/thread) that minimizes the worst-case
// deviation across all steps.
func (o *Optimizer) Optimize(scenario config.Scenario, steps []profile.Step, tool config.Tool, loadModel config.LoadModel, generatorsCount int) (OptimizeResult, error) {
	if loadModel == config.LoadModelOpen {
		return OptimizeResult{}, fmt.Errorf("optimizer is not applicable for open load model")
	}

	multiplier := 3.0
	if scenario.PacingMultiplier != nil {
		multiplier = *scenario.PacingMultiplier
	}

	tolerance := 2.5
	if scenario.DeviationTolerance != nil {
		tolerance = *scenario.DeviationTolerance
	}

	isJMeter := tool == config.ToolJMeter
	generators := generatorsCount
	if generators < 1 {
		generators = 1
	}

	// Normalize target to ops/s
	baseTargetRPS := scenario.TargetIntensity
	switch scenario.IntensityUnit {
	case units.OpsPerHour:
		baseTargetRPS /= 3600
	case units.OpsPerMinute:
		baseTargetRPS /= 60
	case units.OpsPerSecond:
		// already in ops/s
	default:
		return OptimizeResult{}, fmt.Errorf("unknown intensity unit: %s", scenario.IntensityUnit)
	}

	basePacing := float64(scenario.MaxScriptTimeMs) * multiplier

	var low, high float64
	if o.MultiplierRange > 0 {
		low = float64(scenario.MaxScriptTimeMs) * (multiplier - o.MultiplierRange)
		high = float64(scenario.MaxScriptTimeMs) * (multiplier + o.MultiplierRange)
	} else {
		delta := basePacing * 0.25
		low = basePacing - delta
		high = basePacing + delta
	}
	if low < 1 {
		low = 1
	}

	bestScore := math.MaxFloat64
	bestPacing := basePacing
	bestWithinTolerance := false

	for candidatePacing := low; candidatePacing <= high; candidatePacing += 1.0 {
		score, allWithin := evaluateCandidate(candidatePacing, baseTargetRPS, tolerance, steps, isJMeter, generators)
		if score < bestScore {
			bestScore = score
			bestPacing = candidatePacing
			bestWithinTolerance = allWithin
		}
	}

	// Build final step results with best pacing
	stepResults := make([]StepResult, len(steps))
	for i, step := range steps {
		stepTargetRPS := baseTargetRPS * step.PercentOfTarget / 100
		if isJMeter {
			stepTargetRPS /= float64(generators)
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
		result.Warning = fmt.Sprintf("no pacing found within %.2f%% tolerance; best-effort max deviation: %.4f%%", tolerance, bestScore)
	}

	return result, nil
}
