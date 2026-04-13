// Package whatif provides what-if analysis: config overrides, comparison, and formatting.
package whatif

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"loadcalc/internal/config"
)

var scenarioPathRe = regexp.MustCompile(`^scenarios\[(.+?)\]\.(.+)$`)

// ApplyOverrides modifies a TestPlan based on dot-notation overrides.
func ApplyOverrides(plan *config.TestPlan, overrides map[string]string) error {
	for path, value := range overrides {
		if err := applyOne(plan, path, value); err != nil {
			return fmt.Errorf("override %q=%q: %w", path, value, err)
		}
	}
	return nil
}

func applyOne(plan *config.TestPlan, path, value string) error {
	if strings.HasPrefix(path, "global.") {
		return applyGlobal(&plan.GlobalDefaults, strings.TrimPrefix(path, "global."), value)
	}
	if strings.HasPrefix(path, "profile.") {
		return applyProfile(&plan.Profile, strings.TrimPrefix(path, "profile."), value)
	}
	if m := scenarioPathRe.FindStringSubmatch(path); m != nil {
		return applyScenario(plan, m[1], m[2], value)
	}
	return fmt.Errorf("unknown path: %s", path)
}

func applyGlobal(g *config.GlobalDefaults, field, value string) error {
	switch field {
	case "pacing_multiplier":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float: %w", err)
		}
		g.PacingMultiplier = v
	case "generators_count":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid int: %w", err)
		}
		g.GeneratorsCount = v
	case "deviation_tolerance":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float: %w", err)
		}
		g.DeviationTolerance = v
	case "tool":
		g.Tool = config.Tool(value)
	case "load_model":
		g.LoadModel = config.LoadModel(value)
	case "spike_participate":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid bool: %w", err)
		}
		g.SpikeParticipate = v
	default:
		return fmt.Errorf("unknown global field: %s", field)
	}
	return nil
}

func applyProfile(p *config.TestProfile, field, value string) error {
	switch field {
	case "num_steps":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid int: %w", err)
		}
		p.NumSteps = v
	case "start_percent":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float: %w", err)
		}
		p.StartPercent = v
	case "step_increment":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float: %w", err)
		}
		p.StepIncrement = v
	case "percent":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float: %w", err)
		}
		p.Percent = v
	default:
		return fmt.Errorf("unknown profile field: %s", field)
	}
	return nil
}

func applyScenario(plan *config.TestPlan, selector, field, value string) error {
	idx, err := strconv.Atoi(selector)
	if err != nil {
		idx = -1
		for i, s := range plan.Scenarios {
			if s.Name == selector {
				idx = i
				break
			}
		}
		if idx == -1 {
			return fmt.Errorf("scenario not found: %s", selector)
		}
	}
	if idx < 0 || idx >= len(plan.Scenarios) {
		return fmt.Errorf("scenario index %d out of range (have %d)", idx, len(plan.Scenarios))
	}

	return setScenarioField(&plan.Scenarios[idx], field, value)
}

func setScenarioField(s *config.Scenario, field, value string) error {
	switch field {
	case "target_intensity":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float: %w", err)
		}
		s.TargetIntensity = v
	case "max_script_time_ms":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid int: %w", err)
		}
		s.MaxScriptTimeMs = v
	case "pacing_multiplier":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float: %w", err)
		}
		s.PacingMultiplier = &v
	case "deviation_tolerance":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float: %w", err)
		}
		s.DeviationTolerance = &v
	case "load_model":
		lm := config.LoadModel(value)
		s.LoadModel = &lm
	case "spike_participate":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid bool: %w", err)
		}
		s.SpikeParticipate = &v
	default:
		return fmt.Errorf("unknown scenario field: %s", field)
	}
	return nil
}
