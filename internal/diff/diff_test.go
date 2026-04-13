package diff

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"

	"loadcalc/internal/config"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func loadPlan(t *testing.T, name string) *config.TestPlan {
	t.Helper()
	plan, err := config.LoadFromYAML(testdataPath(name))
	if err != nil {
		t.Fatalf("failed to load %s: %v", name, err)
	}
	plan = config.ResolveDefaults(plan)
	return plan
}

func TestNoChanges(t *testing.T) {
	plan := loadPlan(t, "base.yaml")
	result := ComparePlans(plan, plan)

	if len(result.GlobalChanges) != 0 {
		t.Errorf("expected no global changes, got %d", len(result.GlobalChanges))
	}
	if len(result.ProfileChanges) != 0 {
		t.Errorf("expected no profile changes, got %d", len(result.ProfileChanges))
	}
	if len(result.ScenarioChanges) != 0 {
		t.Errorf("expected no scenario changes, got %d", len(result.ScenarioChanges))
	}
}

func TestGlobalFieldChanged(t *testing.T) {
	old := loadPlan(t, "base.yaml")
	modified := loadPlan(t, "modified.yaml")
	result := ComparePlans(old, modified)

	found := false
	for _, c := range result.GlobalChanges {
		if c.Field == "tool" {
			found = true
			if c.OldValue != "jmeter" || c.NewValue != "lre_pc" {
				t.Errorf("tool change: got %s -> %s", c.OldValue, c.NewValue)
			}
		}
	}
	if !found {
		t.Error("expected tool change in global changes")
	}

	// Also check pacing_multiplier changed
	found = false
	for _, c := range result.GlobalChanges {
		if c.Field == "pacing_multiplier" {
			found = true
		}
	}
	if !found {
		t.Error("expected pacing_multiplier change in global changes")
	}
}

func TestProfileTypeChanged(t *testing.T) {
	old := loadPlan(t, "base.yaml")
	modified := loadPlan(t, "modified.yaml")
	result := ComparePlans(old, modified)

	found := false
	for _, c := range result.ProfileChanges {
		if c.Field == "type" {
			found = true
			if c.OldValue != "max_search" || c.NewValue != "stable" {
				t.Errorf("type change: got %s -> %s", c.OldValue, c.NewValue)
			}
		}
	}
	if !found {
		t.Error("expected profile type change")
	}
}

func TestScenarioAdded(t *testing.T) {
	old := loadPlan(t, "base.yaml")
	modified := loadPlan(t, "modified.yaml")
	result := ComparePlans(old, modified)

	found := false
	for _, sc := range result.ScenarioChanges {
		if sc.Type == Added && sc.Name == "Browse" {
			found = true
			if sc.OldIndex != -1 {
				t.Errorf("expected OldIndex -1 for added scenario, got %d", sc.OldIndex)
			}
		}
	}
	if !found {
		t.Error("expected Browse scenario to be added")
	}
}

func TestScenarioRemoved(t *testing.T) {
	old := loadPlan(t, "base.yaml")
	modified := loadPlan(t, "modified.yaml")
	result := ComparePlans(old, modified)

	found := false
	for _, sc := range result.ScenarioChanges {
		if sc.Type == Removed && sc.Name == "Checkout" {
			found = true
			if sc.NewIndex != -1 {
				t.Errorf("expected NewIndex -1 for removed scenario, got %d", sc.NewIndex)
			}
		}
	}
	if !found {
		t.Error("expected Checkout scenario to be removed")
	}
}

func TestScenarioFieldModified(t *testing.T) {
	old := loadPlan(t, "base.yaml")
	modified := loadPlan(t, "modified.yaml")
	result := ComparePlans(old, modified)

	found := false
	for _, sc := range result.ScenarioChanges {
		if sc.Type != Modified || sc.Name != "Login" {
			continue
		}
		found = true
		if len(sc.Fields) == 0 {
			t.Error("expected field changes for Login scenario")
		}
		// Check target_intensity changed
		intensityFound := false
		scriptTimeFound := false
		for _, f := range sc.Fields {
			if f.Field == "target_intensity" {
				intensityFound = true
			}
			if f.Field == "max_script_time_ms" {
				scriptTimeFound = true
			}
		}
		if !intensityFound {
			t.Error("expected target_intensity change")
		}
		if !scriptTimeFound {
			t.Error("expected max_script_time_ms change")
		}
	}
	if !found {
		t.Error("expected Login scenario to be modified")
	}
}

func TestMultipleChanges(t *testing.T) {
	old := loadPlan(t, "base.yaml")
	modified := loadPlan(t, "modified.yaml")
	result := ComparePlans(old, modified)

	// Should have global, profile, and scenario changes
	if len(result.GlobalChanges) == 0 {
		t.Error("expected global changes")
	}
	if len(result.ProfileChanges) == 0 {
		t.Error("expected profile changes")
	}
	if len(result.ScenarioChanges) == 0 {
		t.Error("expected scenario changes")
	}

	// Count scenario change types
	added, removed, modifiedCount := 0, 0, 0
	for _, sc := range result.ScenarioChanges {
		switch sc.Type {
		case Added:
			added++
		case Removed:
			removed++
		case Modified:
			modifiedCount++
		}
	}
	if added != 1 {
		t.Errorf("expected 1 added scenario, got %d", added)
	}
	if removed != 1 {
		t.Errorf("expected 1 removed scenario, got %d", removed)
	}
	if modifiedCount < 1 {
		t.Errorf("expected at least 1 modified scenario, got %d", modifiedCount)
	}
}

func TestFormatJSON(t *testing.T) {
	old := loadPlan(t, "base.yaml")
	modified := loadPlan(t, "modified.yaml")
	result := ComparePlans(old, modified)

	data, err := FormatJSON(result)
	if err != nil {
		t.Fatalf("FormatJSON error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if _, ok := parsed["global_changes"]; !ok {
		t.Error("expected global_changes key in JSON")
	}
	if _, ok := parsed["profile_changes"]; !ok {
		t.Error("expected profile_changes key in JSON")
	}
	if _, ok := parsed["scenario_changes"]; !ok {
		t.Error("expected scenario_changes key in JSON")
	}
}

func TestFormatTable(t *testing.T) {
	old := loadPlan(t, "base.yaml")
	modified := loadPlan(t, "modified.yaml")
	result := ComparePlans(old, modified)

	out := FormatTable(result)
	if out == "" {
		t.Error("expected non-empty table output")
	}
}

func TestUnchangedScenarioNotInDiff(t *testing.T) {
	// When comparing two identical plans, no scenarios should appear in diff
	old := loadPlan(t, "base.yaml")
	old2 := loadPlan(t, "base.yaml")
	result := ComparePlans(old, old2)

	for _, sc := range result.ScenarioChanges {
		t.Errorf("unexpected scenario change: %s (%s)", sc.Name, sc.Type)
	}
}
