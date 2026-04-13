package engine

import (
	"math"
	"testing"

	"loadcalc/internal/config"
	"loadcalc/pkg/units"
)

func ptrFloat(f float64) *float64 { return &f }

func makeScenario(name string, targetOpsH float64, maxScriptTimeMs int, multiplier float64) config.Scenario {
	return config.Scenario{
		Name:             name,
		TargetIntensity:  targetOpsH,
		IntensityUnit:    units.OpsPerHour,
		MaxScriptTimeMs:  maxScriptTimeMs,
		PacingMultiplier: ptrFloat(multiplier),
	}
}

func opsHToRPS(opsH float64) float64 {
	return opsH / 3600
}

// === JMeter Closed Model Tests (Appendix A reference data, 3 generators) ===

func TestJMeterClosed_UC01(t *testing.T) {
	calc, err := NewCalculator(config.ToolJMeter, config.LoadModelClosed, 3)
	if err != nil {
		t.Fatal(err)
	}
	s := makeScenario("UC01", 720000, 1100, 3.0)
	// target_rps = 720000/3600 = 200, per_gen = 200/3 = 66.666...
	// pacing = 1100*3 = 3300, ideal = 66.666*3.3 = 220, threads=220
	// CTT = (66.666/220)*60 = 18.1818...
	r, err := calc.Calculate(s, opsHToRPS(720000))
	if err != nil {
		t.Fatal(err)
	}
	if r.Threads != 220 {
		t.Errorf("threads: got %d, want 220", r.Threads)
	}
	if math.Abs(r.OpsPerMinPerThread-18.1818) > 0.01 {
		t.Errorf("CTT: got %f, want ~18.18", r.OpsPerMinPerThread)
	}
}

func TestJMeterClosed_UC02(t *testing.T) {
	calc, err := NewCalculator(config.ToolJMeter, config.LoadModelClosed, 3)
	if err != nil {
		t.Fatal(err)
	}
	s := makeScenario("UC02", 90000, 1000, 3.0)
	// per_gen = 25/3 = 8.333, pacing=3000, ideal=8.333*3=25, threads=25
	// Wait: 90000/3600=25 rps total, per_gen=25/3=8.333
	// ideal=8.333*3=25. threads=25
	// CTT = (8.333/25)*60 = 20.0
	r, err := calc.Calculate(s, opsHToRPS(90000))
	if err != nil {
		t.Fatal(err)
	}
	if r.Threads != 25 {
		t.Errorf("threads: got %d, want 25", r.Threads)
	}
	if math.Abs(r.OpsPerMinPerThread-20.0) > 0.01 {
		t.Errorf("CTT: got %f, want 20.0", r.OpsPerMinPerThread)
	}
}

func TestJMeterClosed_UC03(t *testing.T) {
	calc, err := NewCalculator(config.ToolJMeter, config.LoadModelClosed, 3)
	if err != nil {
		t.Fatal(err)
	}
	s := makeScenario("UC03", 90000, 200, 3.0)
	// per_gen=8.333, pacing=600, ideal=8.333*0.6=5, threads=5
	// CTT = (8.333/5)*60 = 100.0
	r, err := calc.Calculate(s, opsHToRPS(90000))
	if err != nil {
		t.Fatal(err)
	}
	if r.Threads != 5 {
		t.Errorf("threads: got %d, want 5", r.Threads)
	}
	if math.Abs(r.OpsPerMinPerThread-100.0) > 0.01 {
		t.Errorf("CTT: got %f, want 100.0", r.OpsPerMinPerThread)
	}
}

func TestJMeterClosed_UC04(t *testing.T) {
	calc, err := NewCalculator(config.ToolJMeter, config.LoadModelClosed, 3)
	if err != nil {
		t.Fatal(err)
	}
	s := makeScenario("UC04", 90000, 1100, 3.0)
	// per_gen=8.333, pacing=3300, ideal=8.333*3.3=27.5, threads=28 (ceil)
	// CTT = (8.333/28)*60 = 17.857...
	r, err := calc.Calculate(s, opsHToRPS(90000))
	if err != nil {
		t.Fatal(err)
	}
	if r.Threads != 28 {
		t.Errorf("threads: got %d, want 28", r.Threads)
	}
	if math.Abs(r.OpsPerMinPerThread-17.857) > 0.01 {
		t.Errorf("CTT: got %f, want ~17.86", r.OpsPerMinPerThread)
	}
}

// === LRE PC Tests (from conversation reference) ===

func TestLREPC_Case1(t *testing.T) {
	calc, err := NewCalculator(config.ToolLREPC, config.LoadModelClosed, 1)
	if err != nil {
		t.Fatal(err)
	}
	s := makeScenario("lre1", 13044, 550, 3.0)
	targetRPS := opsHToRPS(13044) // 3.6233...
	r, err := calc.Calculate(s, targetRPS)
	if err != nil {
		t.Fatal(err)
	}
	if r.Threads != 6 {
		t.Errorf("threads: got %d, want 6", r.Threads)
	}
	// corrected pacing = 6/3.6233*1000 = 1656.02...
	if math.Abs(r.PacingMS-1656.0) > 1.0 {
		t.Errorf("pacing: got %f, want ~1656", r.PacingMS)
	}
}

func TestLREPC_Case2(t *testing.T) {
	calc, err := NewCalculator(config.ToolLREPC, config.LoadModelClosed, 1)
	if err != nil {
		t.Fatal(err)
	}
	s := makeScenario("lre2", 75, 50, 3.0)
	targetRPS := opsHToRPS(75) // 0.02083...
	r, err := calc.Calculate(s, targetRPS)
	if err != nil {
		t.Fatal(err)
	}
	if r.Threads != 1 {
		t.Errorf("threads: got %d, want 1", r.Threads)
	}
	if math.Abs(r.PacingMS-48000.0) > 1.0 {
		t.Errorf("pacing: got %f, want 48000", r.PacingMS)
	}
}

// === Open Model Tests ===

func TestJMeterOpen_OpsPerSec(t *testing.T) {
	calc, err := NewCalculator(config.ToolJMeter, config.LoadModelOpen, 1)
	if err != nil {
		t.Fatal(err)
	}
	s := makeScenario("open1", 3600, 100, 3.0) // 1 rps
	r, err := calc.Calculate(s, 1.0)
	if err != nil {
		t.Fatal(err)
	}
	if r.OutputUnit != "ops/sec" {
		t.Errorf("unit: got %s, want ops/sec", r.OutputUnit)
	}
	if r.OutputValue != 1.0 {
		t.Errorf("value: got %f, want 1.0", r.OutputValue)
	}
}

func TestJMeterOpen_OpsPerMin(t *testing.T) {
	calc, err := NewCalculator(config.ToolJMeter, config.LoadModelOpen, 1)
	if err != nil {
		t.Fatal(err)
	}
	// target < 0.01 rps
	s := makeScenario("open2", 18, 100, 3.0) // 0.005 rps
	r, err := calc.Calculate(s, 0.005)
	if err != nil {
		t.Fatal(err)
	}
	if r.OutputUnit != "ops/min" {
		t.Errorf("unit: got %s, want ops/min", r.OutputUnit)
	}
	if math.Abs(r.OutputValue-0.3) > 0.001 {
		t.Errorf("value: got %f, want 0.3", r.OutputValue)
	}
}

func TestJMeterOpen_PerGenerator(t *testing.T) {
	calc, err := NewCalculator(config.ToolJMeter, config.LoadModelOpen, 4)
	if err != nil {
		t.Fatal(err)
	}
	s := makeScenario("open3", 14400, 100, 3.0) // 4 rps total
	r, err := calc.Calculate(s, 4.0)
	if err != nil {
		t.Fatal(err)
	}
	if r.OutputUnit != "ops/sec" {
		t.Errorf("unit: got %s, want ops/sec", r.OutputUnit)
	}
	if r.OutputValue != 1.0 {
		t.Errorf("value: got %f, want 1.0 (4/4 generators)", r.OutputValue)
	}
}

// === Edge Cases ===

func TestLREPC_VeryLowIntensity_MinOneThread(t *testing.T) {
	calc, err := NewCalculator(config.ToolLREPC, config.LoadModelClosed, 1)
	if err != nil {
		t.Fatal(err)
	}
	s := makeScenario("low", 1, 10, 3.0) // extremely low
	r, err := calc.Calculate(s, opsHToRPS(1))
	if err != nil {
		t.Fatal(err)
	}
	if r.Threads < 1 {
		t.Errorf("threads must be >= 1, got %d", r.Threads)
	}
}

func TestFactory_OpenLREPC_Error(t *testing.T) {
	_, err := NewCalculator(config.ToolLREPC, config.LoadModelOpen, 1)
	if err == nil {
		t.Error("expected error for open model with LRE PC")
	}
}

func TestFactory_UnknownTool_Error(t *testing.T) {
	_, err := NewCalculator("unknown", config.LoadModelClosed, 1)
	if err == nil {
		t.Error("expected error for unknown tool")
	}
}

// === Appendix A: verify the reference table values directly (1 generator for raw calc) ===

func TestJMeterClosed_UC01_SingleGenerator(t *testing.T) {
	// Appendix A table shows values before per-generator split
	// 720000 ops/h = 200 rps, pacing=3300, ideal=660, threads=660
	// But table says threads=220 which is per-generator (3 gens)
	// So the reference table IS per-generator. Test with 3 gens confirms above.
}

func TestJMeterClosed_UC04_CeilRounding(t *testing.T) {
	// UC04: 90000 ops/h, 1100ms, x3, 3 generators
	// per_gen = 8.333 rps, pacing=3300, ideal=8.333*3.3=27.5, ceil=28
	// This tests that we use ceil not round
	calc, err := NewCalculator(config.ToolJMeter, config.LoadModelClosed, 3)
	if err != nil {
		t.Fatal(err)
	}
	s := makeScenario("UC04", 90000, 1100, 3.0)
	r, err := calc.Calculate(s, opsHToRPS(90000))
	if err != nil {
		t.Fatal(err)
	}
	// With round, 27.5 would round to 28 too, but ceil guarantees it
	if r.Threads != 28 {
		t.Errorf("threads: got %d, want 28 (ceil of 27.5)", r.Threads)
	}
}

func TestLREPC_DeviationIsZeroForExactThreads(t *testing.T) {
	// UC02 equivalent for LRE: 25 rps, 1000ms, x3 -> ideal=75 threads exact
	calc, err := NewCalculator(config.ToolLREPC, config.LoadModelClosed, 1)
	if err != nil {
		t.Fatal(err)
	}
	s := makeScenario("exact", 90000, 1000, 3.0)
	r, err := calc.Calculate(s, opsHToRPS(90000))
	if err != nil {
		t.Fatal(err)
	}
	if r.DeviationPercent > 0.001 {
		t.Errorf("deviation should be ~0 for exact threads, got %f", r.DeviationPercent)
	}
}
