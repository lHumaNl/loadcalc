package config

import "fmt"

// Severity represents the severity of a validation error.
type Severity string

const (
	SeverityError   Severity = "Error"
	SeverityWarning Severity = "Warning"
)

// ValidationError represents a single validation issue.
type ValidationError struct {
	Field    string
	Message  string
	Severity Severity
}

func (v ValidationError) String() string {
	return fmt.Sprintf("[%s] %s: %s", v.Severity, v.Field, v.Message)
}

// Validate checks all validation rules from §3.3 against a TestPlan.
// The plan should have defaults resolved before validation.
func Validate(plan *TestPlan) []ValidationError {
	errs := make([]ValidationError, 0, len(plan.Scenarios))

	errs = append(errs, validateGlobals(plan)...)

	for i, s := range plan.Scenarios {
		errs = append(errs, validateScenario(s, plan.GlobalDefaults.Tool, i)...)
	}

	errs = append(errs, validateProfile(plan)...)

	return errs
}

func validateGlobals(plan *TestPlan) []ValidationError {
	var errs []ValidationError

	if plan.GlobalDefaults.Tool == "" {
		errs = append(errs, ValidationError{
			Field:    "global.tool",
			Message:  "tool is required",
			Severity: SeverityError,
		})
	}

	if plan.GlobalDefaults.GeneratorsCount < 1 {
		errs = append(errs, ValidationError{
			Field:    "global.generators_count",
			Message:  "must be >= 1",
			Severity: SeverityError,
		})
	}

	if plan.GlobalDefaults.Tool == ToolLREPC && plan.GlobalDefaults.LoadModel == LoadModelOpen {
		errs = append(errs, ValidationError{
			Field:    "global",
			Message:  "open load model is not supported with lre_pc",
			Severity: SeverityError,
		})
	}

	return errs
}

func validateScenario(s Scenario, tool Tool, i int) []ValidationError {
	prefix := fmt.Sprintf("scenarios[%d]", i)
	errs := validateScenarioFields(s, tool, prefix)
	errs = append(errs, validateScenarioConstraints(s, tool, prefix)...)
	return errs
}

func validateScenarioFields(s Scenario, tool Tool, prefix string) []ValidationError {
	var errs []ValidationError

	if s.Name == "" {
		errs = append(errs, ValidationError{
			Field:    prefix + ".name",
			Message:  "name is required",
			Severity: SeverityError,
		})
	}

	if tool == ToolLREPC && s.ScriptID <= 0 {
		errs = append(errs, ValidationError{
			Field:    prefix + ".script_id",
			Message:  "script_id must be > 0 for lre_pc tool",
			Severity: SeverityError,
		})
	}

	if s.MaxScriptTimeMs <= 0 {
		errs = append(errs, ValidationError{
			Field:    prefix + ".max_script_time_ms",
			Message:  "must be > 0",
			Severity: SeverityError,
		})
	}

	if s.TargetIntensity <= 0 {
		errs = append(errs, ValidationError{
			Field:    prefix + ".target_intensity",
			Message:  "must be > 0",
			Severity: SeverityError,
		})
	}

	if s.PacingMultiplier != nil && *s.PacingMultiplier < 1.0 {
		errs = append(errs, ValidationError{
			Field:    prefix + ".pacing_multiplier",
			Message:  "must be >= 1.0",
			Severity: SeverityError,
		})
	}

	if s.DeviationTolerance != nil && *s.DeviationTolerance < 0 {
		errs = append(errs, ValidationError{
			Field:    prefix + ".deviation_tolerance",
			Message:  "must be >= 0",
			Severity: SeverityError,
		})
	}

	return errs
}

func validateScenarioConstraints(s Scenario, tool Tool, prefix string) []ValidationError {
	var errs []ValidationError

	if s.Background && (s.BackgroundPercent < 0 || s.BackgroundPercent > 100) {
		errs = append(errs, ValidationError{
			Field:    prefix + ".background_percent",
			Message:  "must be between 0 and 100",
			Severity: SeverityError,
		})
	}

	if s.LoadModel != nil && *s.LoadModel == LoadModelOpen && tool == ToolLREPC {
		errs = append(errs, ValidationError{
			Field:    prefix + ".load_model",
			Message:  "open load model is not supported with lre_pc",
			Severity: SeverityError,
		})
	}

	if s.Background && s.SpikeParticipate != nil && *s.SpikeParticipate {
		errs = append(errs, ValidationError{
			Field:    prefix,
			Message:  "background scenario with spike_participate=true; background wins, spike participation ignored",
			Severity: SeverityWarning,
		})
	}

	return errs
}

func validateProfile(plan *TestPlan) []ValidationError {
	p := &plan.Profile

	switch p.Type {
	case ProfileStable:
		return validateStableProfile(p)
	case ProfileMaxSearch:
		return validateMaxSearchProfile(p)
	case ProfileCustom:
		return validateCustomProfile(p)
	case ProfileSpike:
		return validateSpikeProfile(p)
	}
	return nil
}

func validateStableProfile(p *TestProfile) []ValidationError {
	var errs []ValidationError
	if p.Percent <= 0 {
		errs = append(errs, ValidationError{
			Field:    "profile.percent",
			Message:  "must be > 0 for stable profile",
			Severity: SeverityError,
		})
	}
	return errs
}

func validateMaxSearchProfile(p *TestProfile) []ValidationError {
	var errs []ValidationError
	if p.StartPercent <= 0 {
		errs = append(errs, ValidationError{
			Field:    "profile.start_percent",
			Message:  "must be > 0",
			Severity: SeverityError,
		})
	}
	if p.StepIncrement <= 0 {
		errs = append(errs, ValidationError{
			Field:    "profile.step_increment",
			Message:  "must be > 0",
			Severity: SeverityError,
		})
	}

	generatedSteps := generateMaxSearchSteps(p)

	for _, step := range p.Steps {
		found := false
		for _, gs := range generatedSteps {
			if step.PercentOfTarget == gs {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, ValidationError{
				Field:    "profile.steps",
				Message:  fmt.Sprintf("step override at percent %.1f does not match any generated step", step.PercentOfTarget),
				Severity: SeverityWarning,
			})
		}
	}

	if p.FineTune != nil {
		lastBaseStep := p.StartPercent + float64(p.NumSteps-1)*p.StepIncrement
		if p.FineTune.AfterPercent != lastBaseStep {
			errs = append(errs, ValidationError{
				Field:    "profile.fine_tune.after_percent",
				Message:  fmt.Sprintf("must match last generated step (%.1f), got %.1f", lastBaseStep, p.FineTune.AfterPercent),
				Severity: SeverityError,
			})
		}
	}
	return errs
}

func validateCustomProfile(p *TestProfile) []ValidationError {
	var errs []ValidationError
	if len(p.Steps) == 0 {
		errs = append(errs, ValidationError{
			Field:    "profile.steps",
			Message:  "custom profile requires at least one step",
			Severity: SeverityError,
		})
	}
	for i, step := range p.Steps {
		if step.PercentOfTarget <= 0 {
			errs = append(errs, ValidationError{
				Field:    fmt.Sprintf("profile.steps[%d].percent", i),
				Message:  "must be > 0",
				Severity: SeverityError,
			})
		}
	}
	return errs
}

func validateSpikeProfile(p *TestProfile) []ValidationError {
	var errs []ValidationError
	if p.BasePercent <= 0 {
		errs = append(errs, ValidationError{
			Field:    "profile.base_percent",
			Message:  "must be > 0",
			Severity: SeverityError,
		})
	}
	return errs
}

// generateMaxSearchSteps generates the percent values for a max_search profile.
func generateMaxSearchSteps(p *TestProfile) []float64 {
	var steps []float64
	for i := 0; i < p.NumSteps; i++ {
		steps = append(steps, p.StartPercent+float64(i)*p.StepIncrement)
	}
	if p.FineTune != nil {
		last := steps[len(steps)-1]
		for i := 0; i < p.FineTune.NumSteps; i++ {
			steps = append(steps, last+float64(i+1)*p.FineTune.StepIncrement)
		}
	}
	return steps
}

// HasErrors returns true if any validation error has Error severity.
func HasErrors(errs []ValidationError) bool {
	for _, e := range errs {
		if e.Severity == SeverityError {
			return true
		}
	}
	return false
}
