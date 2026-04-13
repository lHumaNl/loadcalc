package profile

import (
	"testing"

	"loadcalc/internal/config"
)

func TestSpike_ThreeSpikes(t *testing.T) {
	b := &SpikeProfileBuilder{}
	steps, err := b.BuildSteps(config.TestProfile{
		BasePercent:          100,
		SpikeStartIncrement:  50,
		SpikeIncrementGrowth: 25,
		NumSpikes:            3,
		SpiketimeSec:         60,
		CooldownSec:          120,
		DefaultRampupSec:     30,
		DefaultImpactSec:     300,
		DefaultStabilitySec:  60,
		DefaultRampdownSec:   15,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Expected: base step + 3*(spike + cooldown) = 1 + 6 = 7 steps
	if len(steps) != 7 {
		t.Fatalf("got %d steps, want 7", len(steps))
	}

	// Base step
	if steps[0].PercentOfTarget != 100 {
		t.Errorf("base step percent=%v, want 100", steps[0].PercentOfTarget)
	}
	if steps[0].RampdownSec != 0 {
		t.Errorf("base step should have no rampdown")
	}

	// Spike amplitudes: 100+50+0*25=150, 100+50+1*25=175, 100+50+2*25=200
	expectedSpikes := []float64{150, 175, 200}
	for i, want := range expectedSpikes {
		spikeStep := steps[1+i*2]
		if spikeStep.PercentOfTarget != want {
			t.Errorf("spike %d: percent=%v, want %v", i+1, spikeStep.PercentOfTarget, want)
		}
		if spikeStep.ImpactSec != 60 {
			t.Errorf("spike %d: impact=%d, want 60 (spiketime)", i+1, spikeStep.ImpactSec)
		}

		// Cooldown step
		cooldown := steps[2+i*2]
		if cooldown.PercentOfTarget != 100 {
			t.Errorf("cooldown %d: percent=%v, want 100", i+1, cooldown.PercentOfTarget)
		}
		if cooldown.ImpactSec != 120 {
			t.Errorf("cooldown %d: impact=%d, want 120", i+1, cooldown.ImpactSec)
		}
	}
}

func TestSpike_ZeroSpikes_Error(t *testing.T) {
	b := &SpikeProfileBuilder{}
	_, err := b.BuildSteps(config.TestProfile{NumSpikes: 0})
	if err == nil {
		t.Fatal("expected error for 0 spikes")
	}
}
