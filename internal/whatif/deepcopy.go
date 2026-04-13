package whatif

import (
	"encoding/json"

	"loadcalc/internal/config"
)

// DeepCopyPlan creates a deep copy of a TestPlan via JSON round-trip.
func DeepCopyPlan(plan *config.TestPlan) (*config.TestPlan, error) {
	data, err := json.Marshal(plan)
	if err != nil {
		return nil, err
	}
	var cloned config.TestPlan
	if err := json.Unmarshal(data, &cloned); err != nil {
		return nil, err
	}
	return &cloned, nil
}
