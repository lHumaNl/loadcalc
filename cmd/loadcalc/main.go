package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"loadcalc/internal/config"
	"loadcalc/internal/diff"
	"loadcalc/internal/engine"
	"loadcalc/internal/integration"
	"loadcalc/internal/output"
	"loadcalc/internal/profile"
	"loadcalc/internal/tui"
	"loadcalc/pkg/units"

	"github.com/spf13/cobra"
)

// Version is set via ldflags at build time: -X main.Version=<tag>
var Version = "dev"

func main() {
	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var logLevel string

	root := &cobra.Command{
		Use:           "loadcalc",
		Short:         "Load test parameter calculator",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			setupLogging(logLevel)
		},
	}

	root.PersistentFlags().StringVar(&logLevel, "log-level", "warn", "Log level: debug|info|warn|error")

	root.AddCommand(newCalculateCmd())
	root.AddCommand(newTemplateCmd())
	root.AddCommand(newValidateCmd())
	root.AddCommand(newVersionCmd())
	root.AddCommand(newTUICmd())
	root.AddCommand(newJMXCmd())
	root.AddCommand(newDiffCmd())
	root.AddCommand(newWhatIfCmd())
	root.AddCommand(newMergeCmd())
	root.AddCommand(newLRECmd())

	return root
}

func setupLogging(level string) {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelWarn
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(handler))
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("loadcalc " + Version)
			return nil
		},
	}
}

func newValidateCmd() *cobra.Command {
	var inputPath, scenariosDir, csvDelimiter string
	var scenarioFiles []string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a configuration file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			plan, err := config.LoadFromYAML(inputPath)
			if err != nil {
				cmd.Println(fmt.Sprintf("[Error] Failed to load config: %v", err))
				return err
			}
			slog.Info("config loaded", "path", inputPath)

			// Load additional scenarios
			delim := ';'
			if csvDelimiter != "" {
				delim = rune(csvDelimiter[0])
			}
			for _, sf := range scenarioFiles {
				extra, err := config.LoadScenariosFromFile(sf, delim)
				if err != nil {
					return fmt.Errorf("loading scenarios from %s: %w", sf, err)
				}
				plan.Scenarios = append(plan.Scenarios, extra...)
			}
			if scenariosDir != "" {
				extra, err := config.LoadScenariosFromDir(scenariosDir, delim)
				if err != nil {
					return fmt.Errorf("loading scenarios from dir %s: %w", scenariosDir, err)
				}
				plan.Scenarios = append(plan.Scenarios, extra...)
			}

			plan = config.ResolveDefaults(plan)
			errs := config.Validate(plan)

			if len(errs) > 0 {
				for _, e := range errs {
					cmd.Println(e.String())
				}
			}

			if config.HasErrors(errs) {
				slog.Error("validation failed", "error_count", len(errs))
				return fmt.Errorf("validation failed with errors")
			}

			cmd.Println("Validation passed.")
			slog.Info("validation passed")
			return nil
		},
	}
	cmd.Flags().StringVarP(&inputPath, "input", "i", "", "Input config file path")
	cmd.Flags().StringSliceVar(&scenarioFiles, "scenarios", nil, "Additional scenario files (CSV or XLSX)")
	cmd.Flags().StringVar(&scenariosDir, "scenarios-dir", "", "Directory with scenario files")
	cmd.Flags().StringVar(&csvDelimiter, "csv-delimiter", ";", "CSV delimiter character")
	_ = cmd.MarkFlagRequired("input")
	return cmd
}

func newTemplateCmd() *cobra.Command {
	var format, outputPath, csvDelimiter string

	cmd := &cobra.Command{
		Use:   "template",
		Short: "Generate a template configuration file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			switch strings.ToLower(format) {
			case "yaml":
				return writeYAMLTemplate(outputPath)
			case "csv":
				delim := ';'
				if csvDelimiter != "" {
					delim = rune(csvDelimiter[0])
				}
				return writeCSVTemplate(outputPath, delim)
			case "xlsx":
				cmd.Println("XLSX template not yet implemented")
				return nil
			default:
				return fmt.Errorf("unknown format: %s (use yaml, csv, or xlsx)", format)
			}
		},
	}
	cmd.Flags().StringVar(&format, "format", "yaml", "Template format: yaml|csv|xlsx")
	cmd.Flags().StringVar(&csvDelimiter, "csv-delimiter", ";", "CSV delimiter character")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path")
	_ = cmd.MarkFlagRequired("output")
	return cmd
}

func writeCSVTemplate(path string, delimiter rune) error {
	headers := []string{
		"name", "script_id", "target_intensity", "intensity_unit",
		"max_script_time_ms", "background", "background_percent",
		"load_model", "pacing_multiplier", "deviation_tolerance", "spike_participate",
	}
	d := string(delimiter)
	line := strings.Join(headers, d) + "\n"
	return os.WriteFile(path, []byte(line), 0o600)
}

func newTUICmd() *cobra.Command {
	var inputPath, outputPath string

	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Interactive TUI for exploring calculation results",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTUI(cmd, inputPath, outputPath)
		},
	}
	cmd.Flags().StringVarP(&inputPath, "input", "i", "", "Input config file path")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output XLSX file path for export")
	_ = cmd.MarkFlagRequired("input")
	return cmd
}

func runTUI(cmd *cobra.Command, inputPath, outputPath string) error {
	plan, err := config.LoadFromYAML(inputPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	plan = config.ResolveDefaults(plan)

	valErrs := config.Validate(plan)
	if config.HasErrors(valErrs) {
		for _, e := range valErrs {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), e.String())
		}
		return fmt.Errorf("validation failed")
	}

	builder, err := profile.NewProfileBuilder(plan.Profile.Type)
	if err != nil {
		return fmt.Errorf("creating profile builder: %w", err)
	}
	steps, err := builder.BuildSteps(plan.Profile)
	if err != nil {
		return fmt.Errorf("building steps: %w", err)
	}

	var scenarioResults []engine.ScenarioResult
	for _, scenario := range plan.Scenarios {
		sr, err := calculateScenario(scenario, steps, plan.GlobalDefaults)
		if err != nil {
			return fmt.Errorf("calculating scenario %s: %w", scenario.Name, err)
		}
		scenarioResults = append(scenarioResults, sr)
	}

	timeline := profile.BuildTimeline(steps)

	results := engine.CalculationResults{
		Plan:            plan,
		Steps:           steps,
		Timeline:        timeline,
		ScenarioResults: scenarioResults,
	}

	return tui.Run(results, outputPath)
}

func writeYAMLTemplate(path string) error {
	tmpl := `# loadcalc input configuration
version: "1.0"

global:
  tool: jmeter                    # lre_pc | jmeter
  load_model: closed              # closed | open
  pacing_multiplier: 3.0
  deviation_tolerance: 2.5
  spike_participate: true
  generators_count: 3

scenarios:
  - name: "Main page"
    script_id: 1                    # required for lre_pc, ignored for jmeter
    target_intensity: 720000
    intensity_unit: ops_h
    max_script_time_ms: 1100

  - name: "Test page"
    script_id: 2
    target_intensity: 1500
    intensity_unit: ops_m
    max_script_time_ms: 1000
    pacing_multiplier: 4.0

  - name: "404 page"
    script_id: 3
    target_intensity: 90000
    intensity_unit: ops_h
    max_script_time_ms: 200
    background: true
    background_percent: 100

  - name: "API health check"
    script_id: 4
    target_intensity: 75
    intensity_unit: ops_h
    max_script_time_ms: 50
    load_model: open
    spike_participate: false

profile:
  type: max_search
  default_rampup_sec: 60
  default_impact_sec: 120
  default_stability_sec: 300
  default_rampdown_sec: 60
  start_percent: 50
  step_increment: 25
  num_steps: 5
  steps:
    - percent: 50
      stability_sec: 600
    - percent: 100
      stability_sec: 900
`
	return os.WriteFile(path, []byte(tmpl), 0o600)
}

func newCalculateCmd() *cobra.Command {
	var inputPath, outputPath, format, scenariosDir, csvDelimiter string
	var scenarioFiles []string
	var useTUI bool

	cmd := &cobra.Command{
		Use:   "calculate",
		Short: "Calculate load test parameters",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if useTUI {
				return runTUI(cmd, inputPath, outputPath)
			}
			return runCalculate(cmd, inputPath, outputPath, format, scenarioFiles, scenariosDir, csvDelimiter)
		},
	}
	cmd.Flags().StringVarP(&inputPath, "input", "i", "", "Input config file path")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path (required for xlsx format)")
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table|json|xlsx")
	cmd.Flags().BoolVar(&useTUI, "tui", false, "Launch interactive TUI after calculation")
	cmd.Flags().StringSliceVar(&scenarioFiles, "scenarios", nil, "Additional scenario files (CSV or XLSX)")
	cmd.Flags().StringVar(&scenariosDir, "scenarios-dir", "", "Directory with scenario files")
	cmd.Flags().StringVar(&csvDelimiter, "csv-delimiter", ";", "CSV delimiter character")
	_ = cmd.MarkFlagRequired("input")
	return cmd
}

func loadExtraScenarios(scenarioFiles []string, scenariosDir, csvDelimiter string) ([]config.Scenario, error) {
	delim := ';'
	if csvDelimiter != "" {
		delim = rune(csvDelimiter[0])
	}
	var all []config.Scenario
	for _, sf := range scenarioFiles {
		extra, err := config.LoadScenariosFromFile(sf, delim)
		if err != nil {
			return nil, fmt.Errorf("loading scenarios from %s: %w", sf, err)
		}
		all = append(all, extra...)
	}
	if scenariosDir != "" {
		extra, err := config.LoadScenariosFromDir(scenariosDir, delim)
		if err != nil {
			return nil, fmt.Errorf("loading scenarios from dir %s: %w", scenariosDir, err)
		}
		all = append(all, extra...)
	}
	return all, nil
}

func runCalculate(cmd *cobra.Command, inputPath, outputPath, format string, scenarioFiles []string, scenariosDir, csvDelimiter string) error {
	// 1. Load config
	plan, err := config.LoadFromYAML(inputPath)
	if err != nil {
		cmd.Println(fmt.Sprintf("[Error] Failed to load config: %v", err))
		return err
	}
	slog.Info("config loaded", "path", inputPath)

	// Load additional scenarios
	extra, err := loadExtraScenarios(scenarioFiles, scenariosDir, csvDelimiter)
	if err != nil {
		return err
	}
	plan.Scenarios = append(plan.Scenarios, extra...)

	// 2. Resolve defaults
	plan = config.ResolveDefaults(plan)

	// 3. Validate
	valErrs := config.Validate(plan)
	if len(valErrs) > 0 {
		for _, e := range valErrs {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), e.String())
		}
	}
	if config.HasErrors(valErrs) {
		slog.Error("validation failed")
		return fmt.Errorf("validation failed")
	}
	slog.Info("validation passed")

	// 4. Build steps
	builder, err := profile.NewProfileBuilder(plan.Profile.Type)
	if err != nil {
		return fmt.Errorf("creating profile builder: %w", err)
	}
	steps, err := builder.BuildSteps(plan.Profile)
	if err != nil {
		return fmt.Errorf("building steps: %w", err)
	}
	slog.Debug("steps built", "count", len(steps))

	// 5. Calculate per scenario
	var scenarioResults []engine.ScenarioResult
	for _, scenario := range plan.Scenarios {
		sr, err := calculateScenario(scenario, steps, plan.GlobalDefaults)
		if err != nil {
			return fmt.Errorf("calculating scenario %s: %w", scenario.Name, err)
		}
		scenarioResults = append(scenarioResults, sr)
	}

	// 6. Build timeline
	timeline := profile.BuildTimeline(steps)
	slog.Info("timeline built", "phases", len(timeline))

	results := engine.CalculationResults{
		Plan:            plan,
		Steps:           steps,
		Timeline:        timeline,
		ScenarioResults: scenarioResults,
	}

	// 7. Output
	switch strings.ToLower(format) {
	case "json":
		return writeJSON(cmd, results)
	case "table":
		return writeTable(cmd, results)
	case "xlsx":
		if outputPath == "" {
			return fmt.Errorf("--output (-o) is required for xlsx format")
		}
		w := &output.XLSXWriter{}
		return w.Write(results, outputPath)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

func calculateScenario(scenario config.Scenario, steps []profile.Step, globals config.GlobalDefaults) (engine.ScenarioResult, error) {
	loadModel := globals.LoadModel
	if scenario.LoadModel != nil {
		loadModel = *scenario.LoadModel
	}

	sr := engine.ScenarioResult{
		Scenario:     scenario,
		IsBackground: scenario.Background,
		IsOpenModel:  loadModel == config.LoadModelOpen,
	}

	// Normalize target to ops/sec
	targetRPS, err := units.NormalizeToOpsPerSec(scenario.TargetIntensity, scenario.IntensityUnit)
	if err != nil {
		return sr, err
	}

	if scenario.Background {
		// Background: calculate once at background_percent
		bgRPS := targetRPS * scenario.BackgroundPercent / 100
		var calc engine.Calculator
		calc, err = engine.NewCalculator(globals.Tool, loadModel, globals.GeneratorsCount)
		if err != nil {
			return sr, err
		}
		var result engine.Result
		result, err = calc.Calculate(scenario, bgRPS)
		if err != nil {
			return sr, err
		}
		sr.SingleResult = result
		// Also populate OptimizeResult with single step for consistency in output
		sr.OptimizeResult = engine.OptimizeResult{
			BestPacingMS:           result.PacingMS,
			BestOpsPerMinPerThread: result.OpsPerMinPerThread,
			MaxDeviationPct:        result.DeviationPercent,
			AllWithinTolerance:     true,
			StepResults: []engine.StepResult{
				{
					Step:         profile.Step{StepNumber: 1, PercentOfTarget: scenario.BackgroundPercent},
					TargetRPS:    result.TargetRPS,
					Threads:      result.Threads,
					ActualRPS:    result.ActualRPS,
					DeviationPct: result.DeviationPercent,
				},
			},
		}
		slog.Info("background calculation complete", "scenario", scenario.Name, "threads", result.Threads)
		return sr, nil
	}

	if loadModel == config.LoadModelOpen {
		// Open model: just unit conversion for each step
		var calc engine.Calculator
		calc, err = engine.NewCalculator(globals.Tool, loadModel, globals.GeneratorsCount)
		if err != nil {
			return sr, err
		}
		var stepResults []engine.StepResult
		for _, step := range steps {
			stepRPS := targetRPS * step.PercentOfTarget / 100
			var result engine.Result
			result, err = calc.Calculate(scenario, stepRPS)
			if err != nil {
				return sr, err
			}
			stepResults = append(stepResults, engine.StepResult{
				Step:         step,
				TargetRPS:    result.TargetRPS,
				Threads:      result.Threads,
				ActualRPS:    result.ActualRPS,
				DeviationPct: result.DeviationPercent,
			})
			// Store first result as SingleResult for output
			if sr.SingleResult == (engine.Result{}) {
				sr.SingleResult = result
			}
		}
		sr.OptimizeResult = engine.OptimizeResult{
			StepResults: stepResults,
		}
		slog.Info("open model calculation complete", "scenario", scenario.Name)
		return sr, nil
	}

	// Closed model: run optimizer
	opt := &engine.Optimizer{}
	optResult, err := opt.Optimize(scenario, steps, globals.Tool, loadModel, globals.GeneratorsCount)
	if err != nil {
		return sr, err
	}
	sr.OptimizeResult = optResult
	slog.Info("optimization complete",
		"scenario", scenario.Name,
		"best_pacing_ms", optResult.BestPacingMS,
		"max_deviation", optResult.MaxDeviationPct,
		"within_tolerance", optResult.AllWithinTolerance,
	)
	if optResult.Warning != "" {
		slog.Warn(optResult.Warning, "scenario", scenario.Name)
	}
	return sr, nil
}

// JSON output types.
type jsonOutput struct {
	Scenarios []jsonScenario `json:"scenarios"`
	Timeline  []jsonPhase    `json:"timeline"`
}

type jsonScenario struct {
	Name         string     `json:"name"`
	LoadModel    string     `json:"load_model"`
	Steps        []jsonStep `json:"steps,omitempty"`
	ScriptID     int        `json:"script_id,omitempty"`
	BestPacingMS float64    `json:"best_pacing_ms,omitempty"`
	MaxDeviation float64    `json:"max_deviation_pct,omitempty"`
	IsBackground bool       `json:"is_background"`
}

type jsonStep struct {
	StepNumber int     `json:"step_number"`
	Percent    float64 `json:"percent"`
	TargetRPS  float64 `json:"target_rps"`
	Threads    int     `json:"threads,omitempty"`
	ActualRPS  float64 `json:"actual_rps"`
	Deviation  float64 `json:"deviation_pct"`
}

type jsonPhase struct {
	Name     string  `json:"name"`
	StartSec int     `json:"start_sec"`
	Duration int     `json:"duration_sec"`
	EndSec   int     `json:"end_sec"`
	Percent  float64 `json:"percent"`
}

func writeJSON(cmd *cobra.Command, results engine.CalculationResults) error {
	var out jsonOutput

	for _, sr := range results.ScenarioResults {
		loadModel := string(results.Plan.GlobalDefaults.LoadModel)
		if sr.Scenario.LoadModel != nil {
			loadModel = string(*sr.Scenario.LoadModel)
		}

		js := jsonScenario{
			Name:         sr.Scenario.Name,
			ScriptID:     sr.Scenario.ScriptID,
			LoadModel:    loadModel,
			IsBackground: sr.IsBackground,
			BestPacingMS: sr.OptimizeResult.BestPacingMS,
			MaxDeviation: sr.OptimizeResult.MaxDeviationPct,
		}

		for _, step := range sr.OptimizeResult.StepResults {
			js.Steps = append(js.Steps, jsonStep{
				StepNumber: step.Step.StepNumber,
				Percent:    step.Step.PercentOfTarget,
				TargetRPS:  step.TargetRPS,
				Threads:    step.Threads,
				ActualRPS:  step.ActualRPS,
				Deviation:  step.DeviationPct,
			})
		}

		out.Scenarios = append(out.Scenarios, js)
	}

	for _, phase := range results.Timeline {
		out.Timeline = append(out.Timeline, jsonPhase{
			Name:     phase.PhaseName,
			StartSec: phase.StartTimeSec,
			Duration: phase.DurationSec,
			EndSec:   phase.EndTimeSec,
			Percent:  phase.PercentOfTarget,
		})
	}

	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	cmd.Print(string(b))
	return nil
}

func newJMXCmd() *cobra.Command {
	jmxCmd := &cobra.Command{
		Use:   "jmx",
		Short: "JMeter JMX generation commands",
	}
	jmxCmd.AddCommand(newJMXGenerateCmd())
	jmxCmd.AddCommand(newJMXInjectCmd())
	return jmxCmd
}

func newJMXGenerateCmd() *cobra.Command {
	var inputPath, outputPath, scenariosDir, csvDelimiter string
	var scenarioFiles []string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a complete .jmx file from config",
		RunE: func(cmd *cobra.Command, _ []string) error {
			results, err := runPipeline(inputPath, scenarioFiles, scenariosDir, csvDelimiter)
			if err != nil {
				return err
			}
			data, err := integration.GenerateJMX(results)
			if err != nil {
				return fmt.Errorf("generating JMX: %w", err)
			}
			if err := os.WriteFile(outputPath, data, 0o600); err != nil {
				return fmt.Errorf("writing JMX: %w", err)
			}
			slog.Info("JMX generated", "path", outputPath)
			cmd.Println(fmt.Sprintf("JMX written to %s", outputPath))
			return nil
		},
	}
	cmd.Flags().StringVarP(&inputPath, "input", "i", "", "Input config file path")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output .jmx file path")
	cmd.Flags().StringSliceVar(&scenarioFiles, "scenarios", nil, "Additional scenario files")
	cmd.Flags().StringVar(&scenariosDir, "scenarios-dir", "", "Directory with scenario files")
	cmd.Flags().StringVar(&csvDelimiter, "csv-delimiter", ";", "CSV delimiter character")
	_ = cmd.MarkFlagRequired("input")
	_ = cmd.MarkFlagRequired("output")
	return cmd
}

func newJMXInjectCmd() *cobra.Command {
	var inputPath, outputPath, jmxTemplate, scenariosDir, csvDelimiter string
	var scenarioFiles []string
	var updateExisting bool

	cmd := &cobra.Command{
		Use:   "inject",
		Short: "Inject ThreadGroups into an existing .jmx template",
		RunE: func(cmd *cobra.Command, _ []string) error {
			results, err := runPipeline(inputPath, scenarioFiles, scenariosDir, csvDelimiter)
			if err != nil {
				return err
			}
			var data []byte
			if updateExisting {
				data, err = integration.UpdateExistingJMX(jmxTemplate, results)
			} else {
				data, err = integration.InjectIntoJMX(jmxTemplate, results)
			}
			if err != nil {
				return fmt.Errorf("injecting JMX: %w", err)
			}
			if err := os.WriteFile(outputPath, data, 0o600); err != nil {
				return fmt.Errorf("writing JMX: %w", err)
			}
			slog.Info("JMX injected", "path", outputPath)
			cmd.Println(fmt.Sprintf("JMX written to %s", outputPath))
			return nil
		},
	}
	cmd.Flags().StringVarP(&inputPath, "input", "i", "", "Input config file path")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output .jmx file path")
	cmd.Flags().StringVar(&jmxTemplate, "jmx-template", "", "Existing .jmx template file")
	cmd.Flags().BoolVar(&updateExisting, "update-existing", false, "Update existing ThreadGroups in-place")
	cmd.Flags().StringSliceVar(&scenarioFiles, "scenarios", nil, "Additional scenario files")
	cmd.Flags().StringVar(&scenariosDir, "scenarios-dir", "", "Directory with scenario files")
	cmd.Flags().StringVar(&csvDelimiter, "csv-delimiter", ";", "CSV delimiter character")
	_ = cmd.MarkFlagRequired("input")
	_ = cmd.MarkFlagRequired("output")
	_ = cmd.MarkFlagRequired("jmx-template")
	return cmd
}

// runPipeline runs the full calculation pipeline and returns results.
func runPipeline(inputPath string, scenarioFiles []string, scenariosDir, csvDelimiter string) (engine.CalculationResults, error) {
	plan, err := config.LoadFromYAML(inputPath)
	if err != nil {
		return engine.CalculationResults{}, fmt.Errorf("loading config: %w", err)
	}

	delim := ';'
	if csvDelimiter != "" {
		delim = rune(csvDelimiter[0])
	}
	for _, sf := range scenarioFiles {
		var extra []config.Scenario
		extra, err = config.LoadScenariosFromFile(sf, delim)
		if err != nil {
			return engine.CalculationResults{}, fmt.Errorf("loading scenarios from %s: %w", sf, err)
		}
		plan.Scenarios = append(plan.Scenarios, extra...)
	}
	if scenariosDir != "" {
		var extra []config.Scenario
		extra, err = config.LoadScenariosFromDir(scenariosDir, delim)
		if err != nil {
			return engine.CalculationResults{}, fmt.Errorf("loading scenarios from dir %s: %w", scenariosDir, err)
		}
		plan.Scenarios = append(plan.Scenarios, extra...)
	}

	plan = config.ResolveDefaults(plan)
	valErrs := config.Validate(plan)
	if config.HasErrors(valErrs) {
		return engine.CalculationResults{}, fmt.Errorf("validation failed")
	}

	builder, err := profile.NewProfileBuilder(plan.Profile.Type)
	if err != nil {
		return engine.CalculationResults{}, fmt.Errorf("creating profile builder: %w", err)
	}
	steps, err := builder.BuildSteps(plan.Profile)
	if err != nil {
		return engine.CalculationResults{}, fmt.Errorf("building steps: %w", err)
	}

	var scenarioResults []engine.ScenarioResult
	for _, scenario := range plan.Scenarios {
		sr, err := calculateScenario(scenario, steps, plan.GlobalDefaults)
		if err != nil {
			return engine.CalculationResults{}, fmt.Errorf("calculating scenario %s: %w", scenario.Name, err)
		}
		scenarioResults = append(scenarioResults, sr)
	}

	timeline := profile.BuildTimeline(steps)

	return engine.CalculationResults{
		Plan:            plan,
		ScenarioResults: scenarioResults,
		Steps:           steps,
		Timeline:        timeline,
	}, nil
}

func newDiffCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "diff <old.yaml> <new.yaml>",
		Short: "Compare two configuration files and show differences",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			oldPlan, err := config.LoadFromYAML(args[0])
			if err != nil {
				return fmt.Errorf("loading old config: %w", err)
			}
			oldPlan = config.ResolveDefaults(oldPlan)

			newPlan, err := config.LoadFromYAML(args[1])
			if err != nil {
				return fmt.Errorf("loading new config: %w", err)
			}
			newPlan = config.ResolveDefaults(newPlan)

			result := diff.ComparePlans(oldPlan, newPlan)

			switch strings.ToLower(format) {
			case "json":
				data, err := diff.FormatJSON(result)
				if err != nil {
					return fmt.Errorf("formatting JSON: %w", err)
				}
				cmd.Print(string(data))
			case "table":
				cmd.Print(diff.FormatTable(result))
			default:
				return fmt.Errorf("unknown format: %s (use table or json)", format)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "Output format: table|json")
	return cmd
}

func writeTable(cmd *cobra.Command, results engine.CalculationResults) error {
	cmd.Println(fmt.Sprintf("%-20s %-8s %10s %10s %10s %10s",
		"Name", "Model", "Threads", "Pacing(ms)", "MaxDev%", "Status"))
	cmd.Println(strings.Repeat("-", 70))

	for _, sr := range results.ScenarioResults {
		loadModel := string(results.Plan.GlobalDefaults.LoadModel)
		if sr.Scenario.LoadModel != nil {
			loadModel = string(*sr.Scenario.LoadModel)
		}

		threads := "-"
		pacing := "-"
		maxDev := "-"
		status := "OK"

		if len(sr.OptimizeResult.StepResults) > 0 {
			threads = fmt.Sprintf("%d", sr.OptimizeResult.StepResults[0].Threads)
			pacing = fmt.Sprintf("%.1f", sr.OptimizeResult.BestPacingMS)
			maxDev = fmt.Sprintf("%.2f", sr.OptimizeResult.MaxDeviationPct)
			if !sr.OptimizeResult.AllWithinTolerance {
				status = "WARN"
			}
		}

		bg := ""
		if sr.IsBackground {
			bg = " [BG]"
		}

		cmd.Println(fmt.Sprintf("%-20s %-8s %10s %10s %10s %10s%s",
			sr.Scenario.Name, loadModel, threads, pacing, maxDev, status, bg))
	}

	// Print step details
	cmd.Println("")
	cmd.Println("Step Details:")
	cmd.Println(fmt.Sprintf("%-20s %8s %10s %8s %10s %10s",
		"Scenario", "Step%", "TargetRPS", "Threads", "ActualRPS", "Dev%"))
	cmd.Println(strings.Repeat("-", 70))

	for _, sr := range results.ScenarioResults {
		for _, step := range sr.OptimizeResult.StepResults {
			cmd.Println(fmt.Sprintf("%-20s %7.0f%% %10.4f %8d %10.4f %9.2f%%",
				sr.Scenario.Name, step.Step.PercentOfTarget, step.TargetRPS,
				step.Threads, step.ActualRPS, step.DeviationPct))
		}
	}

	return nil
}

// runPipelineFromPlan runs calculation from an already-resolved TestPlan.
func runPipelineFromPlan(plan *config.TestPlan) (engine.CalculationResults, error) {
	builder, err := profile.NewProfileBuilder(plan.Profile.Type)
	if err != nil {
		return engine.CalculationResults{}, fmt.Errorf("creating profile builder: %w", err)
	}
	steps, err := builder.BuildSteps(plan.Profile)
	if err != nil {
		return engine.CalculationResults{}, fmt.Errorf("building steps: %w", err)
	}

	var scenarioResults []engine.ScenarioResult
	for _, scenario := range plan.Scenarios {
		sr, err := calculateScenario(scenario, steps, plan.GlobalDefaults)
		if err != nil {
			return engine.CalculationResults{}, fmt.Errorf("calculating scenario %s: %w", scenario.Name, err)
		}
		scenarioResults = append(scenarioResults, sr)
	}

	timeline := profile.BuildTimeline(steps)

	return engine.CalculationResults{
		Plan:            plan,
		ScenarioResults: scenarioResults,
		Steps:           steps,
		Timeline:        timeline,
	}, nil
}
