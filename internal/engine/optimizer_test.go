package engine

import (
	"math"
	"testing"

	"loadcalc/internal/config"
	"loadcalc/internal/profile"
)

func makeOptScenario(targetRPS float64, maxScriptTimeMs int, multiplier, tolerance float64) config.Scenario {
	return config.Scenario{
		Name:               "Test",
		TargetIntensity:    targetRPS,
		IntensityUnit:      "ops_s",
		MaxScriptTimeMs:    maxScriptTimeMs,
		PacingMultiplier:   &multiplier,
		DeviationTolerance: &tolerance,
	}
}

func TestOptimizer_SingleStep(t *testing.T) {
	scenario := makeOptScenario(10.0, 500, 3.0, 2.5)
	steps := []profile.Step{
		{StepNumber: 1, PercentOfTarget: 100},
	}

	opt := &Optimizer{}
	result, err := opt.Optimize(scenario, steps, config.ToolLREPC, config.LoadModelClosed, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.StepResults) != 1 {
		t.Fatalf("expected 1 step result, got %d", len(result.StepResults))
	}

	// Single step should have very low deviation
	if result.MaxDeviationPct > 2.5 {
		t.Errorf("expected low deviation, got %.4f%%", result.MaxDeviationPct)
	}

	if result.BestPacingMS <= 0 {
		t.Errorf("expected positive pacing, got %f", result.BestPacingMS)
	}

	sr := result.StepResults[0]
	if sr.Threads < 1 {
		t.Errorf("expected at least 1 thread, got %d", sr.Threads)
	}
}

func TestOptimizer_MultiStep_SpecExample(t *testing.T) {
	// From spec: target ≈ 3.623 ops/s, script=550ms, multiplier=3, steps=[100%, 200%, 210%]
	// Base pacing = 550 * 3 = 1650ms
	// At 100%: ideal = 3.623 * 1.65 = 5.978 → 6 threads → actual = 6/1.65 = 3.636 → dev ≈ 0.36%
	// At 200%: ideal = 7.246 * 1.65 = 11.956 → 12 threads → actual = 12/1.65 = 7.273 → dev ≈ 0.37%
	// At 210%: ideal = 7.608 * 1.65 = 12.554 → 13 threads → actual = 13/1.65 = 7.879 → dev ≈ 3.56% > 2.5%!
	// Optimizer should find a better pacing where 210% maps more cleanly.

	targetRPS := 3.623
	scenario := makeOptScenario(targetRPS, 550, 3.0, 2.5)
	steps := []profile.Step{
		{StepNumber: 1, PercentOfTarget: 100},
		{StepNumber: 2, PercentOfTarget: 200},
		{StepNumber: 3, PercentOfTarget: 210},
	}

	opt := &Optimizer{}
	result, err := opt.Optimize(scenario, steps, config.ToolLREPC, config.LoadModelClosed, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.StepResults) != 3 {
		t.Fatalf("expected 3 step results, got %d", len(result.StepResults))
	}

	// The optimizer should find a pacing that keeps 210% deviation below tolerance
	// or at least better than the naive 3.56%
	naiveMaxDev := 3.56
	if result.MaxDeviationPct >= naiveMaxDev {
		t.Errorf("optimizer should improve on naive max deviation %.2f%%, got %.4f%%", naiveMaxDev, result.MaxDeviationPct)
	}

	t.Logf("Best pacing: %.1f ms, max deviation: %.4f%%", result.BestPacingMS, result.MaxDeviationPct)
	for _, sr := range result.StepResults {
		t.Logf("  Step %d (%.0f%%): threads=%d, target=%.4f, actual=%.4f, dev=%.4f%%",
			sr.Step.StepNumber, sr.Step.PercentOfTarget, sr.Threads, sr.TargetRPS, sr.ActualRPS, sr.DeviationPct)
	}
}

func TestOptimizer_AllCandidsExceedTolerance_Warning(t *testing.T) {
	// Use a very tight tolerance that can't be met
	scenario := makeOptScenario(3.623, 550, 3.0, 0.001)
	steps := []profile.Step{
		{StepNumber: 1, PercentOfTarget: 100},
		{StepNumber: 2, PercentOfTarget: 200},
		{StepNumber: 3, PercentOfTarget: 210},
	}

	opt := &Optimizer{}
	result, err := opt.Optimize(scenario, steps, config.ToolLREPC, config.LoadModelClosed, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Warning == "" {
		t.Error("expected warning when no candidate passes tolerance")
	}
	if result.AllWithinTolerance {
		t.Error("expected AllWithinTolerance to be false")
	}

	// Should still return best-effort result
	if result.BestPacingMS <= 0 {
		t.Error("expected best-effort pacing even with warning")
	}
}

func TestOptimizer_ResultAtLeastAsGoodAsBasePacing(t *testing.T) {
	// Property test: optimizer result should be at least as good as base pacing
	targetRPS := 5.0
	scenario := makeOptScenario(targetRPS, 800, 3.0, 5.0)
	steps := []profile.Step{
		{StepNumber: 1, PercentOfTarget: 100},
		{StepNumber: 2, PercentOfTarget: 150},
		{StepNumber: 3, PercentOfTarget: 175},
		{StepNumber: 4, PercentOfTarget: 200},
	}

	opt := &Optimizer{}
	result, err := opt.Optimize(scenario, steps, config.ToolLREPC, config.LoadModelClosed, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Compute naive max deviation at base pacing
	basePacing := 800.0 * 3.0
	naiveMaxDev := 0.0
	for _, step := range steps {
		stepRPS := targetRPS * step.PercentOfTarget / 100
		ideal := stepRPS * basePacing / 1000
		// Try ceil, floor, round
		candidates := []int{
			int(math.Ceil(ideal)),
			int(math.Floor(ideal)),
			int(math.Round(ideal)),
		}
		bestDev := math.MaxFloat64
		for _, c := range candidates {
			if c < 1 {
				c = 1
			}
			actual := float64(c) / (basePacing / 1000)
			dev := math.Abs(actual-stepRPS) / stepRPS * 100
			if dev < bestDev {
				bestDev = dev
			}
		}
		if bestDev > naiveMaxDev {
			naiveMaxDev = bestDev
		}
	}

	if result.MaxDeviationPct > naiveMaxDev+0.0001 {
		t.Errorf("optimizer result (%.4f%%) should be <= naive base pacing (%.4f%%)",
			result.MaxDeviationPct, naiveMaxDev)
	}
}

func TestOptimizer_JMeterClosed(t *testing.T) {
	scenario := makeOptScenario(10.0, 500, 3.0, 2.5)
	steps := []profile.Step{
		{StepNumber: 1, PercentOfTarget: 100},
		{StepNumber: 2, PercentOfTarget: 200},
	}

	opt := &Optimizer{}
	result, err := opt.Optimize(scenario, steps, config.ToolJMeter, config.LoadModelClosed, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.BestPacingMS <= 0 {
		t.Error("expected positive pacing")
	}
	if result.BestOpsPerMinPerThread <= 0 {
		t.Error("expected positive ops/min/thread for JMeter")
	}

	// Verify ops/min/thread = 60 / (pacing_ms / 1000)
	expectedOps := 60.0 / (result.BestPacingMS / 1000)
	if math.Abs(result.BestOpsPerMinPerThread-expectedOps) > 0.001 {
		t.Errorf("ops/min/thread mismatch: got %f, expected %f", result.BestOpsPerMinPerThread, expectedOps)
	}

	// JMeter target RPS should be split by generators
	sr := result.StepResults[0]
	expectedTarget := 10.0 * 100 / 100 / 3.0
	if math.Abs(sr.TargetRPS-expectedTarget) > 0.001 {
		t.Errorf("expected per-generator target %.4f, got %.4f", expectedTarget, sr.TargetRPS)
	}
}

func TestOptimizer_OpenModel_Error(t *testing.T) {
	scenario := makeOptScenario(10.0, 500, 3.0, 2.5)
	steps := []profile.Step{
		{StepNumber: 1, PercentOfTarget: 100},
	}

	opt := &Optimizer{}
	_, err := opt.Optimize(scenario, steps, config.ToolJMeter, config.LoadModelOpen, 1)
	if err == nil {
		t.Error("expected error for open model")
	}
}

func TestOptimizer_MultiplierRange_AffectsSearchRange(t *testing.T) {
	// With explicit RangeDown/RangeUp, the optimizer uses asymmetric search bounds.
	scenario := makeOptScenario(3.623, 550, 3.0, 2.5)
	steps := []profile.Step{
		{StepNumber: 1, PercentOfTarget: 100},
		{StepNumber: 2, PercentOfTarget: 200},
		{StepNumber: 3, PercentOfTarget: 210},
	}

	// Both zero uses default ±25% of basePacing
	optDefault := &Optimizer{}
	resultDefault, err := optDefault.Optimize(scenario, steps, config.ToolLREPC, config.LoadModelClosed, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Explicit asymmetric range: down=0.2, up=0.5 → pacing range [550*(3.0-0.2), 550*(3.0+0.5)] = [1540, 1925]
	optExplicit := &Optimizer{MultiplierRangeDown: 0.2, MultiplierRangeUp: 0.5}
	resultExplicit, err := optExplicit.Optimize(scenario, steps, config.ToolLREPC, config.LoadModelClosed, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both should produce valid results
	if resultDefault.BestPacingMS <= 0 {
		t.Error("default: expected positive pacing")
	}
	if resultExplicit.BestPacingMS <= 0 {
		t.Error("explicit: expected positive pacing")
	}

	t.Logf("Default pacing: %.1f (dev %.4f%%), Explicit pacing: %.1f (dev %.4f%%)",
		resultDefault.BestPacingMS, resultDefault.MaxDeviationPct,
		resultExplicit.BestPacingMS, resultExplicit.MaxDeviationPct)
}

func TestOptimizer_AsymmetricRange(t *testing.T) {
	// Test that asymmetric range (small down, large up) searches the expected bounds.
	scenario := makeOptScenario(10.0, 500, 3.0, 2.5)
	steps := []profile.Step{
		{StepNumber: 1, PercentOfTarget: 100},
		{StepNumber: 2, PercentOfTarget: 150},
	}

	// down=0.1, up=1.0 → pacing range [500*(3.0-0.1), 500*(3.0+1.0)] = [1450, 2000]
	opt := &Optimizer{MultiplierRangeDown: 0.1, MultiplierRangeUp: 1.0}
	result, err := opt.Optimize(scenario, steps, config.ToolLREPC, config.LoadModelClosed, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.BestPacingMS < 1450 || result.BestPacingMS > 2000 {
		t.Errorf("expected pacing in [1450, 2000], got %.1f", result.BestPacingMS)
	}
}

func TestOptimizer_MultiplierRange_Zero_PreservesOldBehavior(t *testing.T) {
	scenario := makeOptScenario(10.0, 500, 3.0, 2.5)
	steps := []profile.Step{
		{StepNumber: 1, PercentOfTarget: 100},
	}

	// Both RangeDown and RangeUp = 0 should use old ±25% logic
	opt := &Optimizer{MultiplierRangeDown: 0, MultiplierRangeUp: 0}
	result, err := opt.Optimize(scenario, steps, config.ToolLREPC, config.LoadModelClosed, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// basePacing = 500*3 = 1500, range [1125, 1875]
	// Result should be within that range
	if result.BestPacingMS < 1125 || result.BestPacingMS > 1875 {
		t.Errorf("expected pacing in [1125, 1875], got %.1f", result.BestPacingMS)
	}
}

func TestOptimizer_VeryLowIntensity(t *testing.T) {
	// Very low intensity: 1 thread at every step
	scenario := makeOptScenario(0.5, 500, 3.0, 10.0)
	steps := []profile.Step{
		{StepNumber: 1, PercentOfTarget: 50},
		{StepNumber: 2, PercentOfTarget: 100},
		{StepNumber: 3, PercentOfTarget: 150},
	}

	opt := &Optimizer{}
	result, err := opt.Optimize(scenario, steps, config.ToolLREPC, config.LoadModelClosed, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, sr := range result.StepResults {
		if sr.Threads < 1 {
			t.Errorf("step %d: threads should be >= 1, got %d", sr.Step.StepNumber, sr.Threads)
		}
	}
}
