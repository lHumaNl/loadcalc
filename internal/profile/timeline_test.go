package profile

import "testing"

func TestBuildTimeline_Basic(t *testing.T) {
	steps := []Step{
		{StepNumber: 1, PercentOfTarget: 50, RampupSec: 60, ImpactSec: 300, StabilitySec: 120, RampdownSec: 0},
		{StepNumber: 2, PercentOfTarget: 100, RampupSec: 60, ImpactSec: 300, StabilitySec: 120, RampdownSec: 30},
	}
	phases := BuildTimeline(steps)

	// Step 1: Rampup(0-60), Impact(60-360), Stability(360-480) — no rampdown
	// Step 2: Rampup(480-540), Impact(540-840), Stability(840-960), Rampdown(960-990)
	expected := []struct {
		name  string
		start int
		end   int
		pct   float64
	}{
		{"Step 1 Rampup", 0, 60, 50},
		{"Step 1 Impact", 60, 360, 50},
		{"Step 1 Stability", 360, 480, 50},
		{"Step 2 Rampup", 480, 540, 100},
		{"Step 2 Impact", 540, 840, 100},
		{"Step 2 Stability", 840, 960, 100},
		{"Step 2 Rampdown", 960, 990, 100},
	}

	if len(phases) != len(expected) {
		t.Fatalf("got %d phases, want %d", len(phases), len(expected))
	}
	for i, want := range expected {
		p := phases[i]
		if p.PhaseName != want.name {
			t.Errorf("phase %d: name=%q, want %q", i, p.PhaseName, want.name)
		}
		if p.StartTimeSec != want.start {
			t.Errorf("phase %d (%s): start=%d, want %d", i, want.name, p.StartTimeSec, want.start)
		}
		if p.EndTimeSec != want.end {
			t.Errorf("phase %d (%s): end=%d, want %d", i, want.name, p.EndTimeSec, want.end)
		}
		if p.PercentOfTarget != want.pct {
			t.Errorf("phase %d: pct=%v, want %v", i, p.PercentOfTarget, want.pct)
		}
		if p.DurationSec != want.end-want.start {
			t.Errorf("phase %d: duration=%d, want %d", i, p.DurationSec, want.end-want.start)
		}
	}
}

func TestBuildTimeline_SkipZeroDurations(t *testing.T) {
	steps := []Step{
		{StepNumber: 1, PercentOfTarget: 100, RampupSec: 0, ImpactSec: 300, StabilitySec: 0, RampdownSec: 0},
	}
	phases := BuildTimeline(steps)
	if len(phases) != 1 {
		t.Fatalf("got %d phases, want 1 (only Impact)", len(phases))
	}
	if phases[0].PhaseName != "Step 1 Impact" {
		t.Errorf("phase name=%q, want 'Step 1 Impact'", phases[0].PhaseName)
	}
}

func TestBuildTimeline_Empty(t *testing.T) {
	phases := BuildTimeline(nil)
	if len(phases) != 0 {
		t.Errorf("got %d phases for nil input", len(phases))
	}
}
