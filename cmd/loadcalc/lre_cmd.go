package main

import (
	"fmt"
	"os"
	"strings"

	"loadcalc/internal/integration"

	"github.com/spf13/cobra"
)

func newLRECmd() *cobra.Command {
	lreCmd := &cobra.Command{
		Use:   "lre",
		Short: "LRE Performance Center integration commands",
	}
	lreCmd.AddCommand(newLREPushCmd())
	lreCmd.AddCommand(newLREListTestsCmd())
	lreCmd.AddCommand(newLREListScriptsCmd())
	return lreCmd
}

func lreFlags(cmd *cobra.Command, server, domain, project, user, password *string) {
	cmd.Flags().StringVar(server, "server", "", "LRE PC server URL (e.g. https://server:port/LoadTest/rest)")
	cmd.Flags().StringVar(domain, "domain", "", "LRE PC domain")
	cmd.Flags().StringVar(project, "project", "", "LRE PC project")
	cmd.Flags().StringVar(user, "user", "", "LRE PC username (or LOADCALC_LRE_USER env)")
	cmd.Flags().StringVar(password, "password", "", "LRE PC password (or LOADCALC_LRE_PASSWORD env)")
	_ = cmd.MarkFlagRequired("server")
	_ = cmd.MarkFlagRequired("domain")
	_ = cmd.MarkFlagRequired("project")
}

func resolveCredentials(user, password string) (resolvedUser, resolvedPassword string) {
	resolvedUser = user
	resolvedPassword = password
	if resolvedUser == "" {
		resolvedUser = os.Getenv("LOADCALC_LRE_USER")
	}
	if resolvedPassword == "" {
		resolvedPassword = os.Getenv("LOADCALC_LRE_PASSWORD")
	}
	return resolvedUser, resolvedPassword
}

func newLREPushCmd() *cobra.Command {
	var inputPath, server, domain, project, user, password, scenariosDir, csvDelimiter string
	var testName, testFolder string
	var scenarioFiles, scenarios []string
	var testID int
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push calculation results to LRE PC test",
		RunE: func(cmd *cobra.Command, _ []string) error {
			user, password = resolveCredentials(user, password)
			if user == "" || password == "" {
				return fmt.Errorf("credentials required: use --user/--password or LOADCALC_LRE_USER/LOADCALC_LRE_PASSWORD env vars")
			}

			// Run calculation pipeline
			results, calcErr := runPipeline(inputPath, scenarioFiles, scenariosDir, csvDelimiter)
			if calcErr != nil {
				return fmt.Errorf("calculation pipeline: %w", calcErr)
			}

			// Filter scenarios if specified
			if len(scenarios) > 0 {
				filterSet := make(map[string]bool)
				for _, s := range scenarios {
					filterSet[s] = true
				}
				filtered := results
				filtered.ScenarioResults = nil
				for _, sr := range results.ScenarioResults {
					if filterSet[sr.Scenario.Name] {
						filtered.ScenarioResults = append(filtered.ScenarioResults, sr)
					}
				}
				results = filtered
			}

			// Create client and authenticate
			client := integration.NewLREClient(server, domain, project)
			if authErr := client.Authenticate(user, password); authErr != nil {
				return fmt.Errorf("authentication failed: %w", authErr)
			}
			defer func() { _ = client.Logout() }()

			// Push
			pr, pushErr := integration.PushToLRE(client, testID, testName, testFolder, results, dryRun)
			if pushErr != nil {
				return fmt.Errorf("push failed: %w", pushErr)
			}

			// Print results
			if dryRun {
				cmd.Println("[DRY RUN] No changes were made.")
			}

			cmd.Println(fmt.Sprintf("%-20s %-10s %8s %10s %8s",
				"Scenario", "Action", "GroupID", "Pacing(ms)", "Vusers"))
			cmd.Println(strings.Repeat("-", 60))
			for _, a := range pr.Actions {
				cmd.Println(fmt.Sprintf("%-20s %-10s %8d %10.0f %8d",
					a.ScenarioName, a.ActionType, a.GroupID, a.PacingMS, a.VuserCount))
			}

			for _, w := range pr.Warnings {
				cmd.Println(fmt.Sprintf("[WARN] %s", w))
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&inputPath, "input", "i", "", "Input config file path")
	cmd.Flags().IntVar(&testID, "test-id", 0, "LRE PC test ID (mutually exclusive with --test-name)")
	cmd.Flags().StringVar(&testName, "test-name", "", "Name for a new LRE PC test (mutually exclusive with --test-id)")
	cmd.Flags().StringVar(&testFolder, "test-folder", "Subject", "Folder path in LRE PC for new test")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Dry run (no API calls)")
	cmd.Flags().StringSliceVar(&scenarios, "scenarios", nil, "Scenario names to push (default: all)")
	cmd.Flags().StringSliceVar(&scenarioFiles, "scenario-files", nil, "Additional scenario files")
	cmd.Flags().StringVar(&scenariosDir, "scenarios-dir", "", "Directory with scenario files")
	cmd.Flags().StringVar(&csvDelimiter, "csv-delimiter", ";", "CSV delimiter character")
	lreFlags(cmd, &server, &domain, &project, &user, &password)
	_ = cmd.MarkFlagRequired("input")
	return cmd
}

// lreListItem represents a named item with an ID from LRE.
type lreListItem struct {
	Name string
	ID   int
}

func newLREListCmd(use, short string, fetchFn func(*integration.LREClient) ([]lreListItem, error)) *cobra.Command {
	var server, domain, project, user, password string

	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			user, password = resolveCredentials(user, password)
			if user == "" || password == "" {
				return fmt.Errorf("credentials required")
			}

			client := integration.NewLREClient(server, domain, project)
			if err := client.Authenticate(user, password); err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}
			defer func() { _ = client.Logout() }()

			items, err := fetchFn(client)
			if err != nil {
				return err
			}

			cmd.Println(fmt.Sprintf("%-8s %s", "ID", "Name"))
			cmd.Println(strings.Repeat("-", 40))
			for _, item := range items {
				cmd.Println(fmt.Sprintf("%-8d %s", item.ID, item.Name))
			}
			return nil
		},
	}
	lreFlags(cmd, &server, &domain, &project, &user, &password)
	return cmd
}

func newLREListTestsCmd() *cobra.Command {
	return newLREListCmd("list-tests", "List tests in LRE PC project", func(client *integration.LREClient) ([]lreListItem, error) {
		tests, err := client.ListTests()
		if err != nil {
			return nil, fmt.Errorf("listing tests: %w", err)
		}
		items := make([]lreListItem, len(tests))
		for i, t := range tests {
			items[i] = lreListItem{ID: t.ID, Name: t.Name}
		}
		return items, nil
	})
}

func newLREListScriptsCmd() *cobra.Command {
	return newLREListCmd("list-scripts", "List scripts in LRE PC project", func(client *integration.LREClient) ([]lreListItem, error) {
		scripts, err := client.ListScripts()
		if err != nil {
			return nil, fmt.Errorf("listing scripts: %w", err)
		}
		items := make([]lreListItem, len(scripts))
		for i, s := range scripts {
			items[i] = lreListItem{ID: s.ID, Name: s.Name}
		}
		return items, nil
	})
}
