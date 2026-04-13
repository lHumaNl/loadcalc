// Package config handles loading, parsing, validating, and defaulting of YAML configuration.
package config

import (
	"fmt"

	"loadcalc/pkg/units"

	"gopkg.in/yaml.v3"
)

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
	ProfileStability ProfileType = "stability"
	ProfileCapacity  ProfileType = "capacity"
	ProfileCustom    ProfileType = "custom"
	ProfileSpike     ProfileType = "spike"
)

// TestPlan is the top-level container for load test configuration.
type TestPlan struct {
	Version        string         `yaml:"version"`
	OutputFormat   string         `yaml:"output_format"`
	Scenarios      []Scenario     `yaml:"-"`
	GlobalDefaults GlobalDefaults `yaml:"global"`
	Profile        TestProfile    `yaml:"profile"`
}

// testPlanYAML is the on-disk YAML representation with scenarios as a map.
type testPlanYAML struct {
	Version        string              `yaml:"version"`
	OutputFormat   string              `yaml:"output_format,omitempty"`
	Scenarios      map[string]Scenario `yaml:"scenarios"`
	GlobalDefaults GlobalDefaults      `yaml:"global"`
	Profile        TestProfile         `yaml:"profile"`
}

// UnmarshalYAML reads scenarios from a YAML map keyed by scenario name.
func (tp *TestPlan) UnmarshalYAML(value *yaml.Node) error {
	var raw testPlanYAML
	if err := value.Decode(&raw); err != nil {
		return err
	}
	tp.Version = raw.Version
	tp.OutputFormat = raw.OutputFormat
	tp.GlobalDefaults = raw.GlobalDefaults
	tp.Profile = raw.Profile

	// Convert map to slice, preserving YAML key order by iterating the node directly.
	tp.Scenarios = nil
	if raw.Scenarios != nil {
		// Find the scenarios mapping node to preserve order.
		var scenariosNode *yaml.Node
		for i := 0; i < len(value.Content)-1; i += 2 {
			if value.Content[i].Value == "scenarios" {
				scenariosNode = value.Content[i+1]
				break
			}
		}
		if scenariosNode != nil && scenariosNode.Kind == yaml.MappingNode {
			for i := 0; i < len(scenariosNode.Content)-1; i += 2 {
				name := scenariosNode.Content[i].Value
				s, ok := raw.Scenarios[name]
				if !ok {
					return fmt.Errorf("scenario %q not found in parsed map", name)
				}
				s.Name = name
				tp.Scenarios = append(tp.Scenarios, s)
			}
		}
	}
	return nil
}

// MarshalYAML writes scenarios as a YAML map keyed by scenario name.
func (tp TestPlan) MarshalYAML() (interface{}, error) {
	raw := &testPlanYAML{
		Version:        tp.Version,
		OutputFormat:   tp.OutputFormat,
		GlobalDefaults: tp.GlobalDefaults,
		Profile:        tp.Profile,
	}

	// Use yaml.Node to preserve scenario order.
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, s := range tp.Scenarios {
		name := s.Name
		s.Name = "" // don't include name inside the map value
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: name}
		valNode := &yaml.Node{}
		if err := valNode.Encode(s); err != nil {
			return nil, err
		}
		node.Content = append(node.Content, keyNode, valNode)
	}

	// Build full document as a mapping node.
	doc := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	addField := func(key string, val interface{}) error {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
		valNode := &yaml.Node{}
		if err := valNode.Encode(val); err != nil {
			return err
		}
		doc.Content = append(doc.Content, keyNode, valNode)
		return nil
	}

	if raw.Version != "" {
		if err := addField("version", raw.Version); err != nil {
			return nil, err
		}
	}
	if raw.OutputFormat != "" {
		if err := addField("output_format", raw.OutputFormat); err != nil {
			return nil, err
		}
	}
	if err := addField("global", raw.GlobalDefaults); err != nil {
		return nil, err
	}
	doc.Content = append(doc.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "scenarios"},
		node,
	)
	if err := addField("profile", raw.Profile); err != nil {
		return nil, err
	}

	return doc, nil
}

// GlobalDefaults holds default settings for all scenarios.
type GlobalDefaults struct {
	Tool               Tool      `yaml:"tool"`
	LoadModel          LoadModel `yaml:"load_model"`
	PacingMultiplier   float64   `yaml:"pacing_multiplier"`
	DeviationTolerance float64   `yaml:"deviation_tolerance"`
	SpikeParticipate   bool      `yaml:"spike_participate"`
	GeneratorsCount    int       `yaml:"generators_count"`
	RangeDown          float64   `yaml:"range_down"`
	RangeUp            float64   `yaml:"range_up"`
}

// DefaultGlobalDefaults returns GlobalDefaults with spec-defined default values.
func DefaultGlobalDefaults() GlobalDefaults {
	return GlobalDefaults{
		PacingMultiplier:   3.0,
		DeviationTolerance: 2.5,
		SpikeParticipate:   true,
		GeneratorsCount:    1,
		RangeDown:          0.2,
		RangeUp:            0.5,
	}
}

// Scenario represents a single load test scenario.
type Scenario struct {
	LoadModel          *LoadModel          `yaml:"load_model,omitempty"`
	PacingMultiplier   *float64            `yaml:"pacing_multiplier,omitempty"`
	DeviationTolerance *float64            `yaml:"deviation_tolerance,omitempty"`
	SpikeParticipate   *bool               `yaml:"spike_participate,omitempty"`
	Name               string              `yaml:"-"`
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

// FineTune defines a second range with different increment for capacity profiles.
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
