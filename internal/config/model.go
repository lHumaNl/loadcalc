// Package config handles loading, parsing, validating, and defaulting of YAML configuration.
package config

import "loadcalc/pkg/units"

// Tool represents the load testing tool.
type Tool string

const (
	ToolLREPC  Tool = "lre_pc"
	ToolJMeter Tool = "jmeter"
)

// LoadModel represents open or closed load model.
type LoadModel string

const (
	LoadModelClosed LoadModel = "closed"
	LoadModelOpen   LoadModel = "open"
)

// ProfileType represents the test profile type.
type ProfileType string

const (
	ProfileStable    ProfileType = "stable"
	ProfileMaxSearch ProfileType = "max_search"
	ProfileCustom    ProfileType = "custom"
	ProfileSpike     ProfileType = "spike"
)

// TestPlan is the top-level container for load test configuration.
type TestPlan struct {
	Version        string         `yaml:"version"`
	OutputFormat   string         `yaml:"output_format"`
	Scenarios      []Scenario     `yaml:"scenarios"`
	GlobalDefaults GlobalDefaults `yaml:"global"`
	Profile        TestProfile    `yaml:"profile"`
}

// GlobalDefaults holds default settings for all scenarios.
type GlobalDefaults struct {
	Tool               Tool      `yaml:"tool"`
	LoadModel          LoadModel `yaml:"load_model"`
	PacingMultiplier   float64   `yaml:"pacing_multiplier"`
	DeviationTolerance float64   `yaml:"deviation_tolerance"`
	SpikeParticipate   bool      `yaml:"spike_participate"`
	GeneratorsCount    int       `yaml:"generators_count"`
}

// DefaultGlobalDefaults returns GlobalDefaults with spec-defined default values.
func DefaultGlobalDefaults() GlobalDefaults {
	return GlobalDefaults{
		PacingMultiplier:   3.0,
		DeviationTolerance: 2.5,
		SpikeParticipate:   true,
		GeneratorsCount:    1,
	}
}

// Scenario represents a single load test scenario.
type Scenario struct {
	LoadModel          *LoadModel          `yaml:"load_model,omitempty"`
	PacingMultiplier   *float64            `yaml:"pacing_multiplier,omitempty"`
	DeviationTolerance *float64            `yaml:"deviation_tolerance,omitempty"`
	SpikeParticipate   *bool               `yaml:"spike_participate,omitempty"`
	Name               string              `yaml:"name"`
	IntensityUnit      units.IntensityUnit `yaml:"intensity_unit"`
	ScriptID           int                 `yaml:"script_id"`
	TargetIntensity    float64             `yaml:"target_intensity"`
	MaxScriptTimeMs    int                 `yaml:"max_script_time_ms"`
	BackgroundPercent  float64             `yaml:"background_percent"`
	Background         bool                `yaml:"background"`
}

// TestProfile defines the test execution profile.
type TestProfile struct {
	FineTune             *FineTune     `yaml:"fine_tune,omitempty"`
	Type                 ProfileType   `yaml:"type"`
	Steps                []ProfileStep `yaml:"steps,omitempty"`
	NumSteps             int           `yaml:"num_steps,omitempty"`
	Percent              float64       `yaml:"percent,omitempty"`
	DefaultStabilitySec  int           `yaml:"default_stability_sec"`
	StartPercent         float64       `yaml:"start_percent,omitempty"`
	StepIncrement        float64       `yaml:"step_increment,omitempty"`
	DefaultImpactSec     int           `yaml:"default_impact_sec"`
	DefaultRampupSec     int           `yaml:"default_rampup_sec"`
	DefaultRampdownSec   int           `yaml:"default_rampdown_sec"`
	BasePercent          float64       `yaml:"base_percent,omitempty"`
	SpikeStartIncrement  float64       `yaml:"spike_start_increment,omitempty"`
	SpikeIncrementGrowth float64       `yaml:"spike_increment_growth,omitempty"`
	NumSpikes            int           `yaml:"num_spikes,omitempty"`
	SpiketimeSec         int           `yaml:"spiketime_sec,omitempty"`
	CooldownSec          int           `yaml:"cooldown_sec,omitempty"`
}

// FineTune defines a second range with different increment for max_search profiles.
type FineTune struct {
	AfterPercent  float64 `yaml:"after_percent"`
	StepIncrement float64 `yaml:"step_increment"`
	NumSteps      int     `yaml:"num_steps"`
}

// ProfileStep represents a single step in a test profile.
type ProfileStep struct {
	RampupSec       *int    `yaml:"rampup_sec,omitempty"`
	ImpactSec       *int    `yaml:"impact_sec,omitempty"`
	StabilitySec    *int    `yaml:"stability_sec,omitempty"`
	RampdownSec     *int    `yaml:"rampdown_sec,omitempty"`
	PercentOfTarget float64 `yaml:"percent"`
}
