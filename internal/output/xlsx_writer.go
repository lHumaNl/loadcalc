package output

import (
	"fmt"

	"github.com/xuri/excelize/v2"

	"loadcalc/internal/config"
	"loadcalc/internal/engine"
)

// XLSXWriter writes calculation results to an XLSX file.
type XLSXWriter struct{}

// createSheets creates the workbook sheets based on tool type.
func createSheets(f *excelize.File, tool config.Tool) error {
	if err := f.SetSheetName("Sheet1", "Summary"); err != nil {
		return fmt.Errorf("renaming sheet: %w", err)
	}
	for _, name := range []string{"Steps Detail", "Timeline", "Input Parameters"} {
		if _, err := f.NewSheet(name); err != nil {
			return fmt.Errorf("creating %s sheet: %w", name, err)
		}
	}
	if tool == config.ToolJMeter {
		if _, err := f.NewSheet("JMeter Config"); err != nil {
			return fmt.Errorf("creating JMeter Config sheet: %w", err)
		}
	}
	if tool == config.ToolLREPC {
		if _, err := f.NewSheet("LRE PC Config"); err != nil {
			return fmt.Errorf("creating LRE PC Config sheet: %w", err)
		}
	}
	return nil
}

// Write generates the XLSX workbook and saves it to dest.
func (w *XLSXWriter) Write(results engine.CalculationResults, dest string) error {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	if err := createSheets(f, results.Plan.GlobalDefaults.Tool); err != nil {
		return err
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

	if err := w.writeSummary(f, results, boldStyle, redBgStyle); err != nil {
		return err
	}
	if err := w.writeStepsDetail(f, results, boldStyle, redBgStyle); err != nil {
		return err
	}
	if err := w.writeTimeline(f, results, boldStyle); err != nil {
		return err
	}
	if err := w.writeInputParameters(f, results, boldStyle); err != nil {
		return err
	}
	if results.Plan.GlobalDefaults.Tool == config.ToolJMeter {
		if err := w.writeJMeterConfig(f, results, boldStyle); err != nil {
			return err
		}
	}
	if results.Plan.GlobalDefaults.Tool == config.ToolLREPC {
		if err := w.writeLREPCConfig(f, results, boldStyle); err != nil {
			return err
		}
	}

	return f.SaveAs(dest)
}

func setCellValue(f *excelize.File, sheet, cell string, value interface{}) error {
	return f.SetCellValue(sheet, cell, value)
}

// writeRow writes a slice of values to the given row in a sheet.
func writeRow(f *excelize.File, sheet string, row int, vals []interface{}) error {
	for i, v := range vals {
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		if err := setCellValue(f, sheet, cell, v); err != nil {
			return err
		}
	}
	return nil
}

// backgroundThreads returns the thread count for a background scenario.
func backgroundThreads(sr engine.ScenarioResult) int {
	if len(sr.OptimizeResult.StepResults) > 0 {
		return sr.OptimizeResult.StepResults[0].Threads
	}
	return 0
}

// threadsPerGenerator calculates threads per generator with minimum of 1 when threads > 0.
func threadsPerGenerator(threads, generators int) int {
	tpg := threads / generators
	if tpg < 1 && threads > 0 {
		tpg = 1
	}
	return tpg
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

func (w *XLSXWriter) writeJMeterConfig(f *excelize.File, results engine.CalculationResults, boldStyle int) error {
	sheet := "JMeter Config"
	headers := []string{
		"Scenario", "Step", "Step %", "Threads", "Threads Delta",
		"Threads/Generator", "CTT ops/min/thread", "Rampup (s)", "Hold (s)",
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
		if sr.IsOpenModel {
			continue
		}
		if sr.IsBackground {
			threads := backgroundThreads(sr)
			vals := []interface{}{
				sr.Scenario.Name, "BG", "-", threads, threads,
				threadsPerGenerator(threads, generators), sr.OptimizeResult.BestOpsPerMinPerThread, 0, 0,
			}
			if err := writeRow(f, sheet, row, vals); err != nil {
				return err
			}
			row++
			continue
		}
		prevThreads := 0
		for _, stepRes := range sr.OptimizeResult.StepResults {
			delta := stepRes.Threads - prevThreads
			vals := []interface{}{
				sr.Scenario.Name, stepRes.Step.StepNumber, stepRes.Step.PercentOfTarget,
				stepRes.Threads, delta, threadsPerGenerator(stepRes.Threads, generators),
				sr.OptimizeResult.BestOpsPerMinPerThread, stepRes.Step.RampupSec,
				stepRes.Step.StabilitySec + stepRes.Step.ImpactSec,
			}
			if err := writeRow(f, sheet, row, vals); err != nil {
				return err
			}
			prevThreads = stepRes.Threads
			row++
		}
	}

	return autoWidth(f, sheet, headers)
}

func (w *XLSXWriter) writeLREPCConfig(f *excelize.File, results engine.CalculationResults, boldStyle int) error {
	sheet := "LRE PC Config"
	headers := []string{
		"Scenario", "Step", "Step %", "Vusers Total", "Vusers Delta",
		"Batch Size", "Interval (s)", "Actual Rampup (s)", "Target Rampup (s)",
		"Pacing (ms)", "Hold (s)",
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

	row := 2
	for _, sr := range results.ScenarioResults {
		if sr.IsOpenModel {
			continue
		}
		if sr.IsBackground {
			threads := backgroundThreads(sr)
			rampCfg := engine.CalculateRampUp(threads, 0)
			vals := []interface{}{
				sr.Scenario.Name, "BG", "-", threads, threads,
				rampCfg.BatchSize, rampCfg.IntervalSec, rampCfg.ActualSec, 0,
				sr.OptimizeResult.BestPacingMS, 0,
			}
			if err := writeRow(f, sheet, row, vals); err != nil {
				return err
			}
			row++
			continue
		}
		prevThreads := 0
		for _, stepRes := range sr.OptimizeResult.StepResults {
			delta := stepRes.Threads - prevThreads
			rampCfg := engine.CalculateRampUp(delta, stepRes.Step.RampupSec)
			vals := []interface{}{
				sr.Scenario.Name, stepRes.Step.StepNumber, stepRes.Step.PercentOfTarget,
				stepRes.Threads, delta, rampCfg.BatchSize, rampCfg.IntervalSec,
				rampCfg.ActualSec, stepRes.Step.RampupSec, sr.OptimizeResult.BestPacingMS,
				stepRes.Step.StabilitySec + stepRes.Step.ImpactSec,
			}
			if err := writeRow(f, sheet, row, vals); err != nil {
				return err
			}
			prevThreads = stepRes.Threads
			row++
		}
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
