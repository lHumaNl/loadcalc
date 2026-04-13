package main

import (
	"fmt"

	"loadcalc/internal/config"
	"loadcalc/internal/merge"

	"github.com/spf13/cobra"
)

func newMergeCmd() *cobra.Command {
	var outputPath, dir string

	cmd := &cobra.Command{
		Use:   "merge [config1.yaml config2.yaml ...]",
		Short: "Merge multiple configuration files",
		Long:  "Merge multiple loadcalc YAML configs. Globals and profile from the first file; scenarios concatenated.",
		RunE: func(cmd *cobra.Command, args []string) error {
			var plans []*config.TestPlan

			if dir != "" {
				loaded, err := merge.LoadPlansFromDir(dir)
				if err != nil {
					return fmt.Errorf("loading configs from directory: %w", err)
				}
				plans = append(plans, loaded...)
			}

			for _, path := range args {
				plan, err := config.LoadFromYAML(path)
				if err != nil {
					return fmt.Errorf("loading %s: %w", path, err)
				}
				plans = append(plans, plan)
			}

			if len(plans) == 0 {
				return fmt.Errorf("no config files specified; provide file arguments or --dir")
			}

			result := merge.Plans(plans)

			// Print warnings to stderr
			for _, w := range result.Warnings {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "WARNING [%s]: %s\n", w.Field, w.Message)
			}

			dest := outputPath
			if dest == "" {
				dest = "-"
			}
			return merge.WriteMergedYAML(result.Plan, dest)
		},
	}
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (default: stdout)")
	cmd.Flags().StringVar(&dir, "dir", "", "Directory containing YAML config files to merge")
	return cmd
}
