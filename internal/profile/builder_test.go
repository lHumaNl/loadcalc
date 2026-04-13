package profile

import (
	"testing"

	"loadcalc/internal/config"
)

func TestNewProfileBuilder_Stability(t *testing.T) {
	b, err := NewProfileBuilder(config.ProfileStability)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := b.(*StabilityProfileBuilder); !ok {
		t.Fatalf("expected *StabilityProfileBuilder, got %T", b)
	}
}

func TestNewProfileBuilder_Capacity(t *testing.T) {
	b, err := NewProfileBuilder(config.ProfileCapacity)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := b.(*CapacityProfileBuilder); !ok {
		t.Fatalf("expected *CapacityProfileBuilder, got %T", b)
	}
}

func TestNewProfileBuilder_Custom(t *testing.T) {
	b, err := NewProfileBuilder(config.ProfileCustom)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := b.(*CustomProfileBuilder); !ok {
		t.Fatalf("expected *CustomProfileBuilder, got %T", b)
	}
}

func TestNewProfileBuilder_Spike(t *testing.T) {
	b, err := NewProfileBuilder(config.ProfileSpike)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := b.(*SpikeProfileBuilder); !ok {
		t.Fatalf("expected *SpikeProfileBuilder, got %T", b)
	}
}

func TestNewProfileBuilder_Unknown(t *testing.T) {
	_, err := NewProfileBuilder("bogus")
	if err == nil {
		t.Fatal("expected error for unknown profile type")
	}
}
