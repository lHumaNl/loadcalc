package merge

import (
	"fmt"
	"os"

	"loadcalc/internal/config"

	"gopkg.in/yaml.v3"
)

// WriteMergedYAML marshals the plan to YAML and writes it to dest.
// If dest is empty or "-", it writes to stdout.
func WriteMergedYAML(plan *config.TestPlan, dest string) error {
	data, err := yaml.Marshal(plan)
	if err != nil {
		return fmt.Errorf("marshaling YAML: %w", err)
	}

	if dest == "" || dest == "-" {
		_, err = os.Stdout.Write(data)
		return err
	}

	return os.WriteFile(dest, data, 0o600)
}
