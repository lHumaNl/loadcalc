// Package integration handles export to tool-specific configuration formats.
package integration

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"

	"loadcalc/internal/config"
	"loadcalc/internal/engine"
	"loadcalc/internal/profile"
)

// Thread group type constants.
const (
	tgFFATG    = "FreeFormArrivalsThreadGroup"
	tgStepping = "SteppingThreadGroup"
	tgUltimate = "UltimateThreadGroup"
)

// selectThreadGroupType determines which JMeter ThreadGroup element to use.
func selectThreadGroupType(steps []profile.Step, loadModel config.LoadModel) string {
	if loadModel == config.LoadModelOpen {
		return tgFFATG
	}
	if len(steps) <= 1 {
		return tgStepping
	}
	// Check if increments are uniform
	increment := steps[1].PercentOfTarget - steps[0].PercentOfTarget
	for i := 2; i < len(steps); i++ {
		diff := steps[i].PercentOfTarget - steps[i-1].PercentOfTarget
		if math.Abs(diff-increment) > 0.001 {
			return tgUltimate
		}
	}
	return tgStepping
}

// getLoadModel returns the effective load model for a scenario result.
func getLoadModel(sr engine.ScenarioResult, globalModel config.LoadModel) config.LoadModel {
	if sr.Scenario.LoadModel != nil {
		return *sr.Scenario.LoadModel
	}
	return globalModel
}

// GenerateJMX creates a complete .jmx file from calculation results.
func GenerateJMX(results engine.CalculationResults) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	buf.WriteString(`<jmeterTestPlan version="1.2" properties="5.0" jmeter="5.6.3">` + "\n")
	buf.WriteString("  <hashTree>\n")
	buf.WriteString(`    <TestPlan guiclass="TestPlanGui" testclass="TestPlan" testname="Test Plan" enabled="true">` + "\n")
	buf.WriteString(`      <stringProp name="TestPlan.comments"></stringProp>` + "\n")
	buf.WriteString(`      <boolProp name="TestPlan.functional_mode">false</boolProp>` + "\n")
	buf.WriteString(`      <boolProp name="TestPlan.tearDown_on_shutdown">true</boolProp>` + "\n")
	buf.WriteString(`      <elementProp name="TestPlan.user_defined_variables" elementType="Arguments">` + "\n")
	buf.WriteString(`        <collectionProp name="Arguments.arguments"/>` + "\n")
	buf.WriteString("      </elementProp>\n")
	buf.WriteString("    </TestPlan>\n")
	buf.WriteString("    <hashTree>\n")

	for _, sr := range results.ScenarioResults {
		lm := getLoadModel(sr, results.Plan.GlobalDefaults.LoadModel)
		tgXML := buildThreadGroup(sr, results.Steps, lm)
		buf.WriteString(tgXML)
	}

	// Test Fragments
	for _, sr := range results.ScenarioResults {
		name := sr.Scenario.Name
		fmt.Fprintf(&buf, `      <TestFragmentController guiclass="TestFragmentControllerGui" testclass="TestFragmentController" testname="Test Fragment - %s" enabled="false">`+"\n", xmlEscape(name))
		fmt.Fprintf(&buf, `        <GenericController guiclass="LogicControllerGui" testclass="GenericController" testname="%s" enabled="true"/>`+"\n", xmlEscape(name))
		buf.WriteString("      </TestFragmentController>\n")
		buf.WriteString("      <hashTree/>\n")
	}

	buf.WriteString("    </hashTree>\n")
	buf.WriteString("  </hashTree>\n")
	buf.WriteString("</jmeterTestPlan>\n")
	return buf.Bytes(), nil
}

func buildThreadGroup(sr engine.ScenarioResult, steps []profile.Step, lm config.LoadModel) string {
	var buf bytes.Buffer
	tgType := selectThreadGroupType(steps, lm)
	name := sr.Scenario.Name
	switch {
	case lm == config.LoadModelOpen:
		name = "FFATG_" + sr.Scenario.Name
	case tgType == tgStepping:
		name = "STG_" + sr.Scenario.Name
	case tgType == tgUltimate:
		name = "UTG_" + sr.Scenario.Name
	}

	switch tgType {
	case tgStepping:
		buf.WriteString(buildSteppingTG(name, sr, steps))
	case tgUltimate:
		buf.WriteString(buildUltimateTG(name, sr, steps))
	case tgFFATG:
		buf.WriteString(buildFFATG(name, sr, steps))
	}

	// hashTree contents: CTT + ModuleController
	buf.WriteString("      <hashTree>\n")
	if lm != config.LoadModelOpen {
		buf.WriteString(buildCTT(sr))
	}
	buf.WriteString(buildModuleController(sr.Scenario.Name))
	buf.WriteString("      </hashTree>\n")

	return buf.String()
}

func buildSteppingTG(name string, sr engine.ScenarioResult, steps []profile.Step) string {
	maxThreads := 0
	threadsPerStep := 0
	rampupSec := 60
	stabilitySec := 300

	if len(sr.OptimizeResult.StepResults) > 0 {
		last := sr.OptimizeResult.StepResults[len(sr.OptimizeResult.StepResults)-1]
		maxThreads = last.Threads
		if len(sr.OptimizeResult.StepResults) > 1 {
			threadsPerStep = sr.OptimizeResult.StepResults[1].Threads - sr.OptimizeResult.StepResults[0].Threads
		} else {
			threadsPerStep = maxThreads
		}
	}
	if len(steps) > 0 {
		rampupSec = steps[0].RampupSec
		stabilitySec = steps[0].StabilitySec
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, `      <kg.apc.jmeter.threads.SteppingThreadGroup guiclass="kg.apc.jmeter.threads.SteppingThreadGroupGui" testclass="kg.apc.jmeter.threads.SteppingThreadGroup" testname="%s" enabled="true">`+"\n", xmlEscape(name))
	buf.WriteString(`        <stringProp name="ThreadGroup.on_sample_error">continue</stringProp>` + "\n")
	fmt.Fprintf(&buf, `        <stringProp name="ThreadGroup.num_threads">%d</stringProp>`+"\n", maxThreads)
	buf.WriteString(`        <stringProp name="Threads initial delay">0</stringProp>` + "\n")
	fmt.Fprintf(&buf, `        <stringProp name="Start users count">%d</stringProp>`+"\n", threadsPerStep)
	buf.WriteString(`        <stringProp name="Start users count burst">0</stringProp>` + "\n")
	fmt.Fprintf(&buf, `        <stringProp name="Start users period">%d</stringProp>`+"\n", rampupSec)
	buf.WriteString(`        <stringProp name="Stop users count">0</stringProp>` + "\n")
	buf.WriteString(`        <stringProp name="Stop users period">0</stringProp>` + "\n")
	fmt.Fprintf(&buf, `        <stringProp name="flighttime">%d</stringProp>`+"\n", stabilitySec)
	fmt.Fprintf(&buf, `        <stringProp name="rampUp">%d</stringProp>`+"\n", rampupSec)
	buf.WriteString(`        <elementProp name="ThreadGroup.main_controller" elementType="LoopController">` + "\n")
	buf.WriteString(`          <boolProp name="LoopController.continue_forever">false</boolProp>` + "\n")
	buf.WriteString(`          <intProp name="LoopController.loops">-1</intProp>` + "\n")
	buf.WriteString("        </elementProp>\n")
	buf.WriteString("      </kg.apc.jmeter.threads.SteppingThreadGroup>\n")
	return buf.String()
}

func buildUltimateTG(name string, sr engine.ScenarioResult, steps []profile.Step) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, `      <kg.apc.jmeter.threads.UltimateThreadGroup guiclass="kg.apc.jmeter.threads.UltimateThreadGroupGui" testclass="kg.apc.jmeter.threads.UltimateThreadGroup" testname="%s" enabled="true">`+"\n", xmlEscape(name))
	buf.WriteString(`        <collectionProp name="ultimatethreadgroupdata">` + "\n")

	cumulativeDelay := 0
	prevThreads := 0
	for i, stepR := range sr.OptimizeResult.StepResults {
		threads := stepR.Threads - prevThreads
		if threads < 0 {
			threads = 0
		}
		rampup := 60
		hold := 300
		if i < len(steps) {
			rampup = steps[i].RampupSec
			hold = steps[i].StabilitySec
		}
		fmt.Fprintf(&buf, `          <collectionProp name="row_%d">`+"\n", i)
		fmt.Fprintf(&buf, `            <stringProp name="0">%d</stringProp>`+"\n", threads)
		fmt.Fprintf(&buf, `            <stringProp name="1">%d</stringProp>`+"\n", cumulativeDelay)
		fmt.Fprintf(&buf, `            <stringProp name="2">%d</stringProp>`+"\n", rampup)
		fmt.Fprintf(&buf, `            <stringProp name="3">%d</stringProp>`+"\n", hold)
		buf.WriteString(`            <stringProp name="4">0</stringProp>` + "\n")
		buf.WriteString("          </collectionProp>\n")
		cumulativeDelay += rampup + hold
		prevThreads = stepR.Threads
	}

	buf.WriteString("        </collectionProp>\n")
	buf.WriteString(`        <elementProp name="ThreadGroup.main_controller" elementType="LoopController">` + "\n")
	buf.WriteString(`          <boolProp name="LoopController.continue_forever">false</boolProp>` + "\n")
	buf.WriteString(`          <intProp name="LoopController.loops">-1</intProp>` + "\n")
	buf.WriteString("        </elementProp>\n")
	buf.WriteString("      </kg.apc.jmeter.threads.UltimateThreadGroup>\n")
	return buf.String()
}

func buildFFATG(name string, sr engine.ScenarioResult, steps []profile.Step) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, `      <com.blazemeter.jmeter.threads.arrivals.FreeFormArrivalsThreadGroup guiclass="com.blazemeter.jmeter.threads.arrivals.FreeFormArrivalsThreadGroupGui" testclass="com.blazemeter.jmeter.threads.arrivals.FreeFormArrivalsThreadGroup" testname="%s" enabled="true">`+"\n", xmlEscape(name))
	buf.WriteString(`        <stringProp name="ConcurrencyLimit">100</stringProp>` + "\n")
	buf.WriteString(`        <elementProp name="Schedule" elementType="com.blazemeter.jmeter.threads.arrivals.FreeFormArrivalsThreadGroupGui$SchedulePanel">` + "\n")
	buf.WriteString(`          <collectionProp name="schedule">` + "\n")

	prevRPS := 0.0
	for i, stepR := range sr.OptimizeResult.StepResults {
		duration := 300
		if i < len(steps) {
			duration = steps[i].RampupSec + steps[i].StabilitySec
		}
		endRPS := stepR.TargetRPS
		fmt.Fprintf(&buf, `            <collectionProp name="row_%d">`+"\n", i)
		fmt.Fprintf(&buf, `              <stringProp name="0">%.0f</stringProp>`+"\n", prevRPS)
		fmt.Fprintf(&buf, `              <stringProp name="1">%.0f</stringProp>`+"\n", endRPS)
		fmt.Fprintf(&buf, `              <stringProp name="2">%d</stringProp>`+"\n", duration)
		buf.WriteString("            </collectionProp>\n")
		prevRPS = endRPS
	}

	buf.WriteString("          </collectionProp>\n")
	buf.WriteString("        </elementProp>\n")
	buf.WriteString("      </com.blazemeter.jmeter.threads.arrivals.FreeFormArrivalsThreadGroup>\n")
	return buf.String()
}

func buildCTT(sr engine.ScenarioResult) string {
	opsPerMin := sr.OptimizeResult.BestOpsPerMinPerThread
	var buf bytes.Buffer
	buf.WriteString(`        <ConstantThroughputTimer guiclass="TestBeanGUI" testclass="ConstantThroughputTimer" testname="Constant Throughput Timer" enabled="true">` + "\n")
	buf.WriteString(`          <intProp name="calcMode">1</intProp>` + "\n")
	fmt.Fprintf(&buf, `          <doubleProp><name>throughput</name><value>%.2f</value></doubleProp>`+"\n", opsPerMin)
	buf.WriteString("        </ConstantThroughputTimer>\n")
	buf.WriteString("        <hashTree/>\n")
	return buf.String()
}

func buildModuleController(scenarioName string) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, `        <ModuleController guiclass="ModuleControllerGui" testclass="ModuleController" testname="Module: %s" enabled="true">`+"\n", xmlEscape(scenarioName))
	buf.WriteString(`          <collectionProp name="ModuleController.node_path">` + "\n")
	buf.WriteString(`            <stringProp name="764597751">Test Plan</stringProp>` + "\n")
	fmt.Fprintf(&buf, `            <stringProp name="0">Test Fragment - %s</stringProp>`+"\n", xmlEscape(scenarioName))
	fmt.Fprintf(&buf, `            <stringProp name="1">%s</stringProp>`+"\n", xmlEscape(scenarioName))
	buf.WriteString("          </collectionProp>\n")
	buf.WriteString("        </ModuleController>\n")
	buf.WriteString("        <hashTree/>\n")
	return buf.String()
}

func xmlEscape(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

// InjectIntoJMX adds ThreadGroups to an existing .jmx template.
func InjectIntoJMX(templatePath string, results engine.CalculationResults) ([]byte, error) {
	data, err := os.ReadFile(templatePath) //nolint:gosec // user-provided file path
	if err != nil {
		return nil, fmt.Errorf("reading template: %w", err)
	}

	// Build the XML fragments to inject
	var inject bytes.Buffer
	for _, sr := range results.ScenarioResults {
		lm := getLoadModel(sr, results.Plan.GlobalDefaults.LoadModel)
		inject.WriteString(buildThreadGroup(sr, results.Steps, lm))
	}

	// Find the closing </hashTree> under TestPlan and insert before the last two </hashTree> closings
	content := string(data)

	// Check for existing test fragments
	for _, sr := range results.ScenarioResults {
		fragName := "Test Fragment - " + sr.Scenario.Name
		if strings.Contains(content, fragName) {
			slog.Info("found matching TestFragment", "name", fragName)
		} else {
			slog.Warn("no matching TestFragment found", "scenario", sr.Scenario.Name)
		}
	}

	// Insert before the closing of the TestPlan hashTree
	// Find the last occurrence of "    </hashTree>" which closes the TestPlan hashTree
	insertPoint := strings.LastIndex(content, "    </hashTree>")
	if insertPoint < 0 {
		return nil, fmt.Errorf("could not find TestPlan hashTree in template")
	}

	var result bytes.Buffer
	result.WriteString(content[:insertPoint])
	result.WriteString(inject.String())
	result.WriteString(content[insertPoint:])

	return result.Bytes(), nil
}

// UpdateExistingJMX updates existing ThreadGroups in-place and appends new ones.
func UpdateExistingJMX(templatePath string, results engine.CalculationResults) ([]byte, error) {
	data, err := os.ReadFile(templatePath) //nolint:gosec // user-provided file path
	if err != nil {
		return nil, fmt.Errorf("reading template: %w", err)
	}

	content := string(data)
	var toAppend []engine.ScenarioResult

	for _, sr := range results.ScenarioResults {
		lm := getLoadModel(sr, results.Plan.GlobalDefaults.LoadModel)
		name := sr.Scenario.Name

		// Priority 1: Prefix match (STG_, UTG_, FFATG_)
		found := false
		prefixes := []string{"STG_", "UTG_", "FFATG_"}
		for _, prefix := range prefixes {
			prefixedName := prefix + name
			idx := findThreadGroupByName(content, prefixedName)
			if idx >= 0 {
				slog.Info("matched by prefix", "scenario", name, "threadgroup", prefixedName, "prefix", prefix)
				content = replaceThreadGroup(content, prefixedName, sr, results.Steps, lm)
				found = true
				break
			}
		}

		// Priority 2: Exact name match
		if !found {
			idx := findThreadGroupByName(content, name)
			if idx >= 0 {
				slog.Info("matched by exact name", "scenario", name, "threadgroup", name)
				content = replaceThreadGroup(content, name, sr, results.Steps, lm)
				found = true
			}
		}

		// Priority 3: Not found → create new
		if !found {
			slog.Warn("no matching ThreadGroup found, creating new", "scenario", name)
			toAppend = append(toAppend, sr)
		}
	}

	if len(toAppend) > 0 {
		var inject bytes.Buffer
		for _, sr := range toAppend {
			lm := getLoadModel(sr, results.Plan.GlobalDefaults.LoadModel)
			inject.WriteString(buildThreadGroup(sr, results.Steps, lm))
		}
		insertPoint := strings.LastIndex(content, "    </hashTree>")
		if insertPoint < 0 {
			return nil, fmt.Errorf("could not find TestPlan hashTree in template")
		}
		var result bytes.Buffer
		result.WriteString(content[:insertPoint])
		result.WriteString(inject.String())
		result.WriteString(content[insertPoint:])
		content = result.String()
	}

	return []byte(content), nil
}

// findThreadGroupByName returns the index of testname="name" for an enabled ThreadGroup element.
// It only matches elements that have enabled="true".
func findThreadGroupByName(content, name string) int {
	search := fmt.Sprintf(`testname=%q`, name)
	offset := 0
	for {
		idx := strings.Index(content[offset:], search)
		if idx < 0 {
			return -1
		}
		absIdx := offset + idx
		// Find the start of the element's opening tag to check enabled attribute
		elemStart := absIdx
		for elemStart > 0 && content[elemStart] != '<' {
			elemStart--
		}
		// Find end of this opening tag
		tagEnd := strings.Index(content[elemStart:], ">")
		if tagEnd < 0 {
			return -1
		}
		openingTag := content[elemStart : elemStart+tagEnd+1]
		if strings.Contains(openingTag, `enabled="true"`) {
			return absIdx
		}
		// Skip past this match and keep searching
		offset = absIdx + len(search)
	}
}

// findElementBounds locates the start-of-line and end-of-closing-tag for an XML element
// identified by testname at idx in content. Returns lineStart, afterElem, ok.
func findElementBounds(content string, idx int) (lineStart, afterElem int, ok bool) {
	// Scan backward to find the '<' that starts this element
	elemStart := idx
	for elemStart > 0 && content[elemStart] != '<' {
		elemStart--
	}
	// Include leading whitespace on this line
	lineStart = elemStart
	for lineStart > 0 && content[lineStart-1] != '\n' {
		lineStart--
	}

	// Extract the element tag name
	tagEnd := strings.IndexByte(content[elemStart+1:], ' ')
	if tagEnd < 0 {
		return 0, 0, false
	}
	elemName := content[elemStart+1 : elemStart+1+tagEnd]

	// Find the end of this element: either </elemName> or self-closing />
	closingTag := fmt.Sprintf("</%s>", elemName)
	ci := strings.Index(content[idx:], closingTag)
	if ci >= 0 {
		afterElem = idx + ci + len(closingTag)
	} else {
		sc := strings.Index(content[idx:], "/>")
		if sc < 0 {
			return 0, 0, false
		}
		afterElem = idx + sc + 2
	}
	return lineStart, afterElem, true
}

// findHashTreeEnd finds the end of the companion hashTree element following afterElem.
func findHashTreeEnd(content string, afterElem int) int {
	hashStart := afterElem
	for hashStart < len(content) && (content[hashStart] == ' ' || content[hashStart] == '\t' || content[hashStart] == '\n' || content[hashStart] == '\r') {
		hashStart++
	}

	hashEnd := hashStart
	if strings.HasPrefix(content[hashStart:], "<hashTree/>") {
		hashEnd = hashStart + len("<hashTree/>")
	} else if strings.HasPrefix(content[hashStart:], "<hashTree>") {
		ce := strings.Index(content[hashStart:], "</hashTree>")
		if ce >= 0 {
			hashEnd = hashStart + ce + len("</hashTree>")
		}
	}
	// Include trailing newline
	if hashEnd < len(content) && content[hashEnd] == '\n' {
		hashEnd++
	}
	return hashEnd
}

// replaceThreadGroup replaces an existing ThreadGroup element + its following hashTree with a new one.
func replaceThreadGroup(content, name string, sr engine.ScenarioResult, steps []profile.Step, lm config.LoadModel) string {
	search := fmt.Sprintf(`testname=%q`, name)
	idx := strings.Index(content, search)
	if idx < 0 {
		return content
	}

	lineStart, afterElem, ok := findElementBounds(content, idx)
	if !ok {
		return content
	}

	hashEnd := findHashTreeEnd(content, afterElem)

	replacement := buildThreadGroup(sr, steps, lm)
	return content[:lineStart] + replacement + content[hashEnd:]
}
