package tui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"loadcalc/internal/config"
	"loadcalc/internal/engine"
	"loadcalc/internal/profile"
	"loadcalc/internal/tui/styles"
	"loadcalc/pkg/units"
)

type fieldType int

const (
	fieldText  fieldType = iota
	fieldCycle fieldType = iota
)

// Field label constants.
const (
	labelIntensity  = "Intensity"
	labelUnit       = "Unit"
	labelScriptTime = "Script time"
	labelTool       = "Tool"
	labelModel      = "Model"
	labelMultiplier = "Multiplier"
	labelRangeDown  = "Range down"
	labelRangeUp    = "Range up"
	labelGenerators = "Generators"
	labelSteps      = "Steps"
	labelRampup     = "Rampup"
)

type field struct {
	label     string
	options   []string
	fieldType fieldType
}

// QuickModel is the bubbletea model for the quick calculator TUI.
type QuickModel struct {
	unit        string
	intensity   string
	scriptTime  string
	tool        string
	model       string
	multiplier  string
	rangeDown   string
	rangeUp     string
	generators  string
	steps       string
	rampup      string
	resultText  string
	err         string
	fields      []field
	activeField int
}

// NewQuickModel creates a QuickModel with sensible defaults.
func NewQuickModel() QuickModel {
	m := QuickModel{
		fields: []field{
			{label: labelIntensity, fieldType: fieldText},
			{label: labelUnit, fieldType: fieldCycle, options: []string{"ops_h", "ops_m", "ops_s"}},
			{label: labelScriptTime, fieldType: fieldText},
			{label: labelTool, fieldType: fieldCycle, options: []string{"jmeter", "lre_pc"}},
			{label: labelModel, fieldType: fieldCycle, options: []string{"closed", "open"}},
			{label: labelMultiplier, fieldType: fieldText},
			{label: labelRangeDown, fieldType: fieldText},
			{label: labelRangeUp, fieldType: fieldText},
			{label: labelGenerators, fieldType: fieldText},
			{label: labelSteps, fieldType: fieldText},
			{label: labelRampup, fieldType: fieldText},
		},
		activeField: 0,
		intensity:   "720000",
		scriptTime:  "1100",
		tool:        "jmeter",
		model:       "closed",
		unit:        "ops_h",
		multiplier:  "3.0",
		rangeDown:   "0.2",
		rangeUp:     "0.5",
		generators:  "1",
		steps:       "50,75,100,125,150",
		rampup:      "60",
	}
	m.recalculate()
	return m
}

// Init implements tea.Model.
func (m QuickModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m QuickModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return m.handleKey(keyMsg)
	}
	return m, nil
}

func (m QuickModel) isFieldVisible(idx int) bool {
	label := m.fields[idx].label
	if m.tool == "lre_pc" {
		if label == labelGenerators || label == labelModel {
			return false
		}
	}
	return true
}

func (m QuickModel) nextVisibleField(dir int) int {
	n := len(m.fields)
	idx := m.activeField
	for range n {
		idx = (idx + dir + n) % n
		if m.isFieldVisible(idx) {
			return idx
		}
	}
	return m.activeField
}

func (m QuickModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type { //nolint:exhaustive // only relevant keys handled
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyTab, tea.KeyDown:
		m.activeField = m.nextVisibleField(1)
		return m, nil
	case tea.KeyShiftTab, tea.KeyUp:
		m.activeField = m.nextVisibleField(-1)
		return m, nil
	case tea.KeyEnter:
		m.recalculate()
		return m, nil
	default:
		// fall through to field-specific handling
	}

	f := m.fields[m.activeField]
	if f.fieldType == fieldCycle {
		return m.handleCycleKey(msg)
	}
	return m.handleTextKey(msg)
}

func (m QuickModel) handleCycleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type { //nolint:exhaustive // only relevant keys handled
	case tea.KeySpace, tea.KeyRight, tea.KeyLeft:
		f := m.fields[m.activeField]
		current := m.getFieldValue(f.label)
		idx := 0
		for i, o := range f.options {
			if o == current {
				idx = i
				break
			}
		}
		if msg.Type == tea.KeyLeft {
			idx = (idx - 1 + len(f.options)) % len(f.options)
		} else {
			idx = (idx + 1) % len(f.options)
		}
		m.setFieldValue(f.label, f.options[idx])
		// If tool changed and current active field became hidden, move to next visible.
		if !m.isFieldVisible(m.activeField) {
			m.activeField = m.nextVisibleField(1)
		}
		m.recalculate()
		return m, nil
	default:
		return m, nil
	}
}

func (m QuickModel) handleTextKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	f := m.fields[m.activeField]
	val := m.getFieldValue(f.label)

	switch msg.Type { //nolint:exhaustive // only relevant keys handled
	case tea.KeyBackspace:
		if val != "" {
			val = val[:len(val)-1]
		}
	case tea.KeyRunes:
		val += string(msg.Runes)
	default:
		if msg.String() == "q" {
			val += "q"
		} else {
			return m, nil
		}
	}
	m.setFieldValue(f.label, val)
	m.recalculate()
	return m, nil
}

func (m *QuickModel) getFieldValue(label string) string {
	switch label {
	case labelIntensity:
		return m.intensity
	case labelUnit:
		return m.unit
	case labelScriptTime:
		return m.scriptTime
	case labelTool:
		return m.tool
	case labelModel:
		return m.model
	case labelMultiplier:
		return m.multiplier
	case labelRangeDown:
		return m.rangeDown
	case labelRangeUp:
		return m.rangeUp
	case labelGenerators:
		return m.generators
	case labelSteps:
		return m.steps
	case labelRampup:
		return m.rampup
	}
	return ""
}

func (m *QuickModel) setFieldValue(label, val string) {
	switch label {
	case labelIntensity:
		m.intensity = val
	case labelUnit:
		m.unit = val
	case labelScriptTime:
		m.scriptTime = val
	case labelTool:
		m.tool = val
	case labelModel:
		m.model = val
	case labelMultiplier:
		m.multiplier = val
	case labelRangeDown:
		m.rangeDown = val
	case labelRangeUp:
		m.rangeUp = val
	case labelGenerators:
		m.generators = val
	case labelSteps:
		m.steps = val
	case labelRampup:
		m.rampup = val
	}
}

type quickParams struct { //nolint:govet // fieldalignment: contains config.Scenario with pointers
	scenario   config.Scenario
	unitLabel  string
	tool       config.Tool
	loadModel  config.LoadModel
	rangeDown  float64
	rangeUp    float64
	rampupSec  int
	generators int
}

func (m *QuickModel) parseParams() (*quickParams, error) {
	intensity, err := strconv.ParseFloat(m.intensity, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid intensity: %w", err)
	}
	scriptTimeMs, err := strconv.Atoi(m.scriptTime)
	if err != nil {
		return nil, fmt.Errorf("invalid script time: %w", err)
	}
	multiplier, err := strconv.ParseFloat(m.multiplier, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid multiplier: %w", err)
	}
	rangeDown, err := strconv.ParseFloat(m.rangeDown, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid range down: %w", err)
	}
	rangeUp, err := strconv.ParseFloat(m.rangeUp, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid range up: %w", err)
	}
	rampupSec, err := strconv.Atoi(m.rampup)
	if err != nil {
		return nil, fmt.Errorf("invalid rampup: %w", err)
	}

	var tool config.Tool
	switch m.tool {
	case "jmeter":
		tool = config.ToolJMeter
	case "lre_pc":
		tool = config.ToolLREPC
	default:
		return nil, fmt.Errorf("unknown tool: %s", m.tool)
	}

	generators := 1
	if tool != config.ToolLREPC {
		generators, err = strconv.Atoi(m.generators)
		if err != nil {
			return nil, fmt.Errorf("invalid generators: %w", err)
		}
		if generators < 1 {
			return nil, fmt.Errorf("generators must be >= 1")
		}
	}

	loadModel := config.LoadModel(m.model)
	if tool == config.ToolLREPC {
		loadModel = config.LoadModelClosed
	}

	intensityUnit := units.IntensityUnit(m.unit)
	tolerance := 2.5

	return &quickParams{
		tool:       tool,
		loadModel:  loadModel,
		generators: generators,
		scenario: config.Scenario{
			Name:               "quick",
			TargetIntensity:    intensity,
			IntensityUnit:      intensityUnit,
			MaxScriptTimeMs:    scriptTimeMs,
			PacingMultiplier:   &multiplier,
			DeviationTolerance: &tolerance,
		},
		unitLabel: formatUnitLabel(intensityUnit),
		rampupSec: rampupSec,
		rangeDown: rangeDown,
		rangeUp:   rangeUp,
	}, nil
}

func (m *QuickModel) recalculate() {
	m.resultText = ""
	m.err = ""

	// Force LRE PC to closed model in the UI field.
	if m.tool == "lre_pc" {
		m.model = "closed"
	}

	p, err := m.parseParams()
	if err != nil {
		m.err = err.Error()
		return
	}

	if m.steps == "" {
		m.calcSingle(p.scenario, p.tool, p.loadModel, p.generators, p.unitLabel)
	} else {
		m.calcMultiStep(p.scenario, p.tool, p.loadModel, p.generators, p.unitLabel, p.rampupSec, p.rangeDown, p.rangeUp)
	}
}

func (m *QuickModel) calcSingle(scenario config.Scenario, tool config.Tool, loadModel config.LoadModel, generators int, _ string) {
	targetRPS, err := units.NormalizeToOpsPerSec(scenario.TargetIntensity, scenario.IntensityUnit)
	if err != nil {
		m.err = err.Error()
		return
	}

	calc, err := engine.NewCalculator(tool, loadModel, generators)
	if err != nil {
		m.err = err.Error()
		return
	}

	result, err := calc.Calculate(scenario, targetRPS)
	if err != nil {
		m.err = err.Error()
		return
	}

	var sb strings.Builder
	if loadModel == config.LoadModelOpen {
		ratePerGen := targetRPS / float64(max(generators, 1))
		fmt.Fprintf(&sb, "Rate: %.2f ops/sec (per generator)\n", ratePerGen)
	} else {
		fmt.Fprintf(&sb, "Pacing: %s ms    ", quickFormatNumber(result.PacingMS))
		if tool == config.ToolJMeter {
			fmt.Fprintf(&sb, "CTT: %.2f ops/min/thread", result.OpsPerMinPerThread)
		}
		sb.WriteString("\n")
		fmt.Fprintf(&sb, "Threads: %s    ", quickFormatNumber(float64(result.Threads)))
		fmt.Fprintf(&sb, "Deviation: %.2f%% %s\n",
			result.DeviationPercent,
			styles.FormatDeviation(result.DeviationPercent, *scenario.DeviationTolerance))
	}
	m.resultText = sb.String()
}

func (m *QuickModel) calcMultiStep(scenario config.Scenario, tool config.Tool, loadModel config.LoadModel, generators int, unitLabel string, rampupSec int, rangeDown, rangeUp float64) {
	parts := strings.Split(m.steps, ",")
	var stepList []profile.Step
	for i, p := range parts {
		pct, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			m.err = fmt.Sprintf("invalid step %q: %v", p, err)
			return
		}
		stepList = append(stepList, profile.Step{
			StepNumber:      i + 1,
			PercentOfTarget: pct,
		})
	}

	if loadModel == config.LoadModelOpen {
		m.calcMultiStepOpen(scenario, stepList, generators, unitLabel)
		return
	}

	opt := &engine.Optimizer{MultiplierRangeDown: rangeDown, MultiplierRangeUp: rangeUp}
	optResult, err := opt.Optimize(scenario, stepList, tool, loadModel, generators)
	if err != nil {
		m.err = err.Error()
		return
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Pacing: %s ms", quickFormatNumber(optResult.BestPacingMS))
	if tool == config.ToolJMeter {
		fmt.Fprintf(&sb, "    CTT: %.2f ops/min/thread", optResult.BestOpsPerMinPerThread)
	}
	sb.WriteString("\n\n")

	if tool == config.ToolJMeter {
		fmt.Fprintf(&sb, "  %-4s %4s %8s %6s %12s %8s\n", "Step", "%", "Threads", "/Gen", "Actual "+unitLabel, "Dev")
		for _, sr := range optResult.StepResults {
			actualDisplay := units.ConvertFromOpsPerSec(sr.ActualRPS*float64(generators), scenario.IntensityUnit)
			threadsTotal := sr.Threads * generators
			level := styles.ClassifyDeviation(sr.DeviationPct, *scenario.DeviationTolerance)
			sym := styles.DeviationSymbol(level)
			fmt.Fprintf(&sb, "  %3d  %3.0f%%   %6d  %5d  %11s  %5.2f%% %s\n",
				sr.Step.StepNumber, sr.Step.PercentOfTarget,
				threadsTotal, sr.Threads,
				quickFormatNumber(actualDisplay),
				sr.DeviationPct, sym)
		}
	} else {
		fmt.Fprintf(&sb, "  %-4s %4s %7s %6s %6s %8s %7s %8s\n", "Step", "%", "Vusers", "Delta", "Batch", "Every(s)", "Rampup", "Dev")
		prevThreads := 0
		for _, sr := range optResult.StepResults {
			delta := sr.Threads - prevThreads
			ramp := engine.CalculateRampUp(delta, rampupSec)
			level := styles.ClassifyDeviation(sr.DeviationPct, *scenario.DeviationTolerance)
			sym := styles.DeviationSymbol(level)
			fmt.Fprintf(&sb, "  %3d  %3.0f%%   %5d   +%-4d  %4d    %4ds    %4ds  %5.2f%% %s\n",
				sr.Step.StepNumber, sr.Step.PercentOfTarget,
				sr.Threads, delta, ramp.BatchSize,
				ramp.IntervalSec, ramp.ActualSec,
				sr.DeviationPct, sym)
			prevThreads = sr.Threads
		}
	}

	m.resultText = sb.String()
}

func (m *QuickModel) calcMultiStepOpen(scenario config.Scenario, stepList []profile.Step, generators int, unitLabel string) {
	baseRPS, err := units.NormalizeToOpsPerSec(scenario.TargetIntensity, scenario.IntensityUnit)
	if err != nil {
		m.err = err.Error()
		return
	}

	var sb strings.Builder
	sb.WriteString("Open model — rate per step\n\n")
	fmt.Fprintf(&sb, "  %-4s %4s %12s %12s\n", "Step", "%", "Rate "+unitLabel, "Per Generator")
	for _, step := range stepList {
		stepRPS := baseRPS * step.PercentOfTarget / 100
		totalDisplay := units.ConvertFromOpsPerSec(stepRPS, scenario.IntensityUnit)
		perGen := totalDisplay / float64(generators)
		fmt.Fprintf(&sb, "  %3d  %3.0f%%  %11s  %11s\n",
			step.StepNumber, step.PercentOfTarget,
			quickFormatNumber(totalDisplay), quickFormatNumber(perGen))
	}
	m.resultText = sb.String()
}

// View implements tea.Model.
func (m QuickModel) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorCyan)
	activeStyle := lipgloss.NewStyle().Background(lipgloss.Color("#333333")).Foreground(lipgloss.Color("#FFFFFF"))
	labelStyle := lipgloss.NewStyle().Foreground(styles.ColorWhite).Width(14)
	errorStyle := lipgloss.NewStyle().Foreground(styles.ColorRed)
	helpStyle := lipgloss.NewStyle().Foreground(styles.ColorGray)
	sepStyle := lipgloss.NewStyle().Foreground(styles.ColorGray)

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("loadcalc quick calculator"))
	sb.WriteString("\n\n")

	for i, f := range m.fields {
		// Hide generators field for LRE PC (not applicable).
		if f.label == labelGenerators && m.tool == "lre_pc" {
			continue
		}
		label := labelStyle.Render(f.label + ":")
		val := m.getFieldValueByIndex(i)
		display := val
		if f.fieldType == fieldCycle {
			display = formatCycleDisplay(val)
		}

		if i == m.activeField {
			if f.fieldType == fieldText {
				display = "[" + display + "\u258f]"
			} else {
				display = "[" + display + " \u25be]"
			}
			display = activeStyle.Render(display)
		} else {
			display = " " + display
		}

		suffix := ""
		switch f.label {
		case labelScriptTime:
			suffix = " ms"
		case labelRangeDown:
			suffix = " -"
		case labelRangeUp:
			suffix = " +"
		case labelRampup:
			suffix = " s"
		}

		sb.WriteString("  " + label + " " + display + suffix + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString("  " + sepStyle.Render("\u2500\u2500 Results \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500"))
	sb.WriteString("\n")

	if m.err != "" {
		sb.WriteString("  " + errorStyle.Render(m.err) + "\n")
	} else if m.resultText != "" {
		for _, line := range strings.Split(m.resultText, "\n") {
			if line != "" {
				sb.WriteString("  " + line + "\n")
			}
		}
	}

	sb.WriteString("\n")
	sb.WriteString("  " + helpStyle.Render("[Tab] next field  [Space/\u2190/\u2192] cycle  [Ctrl+C] quit"))
	sb.WriteString("\n")

	return sb.String()
}

func (m *QuickModel) getFieldValueByIndex(i int) string {
	return m.getFieldValue(m.fields[i].label)
}

func formatCycleDisplay(val string) string {
	switch val {
	case "ops_h":
		return "ops/h"
	case "ops_m":
		return "ops/m"
	case "ops_s":
		return "ops/s"
	case "lre_pc":
		return "LRE PC"
	default:
		return val
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

func quickFormatNumber(v float64) string {
	rounded := math.Round(v*100) / 100
	isInt := rounded == math.Trunc(rounded)
	var s string
	if isInt {
		s = strconv.FormatInt(int64(rounded), 10)
	} else {
		s = strconv.FormatFloat(rounded, 'f', -1, 64)
	}

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

// RunQuick launches the quick calculator TUI.
func RunQuick() error {
	m := NewQuickModel()
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}
