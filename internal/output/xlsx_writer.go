package output

import (
	"fmt"

	"github.com/xuri/excelize/v2"

	"loadcalc/internal/config"
	"loadcalc/internal/engine"
)

// XLSXWriter writes calculation results to an XLSX file.
type XLSXWriter struct{}

// Write generates the XLSX workbook and saves it to dest.
func (w *XLSXWriter) Write(results engine.CalculationResults, dest string) error {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	// Create sheets (Sheet1 is created by default, rename it)
	if err := f.SetSheetName("Sheet1", "Summary"); err != nil {
		return fmt.Errorf("renaming sheet: %w", err)
	}
	if _, err := f.NewSheet("Steps Detail"); err != nil {
		return fmt.Errorf("creating Steps Detail sheet: %w", err)
	}
	if _, err := f.NewSheet("Timeline"); err != nil {
		return fmt.Errorf("creating Timeline sheet: %w", err)
	}
	if _, err := f.NewSheet("Input Parameters"); err != nil {
		return fmt.Errorf("creating Input Parameters sheet: %w", err)
	}
	if results.Plan.GlobalDefaults.Tool == config.ToolJMeter {
		if _, err := f.NewSheet("JMeter Config"); err != nil {
			return fmt.Errorf("creating JMeter Config sheet: %w", err)
		}
	}

	// Styles
	boldStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	redBgStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"FFCCCC"}, Pattern: 1},
	})
	pctFmt, _ := f.NewStyle(&excelize.Style{
		NumFmt: 10, // 0.00%
	})
	numFmt2, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: strPtr("0.00"),
	})
	_ = pctFmt
	_ = numFmt2

	// Sheet 1: Summary
	if err := w.writeSummary(f, results, boldStyle, redBgStyle); err != nil {
		return err
	}

	// Sheet 2: Steps Detail
	if err := w.writeStepsDetail(f, results, boldStyle, redBgStyle); err != nil {
		return err
	}

	// Sheet 3: Timeline
	if err := w.writeTimeline(f, results, boldStyle); err != nil {
		return err
	}

	// Sheet 4: Input Parameters
	if err := w.writeInputParameters(f, results, boldStyle); err != nil {
		return err
	}

	// Sheet 5: JMeter Config
	if results.Plan.GlobalDefaults.Tool == config.ToolJMeter {
		if err := w.writeJMeterConfig(f, results, boldStyle); err != nil {
			return err
		}
	}

	return f.SaveAs(dest)
}

func setCellValue(f *excelize.File, sheet, cell string, value interface{}) error {
	return f.SetCellValue(sheet, cell, value)
}

func setCellStyle(f *excelize.File, sheet, cell string, styleID int) error {
	return f.SetCellStyle(sheet, cell, cell, styleID)
}

func summaryRowValues(sr engine.ScenarioResult, globals config.GlobalDefaults) (vals []interface{}, maxDev float64) {
	sc := sr.Scenario

	loadModel := string(globals.LoadModel)
	if sc.LoadModel != nil {
		loadModel = string(*sc.LoadModel)
	}

	spikeP := globals.SpikeParticipate
	if sc.SpikeParticipate != nil {
		spikeP = *sc.SpikeParticipate
	}

	baseThreads := 0
	maxDev = sr.OptimizeResult.MaxDeviationPct
	for _, stepRes := range sr.OptimizeResult.StepResults {
		if stepRes.Step.PercentOfTarget == 100 {
			baseThreads = stepRes.Threads
		}
	}
	if baseThreads == 0 && len(sr.OptimizeResult.StepResults) > 0 {
		baseThreads = sr.OptimizeResult.StepResults[0].Threads
	}

	vals = []interface{}{
		sc.Name,
		sc.ScriptID,
		loadModel,
		FormatIntensity(sc.TargetIntensity, sc.IntensityUnit),
		sr.OptimizeResult.BestPacingMS,
		sr.OptimizeResult.BestOpsPerMinPerThread,
		baseThreads,
		boolYesNo(sr.IsBackground),
		boolYesNo(spikeP),
		maxDev,
	}
	return vals, maxDev
}

func (w *XLSXWriter) writeSummary(f *excelize.File, results engine.CalculationResults, boldStyle, redBgStyle int) error {
	sheet := "Summary"
	headers := []string{
		"Scenario Name", "Script ID", "Load Model", "Target Intensity",
		"Pacing (ms)", "Ops/min/thread", "Base Threads", "Is Background",
		"Spike Participate", "Max Deviation %",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := setCellValue(f, sheet, cell, h); err != nil {
			return err
		}
		if err := setCellStyle(f, sheet, cell, boldStyle); err != nil {
			return err
		}
	}

	tolerance := results.Plan.GlobalDefaults.DeviationTolerance

	for idx, sr := range results.ScenarioResults {
		row := idx + 2

		vals, maxDev := summaryRowValues(sr, results.Plan.GlobalDefaults)

		for i, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			if err := setCellValue(f, sheet, cell, v); err != nil {
				return err
			}
		}

		// Red background if max deviation exceeds tolerance
		if maxDev > tolerance {
			for i := 0; i < len(headers); i++ {
				cell, _ := excelize.CoordinatesToCellName(i+1, row)
				if err := setCellStyle(f, sheet, cell, redBgStyle); err != nil {
					return err
				}
			}
		}
	}

	return autoWidth(f, sheet, headers)
}

func (w *XLSXWriter) writeStepsDetail(f *excelize.File, results engine.CalculationResults, boldStyle, redBgStyle int) error {
	sheet := "Steps Detail"
	headers := []string{
		"Step #", "Step %", "Scenario Name", "Target RPS", "Threads",
		"Actual RPS", "Deviation %", "Rampup (s)", "Impact (s)",
		"Stability (s)", "Rampdown (s)",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := setCellValue(f, sheet, cell, h); err != nil {
			return err
		}
		if err := setCellStyle(f, sheet, cell, boldStyle); err != nil {
			return err
		}
	}

	tolerance := results.Plan.GlobalDefaults.DeviationTolerance

	row := 2
	for _, sr := range results.ScenarioResults {
		for _, stepRes := range sr.OptimizeResult.StepResults {
			vals := []interface{}{
				stepRes.Step.StepNumber,
				stepRes.Step.PercentOfTarget,
				sr.Scenario.Name,
				stepRes.TargetRPS,
				stepRes.Threads,
				stepRes.ActualRPS,
				stepRes.DeviationPct,
				stepRes.Step.RampupSec,
				stepRes.Step.ImpactSec,
				stepRes.Step.StabilitySec,
				stepRes.Step.RampdownSec,
			}
			for i, v := range vals {
				cell, _ := excelize.CoordinatesToCellName(i+1, row)
				if err := setCellValue(f, sheet, cell, v); err != nil {
					return err
				}
			}

			// Conditional formatting on deviation column
			if stepRes.DeviationPct > tolerance {
				devCell, _ := excelize.CoordinatesToCellName(7, row)
				if err := setCellStyle(f, sheet, devCell, redBgStyle); err != nil {
					return err
				}
			}
			row++
		}
	}

	return autoWidth(f, sheet, headers)
}

func (w *XLSXWriter) writeTimeline(f *excelize.File, results engine.CalculationResults, boldStyle int) error {
	sheet := "Timeline"

	// Build per-scenario column headers
	scenarioNames := make([]string, 0, len(results.ScenarioResults))
	for _, sr := range results.ScenarioResults {
		scenarioNames = append(scenarioNames, sr.Scenario.Name)
	}

	headers := make([]string, 0, 5+len(scenarioNames))
	headers = append(headers, "Phase", "Start Time (s)", "Duration (s)", "End Time (s)", "Total Threads")
	for _, name := range scenarioNames {
		headers = append(headers, name+" Threads")
	}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := setCellValue(f, sheet, cell, h); err != nil {
			return err
		}
		if err := setCellStyle(f, sheet, cell, boldStyle); err != nil {
			return err
		}
	}

	for rowIdx, phase := range results.Timeline {
		row := rowIdx + 2

		// Calculate per-scenario threads for this phase
		totalThreads := 0
		scenarioThreads := make([]int, len(results.ScenarioResults))
		for i, sr := range results.ScenarioResults {
			threads := 0
			for _, stepRes := range sr.OptimizeResult.StepResults {
				if stepRes.Step.PercentOfTarget == phase.PercentOfTarget {
					threads = stepRes.Threads
					break
				}
			}
			scenarioThreads[i] = threads
			totalThreads += threads
		}

		vals := make([]interface{}, 0, 5+len(scenarioThreads))
		vals = append(vals,
			phase.PhaseName,
			phase.StartTimeSec,
			phase.DurationSec,
			phase.EndTimeSec,
			totalThreads,
		)
		for _, t := range scenarioThreads {
			vals = append(vals, t)
		}

		for i, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			if err := setCellValue(f, sheet, cell, v); err != nil {
				return err
			}
		}
	}

	// Add line chart for total threads over time
	if len(results.Timeline) > 0 {
		lastRow := len(results.Timeline) + 1
		catRange := fmt.Sprintf("Timeline!$B$2:$B$%d", lastRow)
		valRange := fmt.Sprintf("Timeline!$E$2:$E$%d", lastRow)

		if err := f.AddChart(sheet, "H2", &excelize.Chart{
			Type: excelize.Line,
			Series: []excelize.ChartSeries{
				{
					Name:       "Total Threads",
					Categories: catRange,
					Values:     valRange,
				},
			},
			Title: []excelize.RichTextRun{
				{Text: "Thread Count Over Time"},
			},
			XAxis:     excelize.ChartAxis{Title: []excelize.RichTextRun{{Text: "Time (s)"}}},
			YAxis:     excelize.ChartAxis{Title: []excelize.RichTextRun{{Text: "Threads"}}},
			Dimension: excelize.ChartDimension{Width: 600, Height: 400},
		}); err != nil {
			return fmt.Errorf("add chart: %w", err)
		}
	}

	return autoWidth(f, sheet, headers)
}

func (w *XLSXWriter) writeInputParameters(f *excelize.File, results engine.CalculationResults, boldStyle int) error {
	sheet := "Input Parameters"

	headers := []string{"Parameter", "Value"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := setCellValue(f, sheet, cell, h); err != nil {
			return err
		}
		if err := setCellStyle(f, sheet, cell, boldStyle); err != nil {
			return err
		}
	}

	g := results.Plan.GlobalDefaults
	params := make([][]interface{}, 0, 12+5*len(results.Plan.Scenarios))
	params = append(params,
		[]interface{}{"Tool", string(g.Tool)},
		[]interface{}{"Load Model", string(g.LoadModel)},
		[]interface{}{"Pacing Multiplier", g.PacingMultiplier},
		[]interface{}{"Deviation Tolerance %", g.DeviationTolerance},
		[]interface{}{"Spike Participate", boolYesNo(g.SpikeParticipate)},
		[]interface{}{"Generators Count", g.GeneratorsCount},
		[]interface{}{"Profile Type", string(results.Plan.Profile.Type)},
		[]interface{}{"Default Rampup (s)", results.Plan.Profile.DefaultRampupSec},
		[]interface{}{"Default Impact (s)", results.Plan.Profile.DefaultImpactSec},
		[]interface{}{"Default Stability (s)", results.Plan.Profile.DefaultStabilitySec},
		[]interface{}{"Default Rampdown (s)", results.Plan.Profile.DefaultRampdownSec},
		[]interface{}{"Output Format", results.Plan.OutputFormat},
	)

	// Add scenario summaries
	for _, sc := range results.Plan.Scenarios {
		params = append(params,
			[]interface{}{"", ""},
			[]interface{}{fmt.Sprintf("Scenario: %s", sc.Name), fmt.Sprintf("Script ID: %d", sc.ScriptID)},
			[]interface{}{"  Target Intensity", FormatIntensity(sc.TargetIntensity, sc.IntensityUnit)},
			[]interface{}{"  Max Script Time (ms)", sc.MaxScriptTimeMs},
			[]interface{}{"  Background", boolYesNo(sc.Background)},
		)
	}

	for i, p := range params {
		row := i + 2
		for j, v := range p {
			cell, _ := excelize.CoordinatesToCellName(j+1, row)
			if err := setCellValue(f, sheet, cell, v); err != nil {
				return err
			}
		}
	}

	return autoWidth(f, sheet, headers)
}

func jmeterConfigRowValues(sr engine.ScenarioResult, generators int) []interface{} {
	tgType := "ThreadGroup"
	if sr.IsOpenModel {
		tgType = "FreeFormArrivalsThreadGroup"
	}

	baseThreads := 0
	for _, stepRes := range sr.OptimizeResult.StepResults {
		if stepRes.Step.PercentOfTarget == 100 {
			baseThreads = stepRes.Threads
			break
		}
	}
	if baseThreads == 0 && len(sr.OptimizeResult.StepResults) > 0 {
		baseThreads = sr.OptimizeResult.StepResults[0].Threads
	}

	threadsPerGen := baseThreads / generators
	if threadsPerGen < 1 && baseThreads > 0 {
		threadsPerGen = 1
	}

	opsMinPerThread := sr.OptimizeResult.BestOpsPerMinPerThread

	intensityUnit := ""
	intensityVal := 0.0
	if sr.IsOpenModel {
		intensityUnit = "ops/s"
		intensityVal = sr.SingleResult.OutputValue / float64(generators)
		if sr.SingleResult.OutputUnit != "" {
			intensityUnit = sr.SingleResult.OutputUnit
		}
	}

	return []interface{}{
		sr.Scenario.Name,
		tgType,
		threadsPerGen,
		opsMinPerThread,
		intensityUnit,
		intensityVal,
	}
}

func (w *XLSXWriter) writeJMeterConfig(f *excelize.File, results engine.CalculationResults, boldStyle int) error {
	sheet := "JMeter Config"
	headers := []string{
		"Scenario Name", "Thread Group Type", "Threads (per generator)",
		"CTT ops/min (per thread)", "Intensity unit", "Intensity value (per generator)",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := setCellValue(f, sheet, cell, h); err != nil {
			return err
		}
		if err := setCellStyle(f, sheet, cell, boldStyle); err != nil {
			return err
		}
	}

	generators := results.Plan.GlobalDefaults.GeneratorsCount
	if generators < 1 {
		generators = 1
	}

	row := 2
	for _, sr := range results.ScenarioResults {
		vals := jmeterConfigRowValues(sr, generators)
		for i, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			if err := setCellValue(f, sheet, cell, v); err != nil {
				return err
			}
		}
		row++
	}

	return autoWidth(f, sheet, headers)
}

func boolYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func strPtr(s string) *string {
	return &s
}

// autoWidth sets approximate column widths based on header lengths.
func autoWidth(f *excelize.File, sheet string, headers []string) error {
	for i, h := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		width := float64(len(h)) + 4
		if width < 12 {
			width = 12
		}
		if err := f.SetColWidth(sheet, col, col, width); err != nil {
			return err
		}
	}
	return nil
}
