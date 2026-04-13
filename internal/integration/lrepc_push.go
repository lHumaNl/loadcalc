package integration

import (
	"fmt"
	"log/slog"
	"math"

	"loadcalc/internal/engine"
)

// PushAction describes a single action taken during push.
type PushAction struct {
	ScenarioName string
	ActionType   string // "created", "updated", "skipped"
	GroupID      int
	VuserCount   int
	PacingMS     float64
	ScriptID     int
}

// PushResult holds the outcome of a push operation.
type PushResult struct {
	Actions  []PushAction
	Warnings []string
}

// resolveTestID validates inputs and resolves the test ID, creating a new test if needed.
func resolveTestID(client *LREClient, testID int, testName, testFolder string, dryRun bool) (int, error) {
	if testID > 0 && testName != "" {
		return 0, fmt.Errorf("test-id and test-name are mutually exclusive")
	}
	if testID == 0 && testName == "" {
		return 0, fmt.Errorf("either test-id or test-name must be provided")
	}
	if testID > 0 {
		slog.Info("using existing test", "id", testID)
		return testID, nil
	}
	if testFolder == "" {
		testFolder = "Subject"
	}
	if dryRun {
		slog.Info("dry-run: would create test", "name", testName, "folder", testFolder)
		return 0, nil
	}
	created, err := client.CreateTest(testName, testFolder)
	if err != nil {
		return 0, fmt.Errorf("creating test: %w", err)
	}
	slog.Info("created new test", "id", created.ID, "name", testName)
	return created.ID, nil
}

// PushToLRE pushes calculation results to an LRE PC test.
// If testID > 0, updates an existing test. If testID == 0 and testName != "", creates a new test.
// If dryRun is true, no API calls are made but actions are collected.
func PushToLRE(client *LREClient, testID int, testName, testFolder string, results engine.CalculationResults, dryRun bool) (*PushResult, error) {
	resolvedID, err := resolveTestID(client, testID, testName, testFolder, dryRun)
	if err != nil {
		return nil, err
	}
	testID = resolvedID

	pr := &PushResult{}

	// List existing groups
	var existingGroups []LREGroup
	if !dryRun {
		var err error
		existingGroups, err = client.ListGroups(testID)
		if err != nil {
			return nil, fmt.Errorf("listing groups: %w", err)
		}
	}

	// Build lookup by name
	groupByName := make(map[string]LREGroup)
	for _, g := range existingGroups {
		groupByName[g.Name] = g
	}

	for _, sr := range results.ScenarioResults {
		// Skip open model scenarios
		if sr.IsOpenModel {
			pr.Warnings = append(pr.Warnings, fmt.Sprintf("skipping open model scenario %q (not supported by LRE PC)", sr.Scenario.Name))
			slog.Info("skipping open model scenario", "name", sr.Scenario.Name)
			continue
		}

		// Get the max-step thread count (first step for 100% target)
		vuserCount := 0
		if len(sr.OptimizeResult.StepResults) > 0 {
			// Use the last step's thread count (highest load)
			lastStep := sr.OptimizeResult.StepResults[len(sr.OptimizeResult.StepResults)-1]
			vuserCount = lastStep.Threads
		}

		pacingMS := sr.OptimizeResult.BestPacingMS
		pacingInt := int(math.Round(pacingMS))

		action := PushAction{
			ScenarioName: sr.Scenario.Name,
			VuserCount:   vuserCount,
			PacingMS:     pacingMS,
			ScriptID:     sr.Scenario.ScriptID,
		}

		existing, found := groupByName[sr.Scenario.Name]

		if found {
			// Update existing group
			action.ActionType = "updated"
			action.GroupID = existing.ID

			if !dryRun {
				err := client.UpdateGroup(testID, existing.ID, LREGroup{
					VuserCount: vuserCount,
				})
				if err != nil {
					return nil, fmt.Errorf("updating group %q: %w", sr.Scenario.Name, err)
				}

				err = client.UpdateRuntimeSettings(testID, existing.ID, LRERuntimeSettings{
					Pacing: LREPacing{
						Type:     "ConstantPacing",
						MinDelay: pacingInt,
						MaxDelay: pacingInt,
					},
				})
				if err != nil {
					return nil, fmt.Errorf("updating runtime settings for %q: %w", sr.Scenario.Name, err)
				}
			}
		} else {
			// Create new group
			action.ActionType = "created"

			if !dryRun {
				created, err := client.CreateGroup(testID, LREGroup{
					Name:       sr.Scenario.Name,
					VuserCount: vuserCount,
					ScriptID:   sr.Scenario.ScriptID,
				})
				if err != nil {
					return nil, fmt.Errorf("creating group %q: %w", sr.Scenario.Name, err)
				}
				action.GroupID = created.ID

				err = client.UpdateRuntimeSettings(testID, created.ID, LRERuntimeSettings{
					Pacing: LREPacing{
						Type:     "ConstantPacing",
						MinDelay: pacingInt,
						MaxDelay: pacingInt,
					},
				})
				if err != nil {
					return nil, fmt.Errorf("updating runtime settings for new group %q: %w", sr.Scenario.Name, err)
				}
			}
		}

		pr.Actions = append(pr.Actions, action)
		slog.Info("push action", "scenario", sr.Scenario.Name, "action", action.ActionType, "vusers", vuserCount, "pacing_ms", pacingMS)
	}

	return pr, nil
}
