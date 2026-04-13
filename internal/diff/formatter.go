package diff

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FormatTable returns a human-readable table representation of the diff with +/-/~ indicators.
func FormatTable(result Result) string {
	var sb strings.Builder

	if len(result.GlobalChanges) > 0 {
		sb.WriteString("Global Defaults:\n")
		for _, c := range result.GlobalChanges {
			fmt.Fprintf(&sb, "  ~ %-25s %s -> %s\n", c.Field, c.OldValue, c.NewValue)
		}
		sb.WriteString("\n")
	}

	if len(result.ProfileChanges) > 0 {
		sb.WriteString("Profile:\n")
		for _, c := range result.ProfileChanges {
			fmt.Fprintf(&sb, "  ~ %-25s %s -> %s\n", c.Field, c.OldValue, c.NewValue)
		}
		sb.WriteString("\n")
	}

	if len(result.ScenarioChanges) > 0 {
		sb.WriteString("Scenarios:\n")
		for _, sc := range result.ScenarioChanges {
			switch sc.Type {
			case Added:
				fmt.Fprintf(&sb, "  + %s (index %d)\n", sc.Name, sc.NewIndex)
			case Removed:
				fmt.Fprintf(&sb, "  - %s (was index %d)\n", sc.Name, sc.OldIndex)
			case Modified:
				fmt.Fprintf(&sb, "  ~ %s:\n", sc.Name)
				for _, f := range sc.Fields {
					fmt.Fprintf(&sb, "      %-25s %s -> %s\n", f.Field, f.OldValue, f.NewValue)
				}
			}
		}
	}

	if len(result.GlobalChanges) == 0 && len(result.ProfileChanges) == 0 && len(result.ScenarioChanges) == 0 {
		sb.WriteString("No differences found.\n")
	}

	return sb.String()
}

// FormatJSON returns the diff as structured JSON.
func FormatJSON(result Result) ([]byte, error) {
	// Ensure slices are non-nil for clean JSON output
	if result.GlobalChanges == nil {
		result.GlobalChanges = []FieldChange{}
	}
	if result.ProfileChanges == nil {
		result.ProfileChanges = []FieldChange{}
	}
	if result.ScenarioChanges == nil {
		result.ScenarioChanges = []ScenarioChange{}
	}
	return json.MarshalIndent(result, "", "  ")
}
