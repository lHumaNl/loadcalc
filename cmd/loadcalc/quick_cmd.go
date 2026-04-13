package main

import (
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"

	"loadcalc/internal/config"
	"loadcalc/internal/engine"
	"loadcalc/internal/profile"
	"loadcalc/pkg/units"

	"github.com/spf13/cobra"
)

func newQuickCmd() *cobra.Command {
	var (
		multiplier      float64
		generators      int
		model           string
		unit            string
		tolerance       float64
		steps           string
		rampup          int
		multiplierRange float64
	)

	cmd := &cobra.Command{
		Use:   "quick <intensity> <script_time_ms> <tool>",
		Short: "Quick single-scenario calculation without a config file",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			intensity, err := strconv.ParseFloat(args[0], 64)
			if err != nil {
				return fmt.Errorf("invalid intensity: %w", err)
			}

			scriptTimeMs, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid script_time_ms: %w", err)
			}

			toolStr := args[2]
			var tool config.Tool
			switch toolStr {
			case "jmeter":
				tool = config.ToolJMeter
			case "lre_pc":
				tool = config.ToolLREPC
			default:
				return fmt.Errorf("unknown tool: %s (use jmeter or lre_pc)", toolStr)
			}

			if tool == config.ToolLREPC {
				if cmd.Flags().Changed("generators") {
					slog.Info("--generators flag ignored for lre_pc tool, using 1")
				}
				generators = 1
			}

			loadModel := config.LoadModel(model)
			if loadModel != config.LoadModelClosed && loadModel != config.LoadModelOpen {
				return fmt.Errorf("unknown model: %s (use closed or open)", model)
			}

			intensityUnit := units.IntensityUnit(unit)

			scenario := config.Scenario{
				Name:               "quick",
				TargetIntensity:    intensity,
				IntensityUnit:      intensityUnit,
				MaxScriptTimeMs:    scriptTimeMs,
				PacingMultiplier:   &multiplier,
				DeviationTolerance: &tolerance,
			}

			// Display intensity in the given unit
			unitLabel := formatUnitLabel(intensityUnit)
			intensityDisplay := formatNumber(intensity)

			if steps == "" {
				return runQuickSingle(cmd, scenario, tool, loadModel, generators, intensityDisplay, unitLabel, multiplier)
			}
			return runQuickMultiStep(cmd, scenario, tool, loadModel, generators, intensityDisplay, unitLabel, multiplier, steps, rampup, multiplierRange)
		},
	}

	cmd.Flags().Float64VarP(&multiplier, "multiplier", "m", 3.0, "Pacing multiplier")
	cmd.Flags().IntVarP(&generators, "generators", "g", 1, "JMeter generator count (ignored for lre_pc)")
	cmd.Flags().StringVar(&model, "model", "closed", "Load model: closed|open")
	cmd.Flags().StringVarP(&unit, "unit", "u", "ops_h", "Intensity unit: ops_h|ops_m|ops_s")
	cmd.Flags().Float64Var(&tolerance, "tolerance", 2.5, "Deviation tolerance %")
	cmd.Flags().StringVarP(&steps, "steps", "s", "", "Comma-separated step percentages (e.g., 50,75,100)")
	cmd.Flags().IntVar(&rampup, "rampup", 60, "Ramp-up seconds per step (used with --steps for LRE PC)")
	cmd.Flags().Float64Var(&multiplierRange, "range", 0.5, "Multiplier search range ± (0 = default ±25% of base pacing)")

	return cmd
}

func runQuickSingle(cmd *cobra.Command, scenario config.Scenario, tool config.Tool, loadModel config.LoadModel, generators int, intensityDisplay, unitLabel string, multiplier float64) error {
	targetRPS, err := units.NormalizeToOpsPerSec(scenario.TargetIntensity, scenario.IntensityUnit)
	if err != nil {
		return err
	}

	calc, err := engine.NewCalculator(tool, loadModel, generators)
	if err != nil {
		return err
	}

	result, err := calc.Calculate(scenario, targetRPS)
	if err != nil {
		return err
	}

	// Header
	if loadModel == config.LoadModelOpen {
		cmd.Println(fmt.Sprintf("  Scenario: %s %s, script %dms", intensityDisplay, unitLabel, scenario.MaxScriptTimeMs))
	} else {
		cmd.Println(fmt.Sprintf("  Scenario: %s %s, script %dms, pacing ×%.1f", intensityDisplay, unitLabel, scenario.MaxScriptTimeMs, multiplier))
	}
	cmd.Println("")

	toolLabel := formatToolLabel(tool, loadModel, generators)
	cmd.Println(fmt.Sprintf("  Tool:       %s", toolLabel))

	if loadModel == config.LoadModelOpen {
		ratePerGen := targetRPS / float64(max(generators, 1))
		cmd.Println(fmt.Sprintf("  Rate:       %.2f ops/sec (per generator)", ratePerGen))
	} else {
		cmd.Println(fmt.Sprintf("  Threads:    %s", formatNumber(float64(result.Threads))))
		cmd.Println(fmt.Sprintf("  Pacing:     %s ms", formatNumber(result.PacingMS)))
		if tool == config.ToolJMeter {
			cmd.Println(fmt.Sprintf("  CTT:        %.2f ops/min/thread", result.OpsPerMinPerThread))
		}
	}
	cmd.Println(fmt.Sprintf("  Deviation:  %.2f%%  %s", result.DeviationPercent, deviationSymbol(result.DeviationPercent, *scenario.DeviationTolerance)))

	return nil
}

func runQuickMultiStep(cmd *cobra.Command, scenario config.Scenario, tool config.Tool, loadModel config.LoadModel, generators int, intensityDisplay, unitLabel string, multiplier float64, stepsStr string, rampupSec int, multiplierRange float64) error {
	// Parse step percentages
	parts := strings.Split(stepsStr, ",")
	var stepList []profile.Step
	for i, p := range parts {
		pct, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return fmt.Errorf("invalid step percentage %q: %w", p, err)
		}
		stepList = append(stepList, profile.Step{
			StepNumber:      i + 1,
			PercentOfTarget: pct,
		})
	}

	// Header
	cmd.Println(fmt.Sprintf("  Scenario: %s %s, script %dms, pacing ×%.1f", intensityDisplay, unitLabel, scenario.MaxScriptTimeMs, multiplier))
	cmd.Println("")

	toolLabel := formatToolLabel(tool, loadModel, generators)
	cmd.Println(fmt.Sprintf("  Tool:       %s", toolLabel))

	if loadModel == config.LoadModelOpen {
		return fmt.Errorf("multi-step mode is not supported for open load model")
	}

	// Run optimizer
	opt := &engine.Optimizer{MultiplierRange: multiplierRange}
	optResult, err := opt.Optimize(scenario, stepList, tool, loadModel, generators)
	if err != nil {
		return err
	}

	cmd.Println(fmt.Sprintf("  Pacing:     %s ms (optimized)", formatNumber(optResult.BestPacingMS)))
	if tool == config.ToolJMeter {
		cmd.Println(fmt.Sprintf("  CTT:        %.2f ops/min/thread", optResult.BestOpsPerMinPerThread))
	}
	cmd.Println("")

	if tool == config.ToolJMeter {
		printJMeterStepTable(cmd, optResult, scenario.IntensityUnit, generators)
	} else {
		printLREPCStepTable(cmd, optResult, scenario.IntensityUnit, rampupSec)
	}

	return nil
}

func printJMeterStepTable(cmd *cobra.Command, opt engine.OptimizeResult, intensityUnit units.IntensityUnit, generators int) {
	unitLabel := formatUnitLabel(intensityUnit)
	cmd.Println(fmt.Sprintf("  Step   %%    Threads  Threads/Gen  Actual %s   Dev %%", unitLabel))
	cmd.Println("  " + strings.Repeat("\u2500", 59))

	for _, sr := range opt.StepResults {
		actualDisplay := units.ConvertFromOpsPerSec(sr.ActualRPS*float64(generators), intensityUnit)
		threadsTotal := sr.Threads * generators
		cmd.Println(fmt.Sprintf("  %3d  %3.0f%%   %6d    %6d     %11s   %5.2f%%  %s",
			sr.Step.StepNumber,
			sr.Step.PercentOfTarget,
			threadsTotal,
			sr.Threads,
			formatNumber(actualDisplay),
			sr.DeviationPct,
			deviationSymbol(sr.DeviationPct, 2.5),
		))
	}
}

func printLREPCStepTable(cmd *cobra.Command, opt engine.OptimizeResult, _ units.IntensityUnit, rampupSec int) {
	cmd.Println("  Step   %    Vusers  Delta  Batch  Every(s)  Rampup   Dev %")
	cmd.Println("  " + strings.Repeat("\u2500", 60))

	prevThreads := 0
	for _, sr := range opt.StepResults {
		delta := sr.Threads - prevThreads
		ramp := engine.CalculateRampUp(delta, rampupSec)

		everyStr := fmt.Sprintf("%ds", ramp.IntervalSec)
		rampStr := fmt.Sprintf("%ds", ramp.ActualSec)
		deltaStr := fmt.Sprintf("+%d", delta)

		cmd.Println(fmt.Sprintf("  %3d  %3.0f%%   %6d  %5s  %5d    %5s    %5s   %5.2f%%  %s",
			sr.Step.StepNumber,
			sr.Step.PercentOfTarget,
			sr.Threads,
			deltaStr,
			ramp.BatchSize,
			everyStr,
			rampStr,
			sr.DeviationPct,
			deviationSymbol(sr.DeviationPct, 2.5),
		))
		prevThreads = sr.Threads
	}
}

func formatToolLabel(tool config.Tool, loadModel config.LoadModel, generators int) string {
	switch tool {
	case config.ToolJMeter:
		genStr := fmt.Sprintf("%d generator", generators)
		if generators != 1 {
			genStr += "s"
		}
		return fmt.Sprintf("JMeter (%s model, %s)", loadModel, genStr)
	case config.ToolLREPC:
		return fmt.Sprintf("LRE PC (%s model)", loadModel)
	default:
		return string(tool)
	}
}

func formatUnitLabel(u units.IntensityUnit) string {
	switch u {
	case units.OpsPerHour:
		return "ops/h"
	case units.OpsPerMinute:
		return "ops/m"
	case units.OpsPerSecond:
		return "ops/s"
	default:
		return string(u)
	}
}

func deviationSymbol(dev, tolerance float64) string {
	if dev < 1.0 {
		return "\u2713"
	}
	if dev <= tolerance {
		return "\u26A0"
	}
	return "\u2717"
}

func formatNumber(v float64) string {
	// Round to avoid floating point artifacts
	rounded := math.Round(v*100) / 100

	isInt := rounded == math.Trunc(rounded)
	var s string
	if isInt {
		s = strconv.FormatInt(int64(rounded), 10)
	} else {
		s = strconv.FormatFloat(rounded, 'f', -1, 64)
	}

	// Add comma separators to the integer part
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	negative := false
	if strings.HasPrefix(intPart, "-") {
		negative = true
		intPart = intPart[1:]
	}

	if len(intPart) > 3 {
		var result []rune
		for i, c := range intPart {
			if i > 0 && (len(intPart)-i)%3 == 0 {
				result = append(result, ',')
			}
			result = append(result, c)
		}
		intPart = string(result)
	}

	if negative {
		intPart = "-" + intPart
	}
	if len(parts) == 2 {
		return intPart + "." + parts[1]
	}
	return intPart
}
