// Package e2e provides end-to-end integration tests for the loadcalc pipeline.
package e2e

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"loadcalc/internal/config"
	"loadcalc/internal/engine"
	"loadcalc/internal/integration"
	"loadcalc/internal/output"
	"loadcalc/internal/profile"
	"loadcalc/pkg/units"

	"github.com/xuri/excelize/v2"
)

// testdataDir returns the absolute path to the testdata directory.
func testdataDir() string {
	// Tests run from the package dir; testdata is at repo root.
	return filepath.Join("..", "..", "testdata")
}

// runFullPipeline loads a YAML config and runs the entire calculation pipeline.
func runFullPipeline(t *testing.T, yamlPath string) (engine.CalculationResults, []profile.Step) {
	t.Helper()

	plan, err := config.LoadFromYAML(yamlPath)
	if err != nil {
		t.Fatalf("LoadFromYAML(%s): %v", yamlPath, err)
	}

	plan = config.ResolveDefaults(plan)

	valErrs := config.Validate(plan)
	if config.HasErrors(valErrs) {
		for _, e := range valErrs {
			t.Logf("validation: %s", e.String())
		}
		t.Fatalf("validation failed for %s", yamlPath)
	}

	builder, err := profile.NewProfileBuilder(plan.Profile.Type)
	if err != nil {
		t.Fatalf("NewProfileBuilder: %v", err)
	}
	steps, err := builder.BuildSteps(plan.Profile)
	if err != nil {
		t.Fatalf("BuildSteps: %v", err)
	}

	var scenarioResults []engine.ScenarioResult
	for _, scenario := range plan.Scenarios {
		sr, err := calculateScenario(scenario, steps, plan.GlobalDefaults)
		if err != nil {
			t.Fatalf("calculateScenario(%s): %v", scenario.Name, err)
		}
		scenarioResults = append(scenarioResults, sr)
	}

	timeline := profile.BuildTimeline(steps)

	results := engine.CalculationResults{
		Plan:            plan,
		Steps:           steps,
		Timeline:        timeline,
		ScenarioResults: scenarioResults,
	}
	return results, steps
}

// calculateScenario mirrors cmd/loadcalc/main.go logic.
func calculateScenario(scenario config.Scenario, steps []profile.Step, globals config.GlobalDefaults) (engine.ScenarioResult, error) {
	loadModel := globals.LoadModel
	if scenario.LoadModel != nil {
		loadModel = *scenario.LoadModel
	}

	sr := engine.ScenarioResult{
		Scenario:     scenario,
		IsBackground: scenario.Background,
		IsOpenModel:  loadModel == config.LoadModelOpen,
	}

	targetRPS, err := units.NormalizeToOpsPerSec(scenario.TargetIntensity, scenario.IntensityUnit)
	if err != nil {
		return sr, err
	}

	if scenario.Background {
		bgRPS := targetRPS * scenario.BackgroundPercent / 100
		calc, err := engine.NewCalculator(globals.Tool, loadModel, globals.GeneratorsCount)
		if err != nil {
			return sr, err
		}
		result, err := calc.Calculate(scenario, bgRPS)
		if err != nil {
			return sr, err
		}
		sr.SingleResult = result
		sr.OptimizeResult = engine.OptimizeResult{
			BestPacingMS:           result.PacingMS,
			BestOpsPerMinPerThread: result.OpsPerMinPerThread,
			MaxDeviationPct:        result.DeviationPercent,
			AllWithinTolerance:     true,
			StepResults: []engine.StepResult{
				{
					Step:         profile.Step{StepNumber: 1, PercentOfTarget: scenario.BackgroundPercent},
					TargetRPS:    result.TargetRPS,
					Threads:      result.Threads,
					ActualRPS:    result.ActualRPS,
					DeviationPct: result.DeviationPercent,
				},
			},
		}
		return sr, nil
	}

	if loadModel == config.LoadModelOpen {
		calc, err := engine.NewCalculator(globals.Tool, loadModel, globals.GeneratorsCount)
		if err != nil {
			return sr, err
		}
		var stepResults []engine.StepResult
		for _, step := range steps {
			stepRPS := targetRPS * step.PercentOfTarget / 100
			result, err := calc.Calculate(scenario, stepRPS)
			if err != nil {
				return sr, err
			}
			stepResults = append(stepResults, engine.StepResult{
				Step:         step,
				TargetRPS:    result.TargetRPS,
				Threads:      result.Threads,
				ActualRPS:    result.ActualRPS,
				DeviationPct: result.DeviationPercent,
			})
			if sr.SingleResult == (engine.Result{}) {
				sr.SingleResult = result
			}
		}
		sr.OptimizeResult = engine.OptimizeResult{StepResults: stepResults}
		return sr, nil
	}

	opt := &engine.Optimizer{}
	optResult, err := opt.Optimize(scenario, steps, globals.Tool, loadModel, globals.GeneratorsCount)
	if err != nil {
		return sr, err
	}
	sr.OptimizeResult = optResult
	return sr, nil
}

// --- Full Pipeline Tests ---

func TestFullPipeline_MaxSearch(t *testing.T) {
	results, steps := runFullPipeline(t, filepath.Join(testdataDir(), "capacity_config.yaml"))

	// 5 steps: 50, 75, 100, 125, 150
	if len(steps) != 5 {
		t.Errorf("expected 5 steps, got %d", len(steps))
	}
	expectedPercents := []float64{50, 75, 100, 125, 150}
	for i, exp := range expectedPercents {
		if steps[i].PercentOfTarget != exp {
			t.Errorf("step %d: expected %.0f%%, got %.0f%%", i+1, exp, steps[i].PercentOfTarget)
		}
	}

	// 4 scenarios
	if len(results.ScenarioResults) != 4 {
		t.Fatalf("expected 4 scenario results, got %d", len(results.ScenarioResults))
	}

	// Main page, Test page: closed model, should have step results for each step
	for _, name := range []string{"Main page", "Test page"} {
		sr := findScenario(results, name)
		if sr == nil {
			t.Fatalf("scenario %s not found", name)
		}
		if len(sr.OptimizeResult.StepResults) != 5 {
			t.Errorf("%s: expected 5 step results, got %d", name, len(sr.OptimizeResult.StepResults))
		}
		for _, stepRes := range sr.OptimizeResult.StepResults {
			if stepRes.Threads < 1 {
				t.Errorf("%s step %d: threads must be >= 1, got %d", name, stepRes.Step.StepNumber, stepRes.Threads)
			}
			if stepRes.TargetRPS <= 0 {
				t.Errorf("%s step %d: target RPS must be > 0", name, stepRes.Step.StepNumber)
			}
		}
	}

	// 404 page: background, single step result
	uc03 := findScenario(results, "404 page")
	if !uc03.IsBackground {
		t.Error("UC03 should be background")
	}
	if len(uc03.OptimizeResult.StepResults) != 1 {
		t.Errorf("UC03: expected 1 step result (background), got %d", len(uc03.OptimizeResult.StepResults))
	}

	// API health check: open model
	uc04 := findScenario(results, "API health check")
	if !uc04.IsOpenModel {
		t.Error("UC04 should be open model")
	}
	if len(uc04.OptimizeResult.StepResults) != 5 {
		t.Errorf("UC04: expected 5 step results (open model scales per step), got %d", len(uc04.OptimizeResult.StepResults))
	}

	// Timeline should have phases
	if len(results.Timeline) == 0 {
		t.Error("timeline should not be empty")
	}
}

func TestFullPipeline_MaxSearchFineTune(t *testing.T) {
	results, steps := runFullPipeline(t, filepath.Join(testdataDir(), "capacity_finetune_config.yaml"))

	// 6 steps: 100, 200, 300, 310, 320, 330
	if len(steps) != 6 {
		t.Errorf("expected 6 steps, got %d", len(steps))
	}
	expectedPercents := []float64{100, 200, 300, 310, 320, 330}
	for i, exp := range expectedPercents {
		if steps[i].PercentOfTarget != exp {
			t.Errorf("step %d: expected %.0f%%, got %.0f%%", i+1, exp, steps[i].PercentOfTarget)
		}
	}

	// Each closed-model scenario should have 6 step results
	for _, sr := range results.ScenarioResults {
		if !sr.IsOpenModel && !sr.IsBackground {
			if len(sr.OptimizeResult.StepResults) != 6 {
				t.Errorf("%s: expected 6 step results, got %d", sr.Scenario.Name, len(sr.OptimizeResult.StepResults))
			}
		}
	}
}

func TestFullPipeline_Custom(t *testing.T) {
	results, steps := runFullPipeline(t, filepath.Join(testdataDir(), "custom_config.yaml"))

	// 5 steps: 100, 200, 100, 300, 200
	if len(steps) != 5 {
		t.Errorf("expected 5 steps, got %d", len(steps))
	}
	expectedPercents := []float64{100, 200, 100, 300, 200}
	for i, exp := range expectedPercents {
		if steps[i].PercentOfTarget != exp {
			t.Errorf("step %d: expected %.0f%%, got %.0f%%", i+1, exp, steps[i].PercentOfTarget)
		}
	}

	// Custom step 3 has stability_sec override of 600
	if steps[2].StabilitySec != 600 {
		t.Errorf("step 3: expected stability 600, got %d", steps[2].StabilitySec)
	}
	// Other steps use default 300
	if steps[0].StabilitySec != 300 {
		t.Errorf("step 1: expected stability 300 (default), got %d", steps[0].StabilitySec)
	}

	// Rampdown should be 0 (default_rampdown_sec: 0)
	for i, step := range steps {
		if step.RampdownSec != 0 {
			t.Errorf("step %d: expected rampdown 0, got %d", i+1, step.RampdownSec)
		}
	}

	if len(results.ScenarioResults) != 2 {
		t.Errorf("expected 2 scenarios, got %d", len(results.ScenarioResults))
	}
}

func TestFullPipeline_Spike(t *testing.T) {
	results, steps := runFullPipeline(t, filepath.Join(testdataDir(), "spike_config.yaml"))

	// Spike: 1 base + 3*(spike+cooldown) = 7 steps
	if len(steps) != 7 {
		t.Errorf("expected 7 steps, got %d", len(steps))
	}

	// Base step at 70%
	if steps[0].PercentOfTarget != 70 {
		t.Errorf("base step: expected 70%%, got %.0f%%", steps[0].PercentOfTarget)
	}

	// Spike percents: 120, 130, 140
	expectedSpikes := []float64{120, 130, 140}
	for i, exp := range expectedSpikes {
		spikeStep := steps[1+i*2] // indices 1, 3, 5
		if spikeStep.PercentOfTarget != exp {
			t.Errorf("spike %d: expected %.0f%%, got %.0f%%", i+1, exp, spikeStep.PercentOfTarget)
		}
	}

	// Cooldown steps at base 70%
	for i := 0; i < 3; i++ {
		cooldown := steps[2+i*2] // indices 2, 4, 6
		if cooldown.PercentOfTarget != 70 {
			t.Errorf("cooldown %d: expected 70%%, got %.0f%%", i+1, cooldown.PercentOfTarget)
		}
	}

	// Background monitor is background: single step result regardless of profile steps
	uc02 := findScenario(results, "Background monitor")
	if !uc02.IsBackground {
		t.Error("UC02 should be background")
	}
	if len(uc02.OptimizeResult.StepResults) != 1 {
		t.Errorf("UC02 background: expected 1 step result, got %d", len(uc02.OptimizeResult.StepResults))
	}

	// UC03 has spike_participate=false but is closed model, so optimizer still runs on all steps
	// The spike_participate flag affects which steps it actually ramps to in execution,
	// but at calculation level, the optimizer still calculates all steps.
	uc03 := findScenario(results, "Non-spike endpoint")
	if len(uc03.OptimizeResult.StepResults) != 7 {
		t.Errorf("UC03: expected 7 step results, got %d", len(uc03.OptimizeResult.StepResults))
	}
}

func TestFullPipeline_Stable(t *testing.T) {
	results, steps := runFullPipeline(t, filepath.Join(testdataDir(), "stability_config.yaml"))

	// Single step at 100%
	if len(steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(steps))
	}
	if steps[0].PercentOfTarget != 100 {
		t.Errorf("expected 100%%, got %.0f%%", steps[0].PercentOfTarget)
	}
	if steps[0].RampupSec != 120 {
		t.Errorf("expected rampup 120, got %d", steps[0].RampupSec)
	}
	if steps[0].StabilitySec != 1800 {
		t.Errorf("expected stability 1800, got %d", steps[0].StabilitySec)
	}

	if len(results.ScenarioResults) != 1 {
		t.Errorf("expected 1 scenario result, got %d", len(results.ScenarioResults))
	}

	sr := results.ScenarioResults[0]
	if len(sr.OptimizeResult.StepResults) != 1 {
		t.Errorf("expected 1 step result, got %d", len(sr.OptimizeResult.StepResults))
	}
	if sr.OptimizeResult.StepResults[0].Threads < 1 {
		t.Error("threads must be >= 1")
	}
}

func TestFullPipeline_LREPC(t *testing.T) {
	results, steps := runFullPipeline(t, filepath.Join(testdataDir(), "lrepc_config.yaml"))

	if len(steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(steps))
	}

	// Main transaction: 13044 ops/h, 550ms, x3 => ~6 threads
	uc01 := findScenario(results, "Main transaction")
	if uc01 == nil {
		t.Fatal("UC01 not found")
	}
	if len(uc01.OptimizeResult.StepResults) != 1 {
		t.Fatalf("expected 1 step result, got %d", len(uc01.OptimizeResult.StepResults))
	}
	threads01 := uc01.OptimizeResult.StepResults[0].Threads
	// With optimizer searching pacing range, threads may differ from simple ceil calculation.
	// For single-step stability profile, expect threads in reasonable range (5-7).
	if threads01 < 5 || threads01 > 7 {
		t.Errorf("UC01: expected threads in range 5-7, got %d", threads01)
	}

	// Low intensity check: 75 ops/h, 50ms, x3 => 1 thread
	uc02 := findScenario(results, "Low intensity check")
	if uc02 == nil {
		t.Fatal("Low intensity check not found")
	}
	threads02 := uc02.OptimizeResult.StepResults[0].Threads
	if threads02 != 1 {
		t.Errorf("UC02: expected 1 thread, got %d", threads02)
	}
}

// --- Cross-validation with Appendix A ---

func TestAppendixA_JMeter_ReferenceExcel(t *testing.T) {
	// From spec Appendix A: JMeter, closed model, 3 generators
	// UC01: 720000 ops/h, 1100ms, x3 => pacing 3300, threads 220/3gen=73.33->ceil, CTT 18.18
	// UC02: 90000 ops/h, 1000ms, x3 => pacing 3000, threads 25/3gen=8.33->ceil, CTT 20.0
	// UC03: 90000 ops/h, 200ms, x3 => pacing 600, threads 5/3gen=1.67->ceil, CTT 100.0
	// UC04: 90000 ops/h, 1100ms, x3 => pacing 3300, threads 27.5/3gen=9.17->ceil, CTT 18.18

	testCases := []struct {
		name             string
		targetOpsH       float64
		scriptMs         int
		multiplier       float64
		generators       int
		expectedBasePace float64 // base pacing before optimization
	}{
		{"UC01", 720000, 1100, 3.0, 3, 3300},
		{"UC02", 90000, 1000, 3.0, 3, 3000},
		{"UC03", 90000, 200, 3.0, 3, 600},
		{"UC04", 90000, 1100, 3.0, 3, 3300},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			basePacing := float64(tc.scriptMs) * tc.multiplier
			if basePacing != tc.expectedBasePace {
				t.Errorf("base pacing: expected %.0f, got %.0f", tc.expectedBasePace, basePacing)
			}

			targetRPS := tc.targetOpsH / 3600
			perGenRPS := targetRPS / float64(tc.generators)
			idealThreads := perGenRPS * basePacing / 1000
			threads := int(math.Ceil(idealThreads))

			if threads < 1 {
				t.Errorf("threads must be >= 1")
			}

			// Verify threads match reference
			switch tc.name {
			case "UC01":
				// perGenRPS = 720000/3600/3 = 66.6667, * 3.3 = 220.0 (exact in theory).
				// Due to float64 precision, ideal may be 220.00000000000003, so ceil = 221.
				// Reference says 220; accept 220 or 221.
				if threads != 220 && threads != 221 {
					t.Errorf("UC01: expected 220 or 221 threads per gen, got %d (ideal=%.15f)", threads, idealThreads)
				}
			case "UC02":
				// 90000/3600/3 = 8.333, * 3.0 = 25.0
				if threads != 25 {
					t.Errorf("UC02: expected 25 threads per gen, got %d (ideal=%.2f)", threads, idealThreads)
				}
			case "UC03":
				// 90000/3600/3 = 8.333, * 0.6 = 5.0
				if threads != 5 {
					t.Errorf("UC03: expected 5 threads per gen, got %d (ideal=%.2f)", threads, idealThreads)
				}
			case "UC04":
				// 90000/3600/3 = 8.333, * 3.3 = 27.5 -> ceil = 28
				if threads != 28 {
					t.Errorf("UC04: expected 28 threads per gen, got %d (ideal=%.2f)", threads, idealThreads)
				}
			}
		})
	}
}

func TestAppendixA_LREPC_Conversation(t *testing.T) {
	// Case 1: 13044 ops/h, 550ms, x3 => 6 threads, corrected pacing ~1656ms
	t.Run("Case1_13044opsh", func(t *testing.T) {
		targetRPS := 13044.0 / 3600 // 3.6233...
		pacing := 550.0 * 3.0       // 1650
		ideal := targetRPS * pacing / 1000
		threads := int(math.Ceil(ideal))
		if threads != 6 {
			t.Errorf("expected 6 threads, got %d (ideal=%.4f)", threads, ideal)
		}

		correctedPacing := float64(threads) / targetRPS * 1000
		if math.Abs(correctedPacing-1656.0) > 1.0 {
			t.Errorf("corrected pacing: expected ~1656, got %.1f", correctedPacing)
		}
	})

	// Case 2: 75 ops/h, 50ms, x3 => 1 thread, corrected pacing 48000ms
	t.Run("Case2_75opsh", func(t *testing.T) {
		targetRPS := 75.0 / 3600 // 0.02083...
		pacing := 50.0 * 3.0     // 150
		ideal := targetRPS * pacing / 1000
		threads := int(math.Ceil(ideal))
		if threads < 1 {
			threads = 1
		}
		if threads != 1 {
			t.Errorf("expected 1 thread, got %d (ideal=%.6f)", threads, ideal)
		}

		correctedPacing := float64(threads) / targetRPS * 1000
		if math.Abs(correctedPacing-48000.0) > 1.0 {
			t.Errorf("corrected pacing: expected ~48000, got %.1f", correctedPacing)
		}
	})
}

// --- Profile Type Tests ---

func TestProfileTypes_StepCounts(t *testing.T) {
	tests := []struct {
		name          string
		configFile    string
		expectedSteps int
	}{
		{"capacity_5steps", "capacity_config.yaml", 5},
		{"capacity_finetune_6steps", "capacity_finetune_config.yaml", 6},
		{"custom_5steps", "custom_config.yaml", 5},
		{"spike_7steps", "spike_config.yaml", 7},
		{"stability_1step", "stability_config.yaml", 1},
		{"lrepc_1step", "lrepc_config.yaml", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, steps := runFullPipeline(t, filepath.Join(testdataDir(), tt.configFile))
			if len(steps) != tt.expectedSteps {
				t.Errorf("expected %d steps, got %d", tt.expectedSteps, len(steps))
			}
		})
	}
}

// --- Mixed Scenarios Tests ---

func TestMixedScenarios_ClosedOpenBackground(t *testing.T) {
	results, _ := runFullPipeline(t, filepath.Join(testdataDir(), "capacity_config.yaml"))

	var hasClosed, hasOpen, hasBackground bool
	for _, sr := range results.ScenarioResults {
		switch {
		case sr.IsBackground:
			hasBackground = true
		case sr.IsOpenModel:
			hasOpen = true
		default:
			hasClosed = true
		}
	}

	if !hasClosed {
		t.Error("expected at least one closed model scenario")
	}
	if !hasOpen {
		t.Error("expected at least one open model scenario")
	}
	if !hasBackground {
		t.Error("expected at least one background scenario")
	}
}

// --- Background Scenarios ---

func TestBackground_FixedIntensity(t *testing.T) {
	results, _ := runFullPipeline(t, filepath.Join(testdataDir(), "spike_config.yaml"))

	uc02 := findScenario(results, "Background monitor")
	if uc02 == nil {
		t.Fatal("Background monitor not found")
	}
	if !uc02.IsBackground {
		t.Fatal("Background monitor should be background")
	}

	// Background scenario should have exactly 1 step result
	if len(uc02.OptimizeResult.StepResults) != 1 {
		t.Errorf("background scenario should have 1 step result, got %d", len(uc02.OptimizeResult.StepResults))
	}

	// The single step should be at background_percent (100%)
	if uc02.OptimizeResult.StepResults[0].Step.PercentOfTarget != 100 {
		t.Errorf("background step should be at 100%%, got %.0f%%", uc02.OptimizeResult.StepResults[0].Step.PercentOfTarget)
	}
}

// --- Spike Participation ---

func TestSpikeParticipation(t *testing.T) {
	results, steps := runFullPipeline(t, filepath.Join(testdataDir(), "spike_config.yaml"))

	// UC01 has spike_participate=true (global default), UC03 has spike_participate=false
	// Both are calculated across all steps by the optimizer, but UC03's spike_participate=false
	// means at runtime it stays at base. The calculation still covers all steps.

	uc01 := findScenario(results, "Main page")
	uc03 := findScenario(results, "Non-spike endpoint")

	// Both should have results for all 7 steps (base + 3 spikes + 3 cooldowns)
	if len(uc01.OptimizeResult.StepResults) != len(steps) {
		t.Errorf("UC01: expected %d step results, got %d", len(steps), len(uc01.OptimizeResult.StepResults))
	}
	if len(uc03.OptimizeResult.StepResults) != len(steps) {
		t.Errorf("UC03: expected %d step results, got %d", len(steps), len(uc03.OptimizeResult.StepResults))
	}
}

// --- Edge Cases ---

func TestEdgeCase_SingleScenario(t *testing.T) {
	results, _ := runFullPipeline(t, filepath.Join(testdataDir(), "stability_config.yaml"))

	if len(results.ScenarioResults) != 1 {
		t.Errorf("expected 1 scenario, got %d", len(results.ScenarioResults))
	}
	sr := results.ScenarioResults[0]
	if sr.OptimizeResult.StepResults[0].Threads < 1 {
		t.Error("threads must be >= 1")
	}
}

func TestEdgeCase_VeryLowIntensity(t *testing.T) {
	// Create a plan with very low intensity inline
	plan := &config.TestPlan{
		Version: "1.0",
		GlobalDefaults: config.GlobalDefaults{
			Tool:               config.ToolLREPC,
			LoadModel:          config.LoadModelClosed,
			PacingMultiplier:   3.0,
			DeviationTolerance: 2.5,
			GeneratorsCount:    1,
			SpikeParticipate:   true,
		},
		Scenarios: []config.Scenario{
			{
				Name:            "Very low intensity",
				TargetIntensity: 1, // 1 ops/h
				IntensityUnit:   units.OpsPerHour,
				MaxScriptTimeMs: 50,
			},
		},
		Profile: config.TestProfile{
			Type:                config.ProfileStability,
			Percent:             100,
			DefaultRampupSec:    60,
			DefaultImpactSec:    60,
			DefaultStabilitySec: 300,
			DefaultRampdownSec:  60,
		},
	}

	plan = config.ResolveDefaults(plan)

	builder, _ := profile.NewProfileBuilder(plan.Profile.Type)
	steps, _ := builder.BuildSteps(plan.Profile)

	for _, scenario := range plan.Scenarios {
		sr, err := calculateScenario(scenario, steps, plan.GlobalDefaults)
		if err != nil {
			t.Fatalf("calculate: %v", err)
		}
		if len(sr.OptimizeResult.StepResults) < 1 {
			t.Fatal("expected at least 1 step result")
		}
		if sr.OptimizeResult.StepResults[0].Threads < 1 {
			t.Error("threads must be >= 1 even for very low intensity")
		}
	}
}

func TestEdgeCase_VeryHighIntensity(t *testing.T) {
	plan := &config.TestPlan{
		Version: "1.0",
		GlobalDefaults: config.GlobalDefaults{
			Tool:               config.ToolJMeter,
			LoadModel:          config.LoadModelClosed,
			PacingMultiplier:   3.0,
			DeviationTolerance: 5.0,
			GeneratorsCount:    10,
			SpikeParticipate:   true,
		},
		Scenarios: []config.Scenario{
			{
				Name:            "Very high intensity",
				TargetIntensity: 10000000, // 10M ops/h
				IntensityUnit:   units.OpsPerHour,
				MaxScriptTimeMs: 100,
			},
		},
		Profile: config.TestProfile{
			Type:                config.ProfileStability,
			Percent:             100,
			DefaultRampupSec:    60,
			DefaultImpactSec:    60,
			DefaultStabilitySec: 300,
			DefaultRampdownSec:  60,
		},
	}

	plan = config.ResolveDefaults(plan)

	builder, _ := profile.NewProfileBuilder(plan.Profile.Type)
	steps, _ := builder.BuildSteps(plan.Profile)

	for _, scenario := range plan.Scenarios {
		sr, err := calculateScenario(scenario, steps, plan.GlobalDefaults)
		if err != nil {
			t.Fatalf("calculate: %v", err)
		}
		if len(sr.OptimizeResult.StepResults) < 1 {
			t.Fatal("expected at least 1 step result")
		}
		if sr.OptimizeResult.StepResults[0].Threads < 1 {
			t.Error("threads must be >= 1")
		}
		// High intensity should produce many threads (per generator)
		if sr.OptimizeResult.StepResults[0].Threads < 10 {
			t.Errorf("expected many threads for high intensity, got %d", sr.OptimizeResult.StepResults[0].Threads)
		}
	}
}

// --- XLSX Output E2E ---

func TestXLSXOutput_FullPipeline(t *testing.T) {
	results, _ := runFullPipeline(t, filepath.Join(testdataDir(), "capacity_config.yaml"))

	tmpFile := filepath.Join(t.TempDir(), "output.xlsx")

	w := &output.XLSXWriter{}
	if err := w.Write(results, tmpFile); err != nil {
		t.Fatalf("XLSX write: %v", err)
	}

	// Read back and verify
	f, err := excelize.OpenFile(tmpFile)
	if err != nil {
		t.Fatalf("open XLSX: %v", err)
	}
	defer f.Close()

	// Verify sheets exist
	sheets := f.GetSheetList()
	expectedSheets := []string{"Summary", "Steps Detail", "Timeline", "Input Parameters", "JMeter Config"}
	for _, exp := range expectedSheets {
		found := false
		for _, s := range sheets {
			if s == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected sheet %q not found in %v", exp, sheets)
		}
	}

	// Verify Summary sheet has data
	val, err := f.GetCellValue("Summary", "A2")
	if err != nil {
		t.Fatalf("get cell: %v", err)
	}
	if val != "Main page" {
		t.Errorf("Summary A2: expected Main page, got %q", val)
	}

	// Verify Steps Detail has rows
	rows, err := f.GetRows("Steps Detail")
	if err != nil {
		t.Fatalf("get rows: %v", err)
	}
	// Header + data rows; at least header + some step rows
	if len(rows) < 2 {
		t.Errorf("Steps Detail: expected rows > 1, got %d", len(rows))
	}

	// Verify Timeline has phases
	timelineRows, err := f.GetRows("Timeline")
	if err != nil {
		t.Fatalf("get timeline rows: %v", err)
	}
	if len(timelineRows) < 2 {
		t.Errorf("Timeline: expected rows > 1, got %d", len(timelineRows))
	}
}

// --- Performance Test ---

func TestPerformance_ManyScenarios(t *testing.T) {
	// Build config with 50 scenarios and 20 steps
	scenarios := make([]config.Scenario, 50)
	for i := 0; i < 50; i++ {
		pm := 3.0
		scenarios[i] = config.Scenario{
			Name:             fmt.Sprintf("Scenario %d", i+1),
			TargetIntensity:  float64(10000 + i*1000),
			IntensityUnit:    units.OpsPerHour,
			MaxScriptTimeMs:  200 + i*50,
			PacingMultiplier: &pm,
		}
	}

	profileSteps := make([]config.ProfileStep, 20)
	for i := 0; i < 20; i++ {
		profileSteps[i] = config.ProfileStep{
			PercentOfTarget: float64(50 + i*25),
		}
	}

	plan := &config.TestPlan{
		Version: "1.0",
		GlobalDefaults: config.GlobalDefaults{
			Tool:               config.ToolJMeter,
			LoadModel:          config.LoadModelClosed,
			PacingMultiplier:   3.0,
			DeviationTolerance: 5.0,
			GeneratorsCount:    3,
			SpikeParticipate:   true,
		},
		Scenarios: scenarios,
		Profile: config.TestProfile{
			Type:                config.ProfileCustom,
			Steps:               profileSteps,
			DefaultRampupSec:    30,
			DefaultImpactSec:    60,
			DefaultStabilitySec: 120,
			DefaultRampdownSec:  30,
		},
	}

	plan = config.ResolveDefaults(plan)

	start := time.Now()

	builder, err := profile.NewProfileBuilder(plan.Profile.Type)
	if err != nil {
		t.Fatalf("NewProfileBuilder: %v", err)
	}
	steps, err := builder.BuildSteps(plan.Profile)
	if err != nil {
		t.Fatalf("BuildSteps: %v", err)
	}

	for _, scenario := range plan.Scenarios {
		_, err := calculateScenario(scenario, steps, plan.GlobalDefaults)
		if err != nil {
			t.Fatalf("calculateScenario(%s): %v", scenario.Name, err)
		}
	}

	elapsed := time.Since(start)
	t.Logf("50 scenarios x 20 steps completed in %v", elapsed)

	if elapsed > time.Second {
		t.Errorf("performance: expected < 1s, took %v", elapsed)
	}
}

// --- Timeline Verification ---

func TestTimeline_Continuity(t *testing.T) {
	results, _ := runFullPipeline(t, filepath.Join(testdataDir(), "capacity_config.yaml"))

	if len(results.Timeline) == 0 {
		t.Fatal("timeline is empty")
	}

	// First phase starts at 0
	if results.Timeline[0].StartTimeSec != 0 {
		t.Errorf("first phase should start at 0, starts at %d", results.Timeline[0].StartTimeSec)
	}

	// Each phase's end = next phase's start
	for i := 0; i < len(results.Timeline)-1; i++ {
		if results.Timeline[i].EndTimeSec != results.Timeline[i+1].StartTimeSec {
			t.Errorf("phase %d end (%d) != phase %d start (%d)",
				i, results.Timeline[i].EndTimeSec, i+1, results.Timeline[i+1].StartTimeSec)
		}
	}

	// EndTimeSec = StartTimeSec + DurationSec
	for i, phase := range results.Timeline {
		if phase.EndTimeSec != phase.StartTimeSec+phase.DurationSec {
			t.Errorf("phase %d: end (%d) != start (%d) + duration (%d)",
				i, phase.EndTimeSec, phase.StartTimeSec, phase.DurationSec)
		}
	}
}

// --- Deviation Tolerance ---

func TestDeviation_WithinTolerance(t *testing.T) {
	// Stable config with single step should have very low or zero deviation
	results, _ := runFullPipeline(t, filepath.Join(testdataDir(), "stability_config.yaml"))

	for _, sr := range results.ScenarioResults {
		for _, stepRes := range sr.OptimizeResult.StepResults {
			// For a single step, the corrected pacing ensures deviation is near-zero
			if stepRes.DeviationPct > 5.0 {
				t.Errorf("%s step %d: deviation %.2f%% exceeds 5%%",
					sr.Scenario.Name, stepRes.Step.StepNumber, stepRes.DeviationPct)
			}
		}
	}
}

// --- XLSX Output for LRE PC (no JMeter Config sheet) ---

func TestXLSXOutput_LREPC_NoJMeterSheet(t *testing.T) {
	results, _ := runFullPipeline(t, filepath.Join(testdataDir(), "lrepc_config.yaml"))

	tmpFile := filepath.Join(t.TempDir(), "lrepc_output.xlsx")

	w := &output.XLSXWriter{}
	if err := w.Write(results, tmpFile); err != nil {
		t.Fatalf("XLSX write: %v", err)
	}

	f, err := excelize.OpenFile(tmpFile)
	if err != nil {
		t.Fatalf("open XLSX: %v", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	for _, s := range sheets {
		if s == "JMeter Config" {
			t.Error("LRE PC config should not have JMeter Config sheet")
		}
	}
}

// --- Helpers ---

func findScenario(results engine.CalculationResults, name string) *engine.ScenarioResult {
	for i := range results.ScenarioResults {
		if results.ScenarioResults[i].Scenario.Name == name {
			return &results.ScenarioResults[i]
		}
	}
	return nil
}

// --- Validation of all configs ---

func TestAllConfigs_Validate(t *testing.T) {
	configs := []string{
		"capacity_config.yaml",
		"capacity_finetune_config.yaml",
		"custom_config.yaml",
		"spike_config.yaml",
		"stability_config.yaml",
		"lrepc_config.yaml",
		"valid_config.yaml",
	}

	for _, cfg := range configs {
		t.Run(cfg, func(t *testing.T) {
			path := filepath.Join(testdataDir(), cfg)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Skipf("config file %s not found", path)
			}
			plan, err := config.LoadFromYAML(path)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			plan = config.ResolveDefaults(plan)
			valErrs := config.Validate(plan)
			if config.HasErrors(valErrs) {
				for _, e := range valErrs {
					t.Logf("  %s", e.String())
				}
				t.Errorf("validation failed for %s", cfg)
			}
		})
	}
}

// --- Full Pipeline → JMX Generate ---

func TestFullPipeline_JMXGenerate(t *testing.T) {
	results, _ := runFullPipeline(t, filepath.Join(testdataDir(), "capacity_config.yaml"))

	data, err := integration.GenerateJMX(results)
	if err != nil {
		t.Fatalf("GenerateJMX: %v", err)
	}
	s := string(data)

	// Count ThreadGroup elements (use testclass which appears once per TG element)
	stgCount := strings.Count(s, `testclass="kg.apc.jmeter.threads.SteppingThreadGroup"`)
	utgCount := strings.Count(s, `testclass="kg.apc.jmeter.threads.UltimateThreadGroup"`)
	ffatgCount := strings.Count(s, `testclass="com.blazemeter.jmeter.threads.arrivals.FreeFormArrivalsThreadGroup"`)

	totalTG := stgCount + utgCount + ffatgCount
	if totalTG != 4 {
		t.Errorf("expected 4 ThreadGroups total, got %d (STG=%d, UTG=%d, FFATG=%d)", totalTG, stgCount, utgCount, ffatgCount)
	}

	// Open model scenario should be FFATG
	if ffatgCount < 1 {
		t.Error("expected at least 1 FreeFormArrivalsThreadGroup for open model scenario")
	}
	if !strings.Contains(s, "FFATG_API health check") {
		t.Error("expected FFATG_API health check")
	}

	// Closed model scenarios should have CTT
	cttCount := strings.Count(s, `testclass="ConstantThroughputTimer"`)
	if cttCount != 3 {
		t.Errorf("expected 3 ConstantThroughputTimers (closed model), got %d", cttCount)
	}

	// All scenarios should have ModuleController
	mcCount := strings.Count(s, `testclass="ModuleController"`)
	if mcCount != 4 {
		t.Errorf("expected 4 ModuleControllers, got %d", mcCount)
	}

	// Verify thread counts for closed model scenarios
	for _, sr := range results.ScenarioResults {
		if sr.IsOpenModel {
			continue
		}
		if len(sr.OptimizeResult.StepResults) > 0 {
			lastStep := sr.OptimizeResult.StepResults[len(sr.OptimizeResult.StepResults)-1]
			if lastStep.Threads < 1 {
				t.Errorf("scenario %s: last step threads should be >= 1", sr.Scenario.Name)
			}
		}
	}
}

// --- Full Pipeline → JMX Inject ---

func TestFullPipeline_JMXInject(t *testing.T) {
	results, _ := runFullPipeline(t, filepath.Join(testdataDir(), "capacity_config.yaml"))

	templatePath := filepath.Join("..", "integration", "testdata", "template.jmx")

	data, err := integration.InjectIntoJMX(templatePath, results)
	if err != nil {
		t.Fatalf("InjectIntoJMX: %v", err)
	}
	s := string(data)

	// Original content preserved
	if !strings.Contains(s, "Existing TG") {
		t.Error("original Existing TG should be preserved")
	}

	// New TGs added
	if !strings.Contains(s, "STG_Main page") && !strings.Contains(s, "UTG_Main page") {
		t.Error("expected Main page TG to be injected")
	}
	if !strings.Contains(s, "FFATG_API health check") {
		t.Error("expected FFATG_API health check to be injected")
	}

	if !strings.HasPrefix(s, "<?xml") {
		t.Error("should start with XML declaration")
	}
}

// --- Full Pipeline → LRE Push (mock server) ---

func TestFullPipeline_LREPush(t *testing.T) {
	results, _ := runFullPipeline(t, filepath.Join(testdataDir(), "lrepc_config.yaml"))

	var requestCount int32
	nextGroupID := 100

	mux := http.NewServeMux()

	// List groups - return empty (all new)
	mux.HandleFunc("/LoadTest/rest/domains/DEFAULT/projects/PROJ/tests/1/groups", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode([]integration.LREGroup{})
		case http.MethodPost:
			var g integration.LREGroup
			json.NewDecoder(r.Body).Decode(&g)
			g.ID = nextGroupID
			nextGroupID++
			json.NewEncoder(w).Encode(g)
		}
	})

	// Runtime settings
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		if strings.Contains(r.URL.Path, "runtime-settings") {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := integration.NewLREClient(server.URL+"/LoadTest/rest", "DEFAULT", "PROJ")

	pr, err := integration.PushToLRE(client, 1, "", "", results, false)
	if err != nil {
		t.Fatalf("PushToLRE: %v", err)
	}

	if len(pr.Actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(pr.Actions))
	}

	for _, action := range pr.Actions {
		if action.ActionType != "created" {
			t.Errorf("scenario %s: expected action 'created', got %q", action.ScenarioName, action.ActionType)
		}
		if action.VuserCount < 1 {
			t.Errorf("scenario %s: VuserCount should be >= 1, got %d", action.ScenarioName, action.VuserCount)
		}
		if action.PacingMS <= 0 {
			t.Errorf("scenario %s: PacingMS should be > 0, got %.2f", action.ScenarioName, action.PacingMS)
		}
		if action.ScriptID <= 0 {
			t.Errorf("scenario %s: ScriptID should be > 0, got %d", action.ScenarioName, action.ScriptID)
		}
	}

	// Verify VuserCount matches calculated threads
	for _, action := range pr.Actions {
		sr := findScenario(results, action.ScenarioName)
		if sr == nil {
			t.Errorf("scenario %s not found in results", action.ScenarioName)
			continue
		}
		lastStep := sr.OptimizeResult.StepResults[len(sr.OptimizeResult.StepResults)-1]
		if action.VuserCount != lastStep.Threads {
			t.Errorf("scenario %s: VuserCount %d != calculated threads %d", action.ScenarioName, action.VuserCount, lastStep.Threads)
		}
	}

	if atomic.LoadInt32(&requestCount) == 0 {
		t.Error("mock server should have received API requests")
	}
}

// --- Full Pipeline → LRE Push Dry Run ---

func TestFullPipeline_LREPushDryRun(t *testing.T) {
	results, _ := runFullPipeline(t, filepath.Join(testdataDir(), "lrepc_config.yaml"))

	var requestCount int32

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := integration.NewLREClient(server.URL+"/LoadTest/rest", "DEFAULT", "PROJ")

	pr, err := integration.PushToLRE(client, 1, "", "", results, true)
	if err != nil {
		t.Fatalf("PushToLRE dry-run: %v", err)
	}

	if len(pr.Actions) != 2 {
		t.Errorf("expected 2 actions in dry run, got %d", len(pr.Actions))
	}

	for _, action := range pr.Actions {
		if action.VuserCount < 1 {
			t.Errorf("scenario %s: VuserCount should be >= 1, got %d", action.ScenarioName, action.VuserCount)
		}
		if action.PacingMS <= 0 {
			t.Errorf("scenario %s: PacingMS should be > 0", action.ScenarioName)
		}
	}

	// No API calls should have been made
	if atomic.LoadInt32(&requestCount) != 0 {
		t.Errorf("dry run should make no API calls, but server received %d requests", atomic.LoadInt32(&requestCount))
	}
}
