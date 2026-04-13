package main

import (
	"fmt"
	"strings"

	"loadcalc/internal/config"
	"loadcalc/internal/whatif"

	"github.com/spf13/cobra"
)

func newWhatIfCmd() *cobra.Command {
	var inputPath, scenariosDir, csvDelimiter string
	var scenarioFiles, setFlags []string

	cmd := &cobra.Command{
		Use:   "what-if",
		Short: "Compare baseline vs modified calculation with overrides",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Parse --set flags into map
			overrides := make(map[string]string)
			for _, s := range setFlags {
				parts := strings.SplitN(s, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid --set format %q, expected key=value", s)
				}
				overrides[parts[0]] = parts[1]
			}
			if len(overrides) == 0 {
				return fmt.Errorf("at least one --set flag is required")
			}

			// 1. Load config + scenarios
			plan, err := config.LoadFromYAML(inputPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			delim := ';'
			if csvDelimiter != "" {
				delim = rune(csvDelimiter[0])
			}
			for _, sf := range scenarioFiles {
				var extra []config.Scenario
				extra, err = config.LoadScenariosFromFile(sf, delim)
				if err != nil {
					return fmt.Errorf("loading scenarios from %s: %w", sf, err)
				}
				plan.Scenarios = append(plan.Scenarios, extra...)
			}
			if scenariosDir != "" {
				var extra []config.Scenario
				extra, err = config.LoadScenariosFromDir(scenariosDir, delim)
				if err != nil {
					return fmt.Errorf("loading scenarios from dir %s: %w", scenariosDir, err)
				}
				plan.Scenarios = append(plan.Scenarios, extra...)
			}

			// 2. Resolve defaults
			plan = config.ResolveDefaults(plan)

			// 3. Validate baseline
			valErrs := config.Validate(plan)
			if config.HasErrors(valErrs) {
				return fmt.Errorf("baseline validation failed")
			}

			// 4. Run baseline calculation
			baselineResults, err := runPipelineFromPlan(plan)
			if err != nil {
				return fmt.Errorf("baseline calculation: %w", err)
			}

			// 5. Deep-copy and apply overrides
			modPlan, err := whatif.DeepCopyPlan(plan)
			if err != nil {
				return fmt.Errorf("deep copy: %w", err)
			}
			if applyErr := whatif.ApplyOverrides(modPlan, overrides); applyErr != nil {
				return fmt.Errorf("applying overrides: %w", applyErr)
			}

			// 6. Resolve defaults on modified config
			modPlan = config.ResolveDefaults(modPlan)

			// 7. Run modified calculation
			modResults, err := runPipelineFromPlan(modPlan)
			if err != nil {
				return fmt.Errorf("modified calculation: %w", err)
			}

			// 8. Compare and display
			result := whatif.CompareResults(baselineResults, modResults, overrides)
			cmd.Print(whatif.FormatComparison(result))
			return nil
		},
	}
	cmd.Flags().StringVarP(&inputPath, "input", "i", "", "Input config file path")
	cmd.Flags().StringSliceVar(&scenarioFiles, "scenarios", nil, "Additional scenario files (CSV or XLSX)")
	cmd.Flags().StringVar(&scenariosDir, "scenarios-dir", "", "Directory with scenario files")
	cmd.Flags().StringVar(&csvDelimiter, "csv-delimiter", ";", "CSV delimiter character")
	cmd.Flags().StringArrayVar(&setFlags, "set", nil, "Override in key=value format (repeatable)")
	_ = cmd.MarkFlagRequired("input")
	return cmd
}
