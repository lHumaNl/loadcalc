package profile

import (
	"testing"

	"loadcalc/internal/config"
)

func TestNewProfileBuilder_Stable(t *testing.T) {
	b, err := NewProfileBuilder(config.ProfileStable)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := b.(*StableProfileBuilder); !ok {
		t.Fatalf("expected *StableProfileBuilder, got %T", b)
	}
}

func TestNewProfileBuilder_MaxSearch(t *testing.T) {
	b, err := NewProfileBuilder(config.ProfileMaxSearch)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := b.(*MaxSearchProfileBuilder); !ok {
		t.Fatalf("expected *MaxSearchProfileBuilder, got %T", b)
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
