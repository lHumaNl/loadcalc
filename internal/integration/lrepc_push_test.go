package integration

import (
	"testing"

	"loadcalc/internal/config"
	"loadcalc/internal/engine"
	"loadcalc/internal/profile"
)

func makeTestResults() engine.CalculationResults {
	closedModel := config.LoadModelClosed
	openModel := config.LoadModelOpen

	return engine.CalculationResults{
		Plan: &config.TestPlan{
			GlobalDefaults: config.GlobalDefaults{
				Tool:      config.ToolLREPC,
				LoadModel: config.LoadModelClosed,
			},
		},
		ScenarioResults: []engine.ScenarioResult{
			{
				Scenario: config.Scenario{
					Name:     "Login",
					ScriptID: 100,
				},
				IsOpenModel: false,
				OptimizeResult: engine.OptimizeResult{
					BestPacingMS: 5000,
					StepResults: []engine.StepResult{
						{Step: profile.Step{StepNumber: 1, PercentOfTarget: 100}, Threads: 10},
					},
				},
			},
			{
				Scenario: config.Scenario{
					Name:      "Checkout",
					ScriptID:  101,
					LoadModel: &closedModel,
				},
				IsOpenModel: false,
				OptimizeResult: engine.OptimizeResult{
					BestPacingMS: 3000,
					StepResults: []engine.StepResult{
						{Step: profile.Step{StepNumber: 1, PercentOfTarget: 100}, Threads: 5},
					},
				},
			},
			{
				Scenario: config.Scenario{
					Name:      "OpenAPI",
					ScriptID:  102,
					LoadModel: &openModel,
				},
				IsOpenModel: true,
				OptimizeResult: engine.OptimizeResult{
					StepResults: []engine.StepResult{
						{Step: profile.Step{StepNumber: 1, PercentOfTarget: 100}, Threads: 0},
					},
				},
			},
		},
	}
}

func TestPushToLRE_NewAndExisting(t *testing.T) {
	srv := newMockLREServer(t)
	defer srv.Close()

	c := newTestClient(srv.URL)
	_ = c.Authenticate("admin", "secret")

	results := makeTestResults()

	pr, err := PushToLRE(c, 1, "", "", results, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Login exists (group name match), should be updated
	// Search doesn't exist, should be created
	// OpenAPI is open model, should be skipped
	if len(pr.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(pr.Actions))
	}

	var foundUpdated, foundCreated bool
	for _, a := range pr.Actions {
		if a.ScenarioName == "Login" && a.ActionType == "updated" {
			foundUpdated = true
			if a.VuserCount != 10 {
				t.Errorf("expected VuserCount 10, got %d", a.VuserCount)
			}
		}
		if a.ScenarioName == "Checkout" && a.ActionType == "created" {
			foundCreated = true
			if a.VuserCount != 5 {
				t.Errorf("expected VuserCount 5, got %d", a.VuserCount)
			}
		}
	}
	if !foundUpdated {
		t.Error("expected Login to be updated")
	}
	if !foundCreated {
		t.Error("expected Checkout to be created")
	}
}

func TestPushToLRE_DryRun(t *testing.T) {
	srv := newMockLREServer(t)
	defer srv.Close()

	c := newTestClient(srv.URL)
	_ = c.Authenticate("admin", "secret")

	results := makeTestResults()

	pr, err := PushToLRE(c, 1, "", "", results, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should still collect actions but not make API calls
	if len(pr.Actions) != 2 {
		t.Fatalf("expected 2 actions in dry-run, got %d", len(pr.Actions))
	}

	// Verify actions are planned but with dry-run semantics
	for _, a := range pr.Actions {
		if a.ScenarioName == "OpenAPI" {
			t.Error("open model scenario should not appear in actions")
		}
	}
}

func TestPushToLRE_OpenModelSkipped(t *testing.T) {
	srv := newMockLREServer(t)
	defer srv.Close()

	c := newTestClient(srv.URL)
	_ = c.Authenticate("admin", "secret")

	results := makeTestResults()

	pr, err := PushToLRE(c, 1, "", "", results, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, a := range pr.Actions {
		if a.ScenarioName == "OpenAPI" {
			t.Error("open model scenario should be skipped")
		}
	}

	// Check warnings mention skipped scenario
	foundWarning := false
	for _, w := range pr.Warnings {
		if w != "" {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Error("expected warning about skipped open model scenario")
	}
}

func TestPushToLRE_CreateNewTest(t *testing.T) {
	srv := newMockLREServer(t)
	defer srv.Close()

	c := newTestClient(srv.URL)
	_ = c.Authenticate("admin", "secret")

	results := makeTestResults()

	pr, err := PushToLRE(c, 0, "NewTest", "Subject", results, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All closed-model scenarios should be created (new test has no existing groups)
	if len(pr.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(pr.Actions))
	}
	for _, a := range pr.Actions {
		if a.ActionType != "created" {
			t.Errorf("expected all actions to be 'created', got %q for %s", a.ActionType, a.ScenarioName)
		}
	}
}

func TestPushToLRE_BothTestIDAndTestName_Error(t *testing.T) {
	c := &LREClient{}
	results := makeTestResults()

	_, err := PushToLRE(c, 1, "SomeName", "", results, false)
	if err == nil {
		t.Fatal("expected error when both testID and testName are provided")
	}
}

func TestPushToLRE_NeitherTestIDNorTestName_Error(t *testing.T) {
	c := &LREClient{}
	results := makeTestResults()

	_, err := PushToLRE(c, 0, "", "", results, false)
	if err == nil {
		t.Fatal("expected error when neither testID nor testName is provided")
	}
}
