package units

import (
	"math"
	"testing"
)

func TestNormalizeToOpsPerSec_Basic(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		unit  IntensityUnit
		want  float64
	}{
		{"3600 ops/h -> 1 ops/s", 3600, OpsPerHour, 1.0},
		{"60 ops/m -> 1 ops/s", 60, OpsPerMinute, 1.0},
		{"1 ops/s -> 1 ops/s", 1, OpsPerSecond, 1.0},
		{"zero ops/h", 0, OpsPerHour, 0},
		{"zero ops/m", 0, OpsPerMinute, 0},
		{"zero ops/s", 0, OpsPerSecond, 0},
		{"very small 0.001 ops/h", 0.001, OpsPerHour, 0.001 / 3600},
		{"very large 1000000 ops/h", 1_000_000, OpsPerHour, 1_000_000.0 / 3600},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeToOpsPerSec(tt.value, tt.unit)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeToOpsPerSec_Negative(t *testing.T) {
	_, err := NormalizeToOpsPerSec(-1, OpsPerHour)
	if err == nil {
		t.Error("expected error for negative value")
	}
	_, err = NormalizeToOpsPerSec(-0.5, OpsPerSecond)
	if err == nil {
		t.Error("expected error for negative value")
	}
}

func TestNormalizeToOpsPerSec_UnknownUnit(t *testing.T) {
	_, err := NormalizeToOpsPerSec(100, IntensityUnit("ops_d"))
	if err == nil {
		t.Error("expected error for unknown unit")
	}
}

func TestConvertFromOpsPerSec(t *testing.T) {
	tests := []struct {
		name      string
		opsPerSec float64
		unit      IntensityUnit
		want      float64
	}{
		{"to ops/h", 1.0, OpsPerHour, 3600.0},
		{"to ops/m", 1.0, OpsPerMinute, 60.0},
		{"to ops/s", 1.0, OpsPerSecond, 1.0},
		{"zero", 0, OpsPerHour, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertFromOpsPerSec(tt.opsPerSec, tt.unit)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundtrip(t *testing.T) {
	units := []IntensityUnit{OpsPerHour, OpsPerMinute, OpsPerSecond}
	values := []float64{0, 0.001, 1, 60, 3600, 100000}
	for _, u := range units {
		for _, v := range values {
			normalized, err := NormalizeToOpsPerSec(v, u)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			back := ConvertFromOpsPerSec(normalized, u)
			if math.Abs(back-v) > 1e-6 {
				t.Errorf("roundtrip failed for %v %s: got %v", v, u, back)
			}
		}
	}
}
