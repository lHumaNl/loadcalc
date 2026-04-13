package whatif

import (
	"strings"
	"testing"

	"loadcalc/internal/config"
	"loadcalc/internal/engine"
	"loadcalc/internal/profile"
)

func TestCompareResults_Basic(t *testing.T) {
	baseline := engine.CalculationResults{
		ScenarioResults: []engine.ScenarioResult{
			{
				Scenario: newScenario("A"),
				OptimizeResult: engine.OptimizeResult{
					BestPacingMS:    3000,
					MaxDeviationPct: 2.0,
					StepResults: []engine.StepResult{
						{Step: profile.Step{StepNumber: 1, PercentOfTarget: 100}, Threads: 10, DeviationPct: 2.0},
					},
				},
			},
		},
	}
	modified := engine.CalculationResults{
		ScenarioResults: []engine.ScenarioResult{
			{
				Scenario: newScenario("A"),
				OptimizeResult: engine.OptimizeResult{
					BestPacingMS:    3300,
					MaxDeviationPct: 1.0,
					StepResults: []engine.StepResult{
						{Step: profile.Step{StepNumber: 1, PercentOfTarget: 100}, Threads: 12, DeviationPct: 1.0},
					},
				},
			},
		},
	}

	result := CompareResults(baseline, modified, map[string]string{"global.pacing_multiplier": "4.0"})

	if len(result.Comparisons) != 1 {
		t.Fatalf("expected 1 comparison, got %d", len(result.Comparisons))
	}
	c := result.Comparisons[0]
	if c.Name != "A" {
		t.Errorf("expected name A, got %s", c.Name)
	}
	if c.OldThreads != 10 || c.NewThreads != 12 {
		t.Errorf("threads mismatch: old=%d new=%d", c.OldThreads, c.NewThreads)
	}
	if c.OldPacingMS != 3000 || c.NewPacingMS != 3300 {
		t.Errorf("pacing mismatch: old=%f new=%f", c.OldPacingMS, c.NewPacingMS)
	}
	if c.OldMaxDeviation != 2.0 || c.NewMaxDeviation != 1.0 {
		t.Errorf("deviation mismatch: old=%f new=%f", c.OldMaxDeviation, c.NewMaxDeviation)
	}
}

func TestCompareResults_Improved(t *testing.T) {
	baseline := makeResults("A", 10, 3000, 2.5)
	modified := makeResults("A", 12, 3300, 1.0)

	result := CompareResults(baseline, modified, nil)
	if !result.Comparisons[0].Improved {
		t.Error("expected Improved=true when new deviation < old deviation")
	}
}

func TestCompareResults_Worsened(t *testing.T) {
	baseline := makeResults("A", 10, 3000, 1.0)
	modified := makeResults("A", 12, 3300, 3.0)

	result := CompareResults(baseline, modified, nil)
	if result.Comparisons[0].Improved {
		t.Error("expected Improved=false when new deviation > old deviation")
	}
}

func TestFormatComparison(t *testing.T) {
	result := Result{
		Overrides: map[string]string{"global.pacing_multiplier": "4.0"},
		Comparisons: []ScenarioComparison{
			{Name: "A", OldThreads: 10, NewThreads: 12, OldPacingMS: 3000, NewPacingMS: 3300,
				OldMaxDeviation: 2.5, NewMaxDeviation: 1.0, Improved: true},
		},
	}
	out := FormatComparison(result)
	if !strings.Contains(out, "A") {
		t.Error("expected scenario name in output")
	}
	if !strings.Contains(out, "10") || !strings.Contains(out, "12") {
		t.Error("expected thread counts in output")
	}
}

// helpers

func newScenario(name string) config.Scenario {
	return config.Scenario{Name: name}
}

func makeResults(name string, threads int, pacingMS, maxDev float64) engine.CalculationResults {
	return engine.CalculationResults{
		ScenarioResults: []engine.ScenarioResult{
			{
				Scenario: newScenario(name),
				OptimizeResult: engine.OptimizeResult{
					BestPacingMS:    pacingMS,
					MaxDeviationPct: maxDev,
					StepResults: []engine.StepResult{
						{Step: profile.Step{StepNumber: 1, PercentOfTarget: 100}, Threads: threads, DeviationPct: maxDev},
					},
				},
			},
		},
	}
}
