package engine

import (
	"testing"
)

func TestCalculateRampUp(t *testing.T) {
	tests := []struct {
		name      string
		delta     int
		rampupSec int
		wantBatch int
		wantIntvl int
		wantAct   int
	}{
		{"exact 60/60", 60, 60, 1, 1, 60},
		{"10 vusers 60s", 10, 60, 1, 6, 60},
		{"single vuser", 1, 30, 1, 30, 30},
		{"100 in 10s", 100, 10, 10, 1, 10},
		{"55 in 60s", 55, 60, 0, 0, 0}, // just check actual is close to 60
		{"7 in 60s", 7, 60, 0, 0, 0},   // check actual closest to 60
		{"zero delta", 0, 60, 0, 0, 0},
		{"zero rampup", 5, 0, 5, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateRampUp(tt.delta, tt.rampupSec)

			switch tt.name {
			case "zero delta":
				if got.BatchSize != 0 || got.IntervalSec != 0 || got.Iterations != 0 {
					t.Errorf("delta=0: got %+v, want zero config", got)
				}
			case "zero rampup":
				if got.BatchSize != tt.delta || got.IntervalSec != 0 || got.ActualSec != 0 {
					t.Errorf("rampup=0: got %+v", got)
				}
			case "55 in 60s":
				dev := got.ActualSec - tt.rampupSec
				if dev < 0 {
					dev = -dev
				}
				if dev > 1 {
					t.Errorf("55/60: actual=%d, deviation=%d too large; config=%+v", got.ActualSec, dev, got)
				}
				if got.BatchSize < 1 || got.IntervalSec < 1 {
					t.Errorf("55/60: invalid config %+v", got)
				}
			case "7 in 60s":
				dev := got.ActualSec - tt.rampupSec
				if dev < 0 {
					dev = -dev
				}
				if dev > 4 {
					t.Errorf("7/60: actual=%d, deviation=%d too large; config=%+v", got.ActualSec, dev, got)
				}
			default:
				if tt.wantBatch != 0 && got.BatchSize != tt.wantBatch {
					t.Errorf("batch: got %d, want %d", got.BatchSize, tt.wantBatch)
				}
				if tt.wantIntvl != 0 && got.IntervalSec != tt.wantIntvl {
					t.Errorf("interval: got %d, want %d", got.IntervalSec, tt.wantIntvl)
				}
				if tt.wantAct != 0 && got.ActualSec != tt.wantAct {
					t.Errorf("actual: got %d, want %d", got.ActualSec, tt.wantAct)
				}
			}

			// General invariants
			if tt.delta > 0 && tt.rampupSec > 0 {
				if got.BatchSize < 1 {
					t.Errorf("batch must be >= 1, got %d", got.BatchSize)
				}
				if got.IntervalSec < 1 {
					t.Errorf("interval must be >= 1, got %d", got.IntervalSec)
				}
				// Verify total vusers
				totalVusers := (got.Iterations-1)*got.BatchSize + got.LastBatch
				if totalVusers != tt.delta {
					t.Errorf("total vusers: got %d, want %d (config=%+v)", totalVusers, tt.delta, got)
				}
			}
		})
	}
}
