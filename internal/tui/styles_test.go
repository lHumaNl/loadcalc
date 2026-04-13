package tui_test

import (
	"strings"
	"testing"

	"loadcalc/internal/tui/styles"
)

func TestClassifyDeviation(t *testing.T) {
	tests := []struct {
		name      string
		deviation float64
		tolerance float64
		want      styles.DeviationLevel
	}{
		{"zero deviation", 0.0, 2.5, styles.DeviationOK},
		{"small deviation", 0.5, 2.5, styles.DeviationOK},
		{"at green boundary", 0.99, 2.5, styles.DeviationOK},
		{"just above green", 1.0, 2.5, styles.DeviationWarn},
		{"mid warning", 1.5, 2.5, styles.DeviationWarn},
		{"at yellow boundary", 2.5, 2.5, styles.DeviationWarn},
		{"above tolerance", 3.0, 2.5, styles.DeviationExceeded},
		{"negative deviation OK", -0.5, 2.5, styles.DeviationOK},
		{"negative deviation warn", -1.5, 2.5, styles.DeviationWarn},
		{"negative deviation exceeded", -3.0, 2.5, styles.DeviationExceeded},
		{"custom tolerance exceeded", 2.0, 1.5, styles.DeviationExceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := styles.ClassifyDeviation(tt.deviation, tt.tolerance)
			if got != tt.want {
				t.Errorf("ClassifyDeviation(%v, %v) = %v, want %v", tt.deviation, tt.tolerance, got, tt.want)
			}
		})
	}
}

func TestDeviationSymbol(t *testing.T) {
	if styles.DeviationSymbol(styles.DeviationOK) != styles.SymbolOK {
		t.Errorf("expected %s for OK", styles.SymbolOK)
	}
	if styles.DeviationSymbol(styles.DeviationWarn) != styles.SymbolWarning {
		t.Errorf("expected %s for Warn", styles.SymbolWarning)
	}
	if styles.DeviationSymbol(styles.DeviationExceeded) != styles.SymbolExceeded {
		t.Errorf("expected %s for Exceeded", styles.SymbolExceeded)
	}
}

func TestFormatDeviation(t *testing.T) {
	result := styles.FormatDeviation(0.5, 2.5)
	clean := stripANSI(result)
	if !strings.Contains(clean, styles.SymbolOK) {
		t.Errorf("expected OK symbol in %q", clean)
	}
	if !strings.Contains(clean, "0.50%") {
		t.Errorf("expected 0.50%% in %q", clean)
	}

	result = styles.FormatDeviation(1.5, 2.5)
	clean = stripANSI(result)
	if !strings.Contains(clean, styles.SymbolWarning) {
		t.Errorf("expected warning symbol in %q", clean)
	}

	result = styles.FormatDeviation(3.0, 2.5)
	clean = stripANSI(result)
	if !strings.Contains(clean, styles.SymbolExceeded) {
		t.Errorf("expected exceeded symbol in %q", clean)
	}
}

func stripANSI(s string) string {
	var result []byte
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j
		} else {
			result = append(result, s[i])
			i++
		}
	}
	return string(result)
}
