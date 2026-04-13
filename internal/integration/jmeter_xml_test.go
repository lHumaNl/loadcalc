package integration

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"loadcalc/internal/config"
	"loadcalc/internal/engine"
	"loadcalc/internal/profile"
)

func openModel() *config.LoadModel {
	m := config.LoadModelOpen
	return &m
}

func makeResults(scenarios []config.Scenario, scenarioResults []engine.ScenarioResult, steps []profile.Step) engine.CalculationResults {
	plan := &config.TestPlan{
		Version: "1.0",
		GlobalDefaults: config.GlobalDefaults{
			Tool:      config.ToolJMeter,
			LoadModel: config.LoadModelClosed,
		},
		Scenarios: scenarios,
	}
	return engine.CalculationResults{
		Plan:            plan,
		ScenarioResults: scenarioResults,
		Steps:           steps,
	}
}

// uniformSteps returns steps with equal percent increments.
func uniformSteps() []profile.Step {
	return []profile.Step{
		{StepNumber: 1, PercentOfTarget: 50, RampupSec: 60, StabilitySec: 300},
		{StepNumber: 2, PercentOfTarget: 100, RampupSec: 60, StabilitySec: 300},
		{StepNumber: 3, PercentOfTarget: 150, RampupSec: 60, StabilitySec: 300},
	}
}

// nonUniformSteps returns steps with unequal percent increments.
func nonUniformSteps() []profile.Step {
	return []profile.Step{
		{StepNumber: 1, PercentOfTarget: 50, RampupSec: 60, StabilitySec: 300},
		{StepNumber: 2, PercentOfTarget: 100, RampupSec: 60, StabilitySec: 300},
		{StepNumber: 3, PercentOfTarget: 120, RampupSec: 60, StabilitySec: 300},
	}
}

func closedScenarioResult(name string, steps []profile.Step) (config.Scenario, engine.ScenarioResult) {
	sc := config.Scenario{Name: name, TargetIntensity: 100, MaxScriptTimeMs: 1000}
	var stepResults []engine.StepResult
	for _, s := range steps {
		threads := int(s.PercentOfTarget / 10)
		if threads < 1 {
			threads = 1
		}
		stepResults = append(stepResults, engine.StepResult{
			Step:      s,
			TargetRPS: s.PercentOfTarget,
			Threads:   threads,
			ActualRPS: s.PercentOfTarget,
		})
	}
	sr := engine.ScenarioResult{
		Scenario:    sc,
		IsOpenModel: false,
		OptimizeResult: engine.OptimizeResult{
			BestPacingMS:           3000,
			BestOpsPerMinPerThread: 20,
			StepResults:            stepResults,
		},
	}
	return sc, sr
}

func openScenarioResult(name string, steps []profile.Step) (config.Scenario, engine.ScenarioResult) {
	sc := config.Scenario{Name: name, TargetIntensity: 100, MaxScriptTimeMs: 500}
	sc.LoadModel = openModel()
	var stepResults []engine.StepResult
	for _, s := range steps {
		stepResults = append(stepResults, engine.StepResult{
			Step:      s,
			TargetRPS: s.PercentOfTarget,
			Threads:   0,
			ActualRPS: s.PercentOfTarget,
		})
	}
	sr := engine.ScenarioResult{
		Scenario:    sc,
		IsOpenModel: true,
		OptimizeResult: engine.OptimizeResult{
			StepResults: stepResults,
		},
	}
	return sc, sr
}

// --- selectThreadGroupType tests ---

func TestSelectThreadGroupType_OpenModel(t *testing.T) {
	got := selectThreadGroupType(uniformSteps(), config.LoadModelOpen)
	if got != "FreeFormArrivalsThreadGroup" {
		t.Errorf("expected FreeFormArrivalsThreadGroup, got %s", got)
	}
}

func TestSelectThreadGroupType_ClosedUniform(t *testing.T) {
	got := selectThreadGroupType(uniformSteps(), config.LoadModelClosed)
	if got != "SteppingThreadGroup" {
		t.Errorf("expected SteppingThreadGroup, got %s", got)
	}
}

func TestSelectThreadGroupType_ClosedNonUniform(t *testing.T) {
	got := selectThreadGroupType(nonUniformSteps(), config.LoadModelClosed)
	if got != "UltimateThreadGroup" {
		t.Errorf("expected UltimateThreadGroup, got %s", got)
	}
}

func TestSelectThreadGroupType_SingleStep(t *testing.T) {
	steps := []profile.Step{{StepNumber: 1, PercentOfTarget: 100, RampupSec: 60, StabilitySec: 300}}
	got := selectThreadGroupType(steps, config.LoadModelClosed)
	if got != "SteppingThreadGroup" {
		t.Errorf("expected SteppingThreadGroup for single step, got %s", got)
	}
}

// --- GenerateJMX tests ---

func TestGenerateJMX_ValidXML(t *testing.T) {
	steps := uniformSteps()
	sc, sr := closedScenarioResult("Main page", steps)
	results := makeResults([]config.Scenario{sc}, []engine.ScenarioResult{sr}, steps)

	data, err := GenerateJMX(results)
	if err != nil {
		t.Fatalf("GenerateJMX error: %v", err)
	}

	// Must be valid XML
	if !strings.HasPrefix(string(data), "<?xml") {
		t.Error("output should start with XML declaration")
	}
	var dummy interface{}
	if err := xml.Unmarshal(data, &dummy); err != nil {
		t.Errorf("output is not valid XML: %v", err)
	}
}

func TestGenerateJMX_SteppingThreadGroup(t *testing.T) {
	steps := uniformSteps()
	sc, sr := closedScenarioResult("Main page", steps)
	results := makeResults([]config.Scenario{sc}, []engine.ScenarioResult{sr}, steps)

	data, err := GenerateJMX(results)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "kg.apc.jmeter.threads.SteppingThreadGroup") {
		t.Error("expected SteppingThreadGroup in output")
	}
	// Max threads should be from the last step
	if !strings.Contains(s, `"ThreadGroup.num_threads">15`) {
		t.Errorf("expected max threads 15 in output, got:\n%s", s)
	}
}

func TestGenerateJMX_UltimateThreadGroup(t *testing.T) {
	steps := nonUniformSteps()
	sc, sr := closedScenarioResult("API call", steps)
	results := makeResults([]config.Scenario{sc}, []engine.ScenarioResult{sr}, steps)

	data, err := GenerateJMX(results)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "kg.apc.jmeter.threads.UltimateThreadGroup") {
		t.Error("expected UltimateThreadGroup in output")
	}
	if !strings.Contains(s, "ultimatethreadgroupdata") {
		t.Error("expected ultimatethreadgroupdata collection")
	}
}

func TestGenerateJMX_OpenModel_FFATG(t *testing.T) {
	steps := uniformSteps()
	sc, sr := openScenarioResult("API health", steps)
	plan := &config.TestPlan{
		Version: "1.0",
		GlobalDefaults: config.GlobalDefaults{
			Tool:      config.ToolJMeter,
			LoadModel: config.LoadModelClosed,
		},
		Scenarios: []config.Scenario{sc},
	}
	results := engine.CalculationResults{
		Plan:            plan,
		ScenarioResults: []engine.ScenarioResult{sr},
		Steps:           steps,
	}

	data, err := GenerateJMX(results)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "FreeFormArrivalsThreadGroup") {
		t.Error("expected FreeFormArrivalsThreadGroup in output")
	}
	if !strings.Contains(s, "FFATG_API health") {
		t.Error("expected FFATG_ prefix in testname for open model")
	}
}

func TestGenerateJMX_ConstantThroughputTimer(t *testing.T) {
	steps := uniformSteps()
	sc, sr := closedScenarioResult("Main page", steps)
	results := makeResults([]config.Scenario{sc}, []engine.ScenarioResult{sr}, steps)

	data, err := GenerateJMX(results)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "ConstantThroughputTimer") {
		t.Error("expected ConstantThroughputTimer for closed model")
	}
	if !strings.Contains(s, "<name>throughput</name>") {
		t.Error("expected throughput doubleProp")
	}
}

func TestGenerateJMX_NoThroughputTimerForOpenModel(t *testing.T) {
	steps := uniformSteps()
	sc, sr := openScenarioResult("API health", steps)
	plan := &config.TestPlan{
		Version:        "1.0",
		GlobalDefaults: config.GlobalDefaults{Tool: config.ToolJMeter, LoadModel: config.LoadModelOpen},
		Scenarios:      []config.Scenario{sc},
	}
	results := engine.CalculationResults{
		Plan:            plan,
		ScenarioResults: []engine.ScenarioResult{sr},
		Steps:           steps,
	}

	data, err := GenerateJMX(results)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if strings.Contains(s, "ConstantThroughputTimer") {
		t.Error("should not have ConstantThroughputTimer for open model")
	}
}

func TestGenerateJMX_ModuleController(t *testing.T) {
	steps := uniformSteps()
	sc, sr := closedScenarioResult("Main page", steps)
	results := makeResults([]config.Scenario{sc}, []engine.ScenarioResult{sr}, steps)

	data, err := GenerateJMX(results)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "ModuleController") {
		t.Error("expected ModuleController")
	}
	if !strings.Contains(s, "Module: Main page") {
		t.Error("expected Module: Main page testname")
	}
}

func TestGenerateJMX_TestFragment(t *testing.T) {
	steps := uniformSteps()
	sc, sr := closedScenarioResult("Main page", steps)
	results := makeResults([]config.Scenario{sc}, []engine.ScenarioResult{sr}, steps)

	data, err := GenerateJMX(results)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "TestFragmentController") {
		t.Error("expected TestFragmentController")
	}
}

// --- InjectIntoJMX tests ---

func testdataPath(name string) string {
	return filepath.Join("testdata", name)
}

func TestInjectIntoJMX_PreservesExisting(t *testing.T) {
	steps := uniformSteps()
	sc, sr := closedScenarioResult("New scenario", steps)
	results := makeResults([]config.Scenario{sc}, []engine.ScenarioResult{sr}, steps)

	data, err := InjectIntoJMX(testdataPath("template.jmx"), results)
	if err != nil {
		t.Fatalf("InjectIntoJMX error: %v", err)
	}
	s := string(data)
	// Original TG preserved
	if !strings.Contains(s, "Existing TG") {
		t.Error("original ThreadGroup should be preserved")
	}
	// New TG appended
	if !strings.Contains(s, "New scenario") {
		t.Error("new ThreadGroup should be appended")
	}
}

func TestInjectIntoJMX_ValidXML(t *testing.T) {
	steps := uniformSteps()
	sc, sr := closedScenarioResult("Injected", steps)
	results := makeResults([]config.Scenario{sc}, []engine.ScenarioResult{sr}, steps)

	data, err := InjectIntoJMX(testdataPath("template.jmx"), results)
	if err != nil {
		t.Fatal(err)
	}
	var dummy interface{}
	if err := xml.Unmarshal(data, &dummy); err != nil {
		t.Errorf("output is not valid XML: %v", err)
	}
}

// --- UpdateExistingJMX tests ---

func TestUpdateExistingJMX_ExactNameMatchWhenNoPrefixExists(t *testing.T) {
	// Template has exact name "Main page" (no prefix) - should match by exact name
	tmpl := `<?xml version="1.0" encoding="UTF-8"?>
<jmeterTestPlan version="1.2" properties="5.0" jmeter="5.6.3">
  <hashTree>
    <TestPlan guiclass="TestPlanGui" testclass="TestPlan" testname="Test Plan" enabled="true"/>
    <hashTree>
      <kg.apc.jmeter.threads.SteppingThreadGroup guiclass="kg.apc.jmeter.threads.SteppingThreadGroupGui" testclass="kg.apc.jmeter.threads.SteppingThreadGroup" testname="Main page" enabled="true">
        <stringProp name="ThreadGroup.num_threads">5</stringProp>
      </kg.apc.jmeter.threads.SteppingThreadGroup>
      <hashTree/>
    </hashTree>
  </hashTree>
</jmeterTestPlan>`
	tmpFile := filepath.Join(t.TempDir(), "exact.jmx")
	os.WriteFile(tmpFile, []byte(tmpl), 0o644)

	steps := uniformSteps()
	sc, sr := closedScenarioResult("Main page", steps)
	results := makeResults([]config.Scenario{sc}, []engine.ScenarioResult{sr}, steps)

	data, err := UpdateExistingJMX(tmpFile, results)
	if err != nil {
		t.Fatalf("UpdateExistingJMX error: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "Main page") {
		t.Error("should contain Main page")
	}
	if strings.Contains(s, `"ThreadGroup.num_threads">5<`) {
		t.Error("old thread count should be updated")
	}
}

func TestUpdateExistingJMX_PrefixMatchTakesPriorityOverExactName(t *testing.T) {
	// Template has both "Main page" and "STG_Main page" - prefix should win
	tmpl := `<?xml version="1.0" encoding="UTF-8"?>
<jmeterTestPlan version="1.2" properties="5.0" jmeter="5.6.3">
  <hashTree>
    <TestPlan guiclass="TestPlanGui" testclass="TestPlan" testname="Test Plan" enabled="true"/>
    <hashTree>
      <kg.apc.jmeter.threads.SteppingThreadGroup guiclass="kg.apc.jmeter.threads.SteppingThreadGroupGui" testclass="kg.apc.jmeter.threads.SteppingThreadGroup" testname="Main page" enabled="true">
        <stringProp name="ThreadGroup.num_threads">3</stringProp>
      </kg.apc.jmeter.threads.SteppingThreadGroup>
      <hashTree/>
      <kg.apc.jmeter.threads.SteppingThreadGroup guiclass="kg.apc.jmeter.threads.SteppingThreadGroupGui" testclass="kg.apc.jmeter.threads.SteppingThreadGroup" testname="STG_Main page" enabled="true">
        <stringProp name="ThreadGroup.num_threads">5</stringProp>
      </kg.apc.jmeter.threads.SteppingThreadGroup>
      <hashTree/>
    </hashTree>
  </hashTree>
</jmeterTestPlan>`
	tmpFile := filepath.Join(t.TempDir(), "priority.jmx")
	os.WriteFile(tmpFile, []byte(tmpl), 0o644)

	steps := uniformSteps()
	sc, sr := closedScenarioResult("Main page", steps)
	results := makeResults([]config.Scenario{sc}, []engine.ScenarioResult{sr}, steps)

	data, err := UpdateExistingJMX(tmpFile, results)
	if err != nil {
		t.Fatalf("UpdateExistingJMX error: %v", err)
	}
	s := string(data)
	// STG_ prefix version should be updated (old value 5 gone)
	if strings.Contains(s, `"ThreadGroup.num_threads">5<`) {
		t.Error("STG_ prefixed TG should have been updated (old value 5 replaced)")
	}
	// Exact name "Main page" with value 3 should remain untouched
	if !strings.Contains(s, `"ThreadGroup.num_threads">3<`) {
		t.Error("exact name TG should remain untouched since prefix match took priority")
	}
}

func TestUpdateExistingJMX_DisabledTGSkipped(t *testing.T) {
	// Template has a disabled TG with matching name - should be skipped
	tmpl := `<?xml version="1.0" encoding="UTF-8"?>
<jmeterTestPlan version="1.2" properties="5.0" jmeter="5.6.3">
  <hashTree>
    <TestPlan guiclass="TestPlanGui" testclass="TestPlan" testname="Test Plan" enabled="true"/>
    <hashTree>
      <kg.apc.jmeter.threads.SteppingThreadGroup guiclass="kg.apc.jmeter.threads.SteppingThreadGroupGui" testclass="kg.apc.jmeter.threads.SteppingThreadGroup" testname="STG_Main page" enabled="false">
        <stringProp name="ThreadGroup.num_threads">5</stringProp>
      </kg.apc.jmeter.threads.SteppingThreadGroup>
      <hashTree/>
    </hashTree>
  </hashTree>
</jmeterTestPlan>`
	tmpFile := filepath.Join(t.TempDir(), "disabled.jmx")
	os.WriteFile(tmpFile, []byte(tmpl), 0o644)

	steps := uniformSteps()
	sc, sr := closedScenarioResult("Main page", steps)
	results := makeResults([]config.Scenario{sc}, []engine.ScenarioResult{sr}, steps)

	data, err := UpdateExistingJMX(tmpFile, results)
	if err != nil {
		t.Fatalf("UpdateExistingJMX error: %v", err)
	}
	s := string(data)
	// Disabled TG should remain untouched
	if !strings.Contains(s, `testname="STG_Main page" enabled="false"`) {
		t.Error("disabled TG should remain untouched")
	}
	// A new TG should be appended
	if !strings.Contains(s, `testname="STG_Main page" enabled="true"`) {
		t.Error("new TG should be created since disabled TG was skipped")
	}
}

func TestUpdateExistingJMX_NotFoundCreatesNew(t *testing.T) {
	// Template has no matching TG at all
	tmpl := `<?xml version="1.0" encoding="UTF-8"?>
<jmeterTestPlan version="1.2" properties="5.0" jmeter="5.6.3">
  <hashTree>
    <TestPlan guiclass="TestPlanGui" testclass="TestPlan" testname="Test Plan" enabled="true"/>
    <hashTree>
      <kg.apc.jmeter.threads.SteppingThreadGroup guiclass="kg.apc.jmeter.threads.SteppingThreadGroupGui" testclass="kg.apc.jmeter.threads.SteppingThreadGroup" testname="Other scenario" enabled="true">
        <stringProp name="ThreadGroup.num_threads">5</stringProp>
      </kg.apc.jmeter.threads.SteppingThreadGroup>
      <hashTree/>
    </hashTree>
  </hashTree>
</jmeterTestPlan>`
	tmpFile := filepath.Join(t.TempDir(), "notfound.jmx")
	os.WriteFile(tmpFile, []byte(tmpl), 0o644)

	steps := uniformSteps()
	sc, sr := closedScenarioResult("Main page", steps)
	results := makeResults([]config.Scenario{sc}, []engine.ScenarioResult{sr}, steps)

	data, err := UpdateExistingJMX(tmpFile, results)
	if err != nil {
		t.Fatalf("UpdateExistingJMX error: %v", err)
	}
	s := string(data)
	// Original should be preserved
	if !strings.Contains(s, "Other scenario") {
		t.Error("existing TG should be preserved")
	}
	// New TG should be appended
	if !strings.Contains(s, "STG_Main page") {
		t.Error("new TG should be created for unmatched scenario")
	}
}

func TestUpdateExistingJMX_UpdatesExisting(t *testing.T) {
	// Create a template with a ThreadGroup named "Main page"
	tmpl := `<?xml version="1.0" encoding="UTF-8"?>
<jmeterTestPlan version="1.2" properties="5.0" jmeter="5.6.3">
  <hashTree>
    <TestPlan guiclass="TestPlanGui" testclass="TestPlan" testname="Test Plan" enabled="true"/>
    <hashTree>
      <kg.apc.jmeter.threads.SteppingThreadGroup guiclass="kg.apc.jmeter.threads.SteppingThreadGroupGui" testclass="kg.apc.jmeter.threads.SteppingThreadGroup" testname="Main page" enabled="true">
        <stringProp name="ThreadGroup.num_threads">5</stringProp>
      </kg.apc.jmeter.threads.SteppingThreadGroup>
      <hashTree/>
    </hashTree>
  </hashTree>
</jmeterTestPlan>`
	tmpFile := filepath.Join(t.TempDir(), "update.jmx")
	os.WriteFile(tmpFile, []byte(tmpl), 0o644)

	steps := uniformSteps()
	sc, sr := closedScenarioResult("Main page", steps)
	results := makeResults([]config.Scenario{sc}, []engine.ScenarioResult{sr}, steps)

	data, err := UpdateExistingJMX(tmpFile, results)
	if err != nil {
		t.Fatalf("UpdateExistingJMX error: %v", err)
	}
	s := string(data)
	// Should still contain Main page but with updated thread count
	if !strings.Contains(s, "Main page") {
		t.Error("should still contain Main page")
	}
	// Old value 5 should be replaced
	if strings.Contains(s, `"ThreadGroup.num_threads">5<`) {
		t.Error("old thread count should be updated")
	}
}

func TestUpdateExistingJMX_AppendsNew(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "update2.jmx")
	tmpl, _ := os.ReadFile(testdataPath("template.jmx"))
	os.WriteFile(tmpFile, tmpl, 0o644)

	steps := uniformSteps()
	sc, sr := closedScenarioResult("Brand new", steps)
	results := makeResults([]config.Scenario{sc}, []engine.ScenarioResult{sr}, steps)

	data, err := UpdateExistingJMX(tmpFile, results)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "Brand new") {
		t.Error("new scenario should be appended")
	}
	if !strings.Contains(s, "Existing TG") {
		t.Error("existing TG should be preserved")
	}
}

func TestUpdateExistingJMX_FFATGPrefixMatch(t *testing.T) {
	tmpl := `<?xml version="1.0" encoding="UTF-8"?>
<jmeterTestPlan version="1.2" properties="5.0" jmeter="5.6.3">
  <hashTree>
    <TestPlan guiclass="TestPlanGui" testclass="TestPlan" testname="Test Plan" enabled="true"/>
    <hashTree>
      <com.blazemeter.jmeter.threads.arrivals.FreeFormArrivalsThreadGroup guiclass="com.blazemeter.jmeter.threads.arrivals.FreeFormArrivalsThreadGroupGui" testclass="com.blazemeter.jmeter.threads.arrivals.FreeFormArrivalsThreadGroup" testname="FFATG_API health" enabled="true">
        <stringProp name="ConcurrencyLimit">50</stringProp>
      </com.blazemeter.jmeter.threads.arrivals.FreeFormArrivalsThreadGroup>
      <hashTree/>
    </hashTree>
  </hashTree>
</jmeterTestPlan>`
	tmpFile := filepath.Join(t.TempDir(), "ffatg.jmx")
	os.WriteFile(tmpFile, []byte(tmpl), 0o644)

	steps := uniformSteps()
	sc, sr := openScenarioResult("API health", steps)
	plan := &config.TestPlan{
		Version:        "1.0",
		GlobalDefaults: config.GlobalDefaults{Tool: config.ToolJMeter, LoadModel: config.LoadModelClosed},
		Scenarios:      []config.Scenario{sc},
	}
	results := engine.CalculationResults{Plan: plan, ScenarioResults: []engine.ScenarioResult{sr}, Steps: steps}

	data, err := UpdateExistingJMX(tmpFile, results)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	// Should match by FFATG_ prefix and update
	if !strings.Contains(s, "FFATG_API health") {
		t.Error("should preserve FFATG_ prefix name")
	}
	// Old concurrency limit should be gone
	if strings.Contains(s, `"ConcurrencyLimit">50<`) {
		t.Error("old ConcurrencyLimit should be updated")
	}
}

func TestUpdateExistingJMX_STGPrefixMatch(t *testing.T) {
	tmpl := `<?xml version="1.0" encoding="UTF-8"?>
<jmeterTestPlan version="1.2" properties="5.0" jmeter="5.6.3">
  <hashTree>
    <TestPlan guiclass="TestPlanGui" testclass="TestPlan" testname="Test Plan" enabled="true"/>
    <hashTree>
      <kg.apc.jmeter.threads.SteppingThreadGroup guiclass="kg.apc.jmeter.threads.SteppingThreadGroupGui" testclass="kg.apc.jmeter.threads.SteppingThreadGroup" testname="STG_Main page" enabled="true">
        <stringProp name="ThreadGroup.num_threads">5</stringProp>
      </kg.apc.jmeter.threads.SteppingThreadGroup>
      <hashTree/>
    </hashTree>
  </hashTree>
</jmeterTestPlan>`
	tmpFile := filepath.Join(t.TempDir(), "stg.jmx")
	os.WriteFile(tmpFile, []byte(tmpl), 0o644)

	steps := uniformSteps()
	sc, sr := closedScenarioResult("Main page", steps)
	results := makeResults([]config.Scenario{sc}, []engine.ScenarioResult{sr}, steps)

	data, err := UpdateExistingJMX(tmpFile, results)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	// Should match by STG_ prefix and update
	if !strings.Contains(s, "STG_Main page") {
		t.Error("should preserve STG_ prefix name")
	}
	// Old thread count should be gone
	if strings.Contains(s, `"ThreadGroup.num_threads">5<`) {
		t.Error("old thread count should be updated")
	}
}

func TestUpdateExistingJMX_UTGPrefixMatch(t *testing.T) {
	tmpl := `<?xml version="1.0" encoding="UTF-8"?>
<jmeterTestPlan version="1.2" properties="5.0" jmeter="5.6.3">
  <hashTree>
    <TestPlan guiclass="TestPlanGui" testclass="TestPlan" testname="Test Plan" enabled="true"/>
    <hashTree>
      <kg.apc.jmeter.threads.UltimateThreadGroup guiclass="kg.apc.jmeter.threads.UltimateThreadGroupGui" testclass="kg.apc.jmeter.threads.UltimateThreadGroup" testname="UTG_API call" enabled="true">
        <collectionProp name="ultimatethreadgroupdata">
          <collectionProp name="row_0">
            <stringProp name="0">3</stringProp>
          </collectionProp>
        </collectionProp>
      </kg.apc.jmeter.threads.UltimateThreadGroup>
      <hashTree/>
    </hashTree>
  </hashTree>
</jmeterTestPlan>`
	tmpFile := filepath.Join(t.TempDir(), "utg.jmx")
	os.WriteFile(tmpFile, []byte(tmpl), 0o644)

	steps := nonUniformSteps()
	sc, sr := closedScenarioResult("API call", steps)
	results := makeResults([]config.Scenario{sc}, []engine.ScenarioResult{sr}, steps)

	data, err := UpdateExistingJMX(tmpFile, results)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	// Should match by UTG_ prefix and update
	if !strings.Contains(s, "UTG_API call") {
		t.Error("should preserve UTG_ prefix name")
	}
}

func TestBuildThreadGroup_STGPrefix(t *testing.T) {
	steps := uniformSteps()
	_, sr := closedScenarioResult("Main page", steps)
	// SteppingTG with closed model should get STG_ prefix
	tgXML := buildThreadGroup(sr, steps, config.LoadModelClosed)
	if !strings.Contains(tgXML, `testname="STG_Main page"`) {
		t.Error("SteppingThreadGroup should have STG_ prefix")
	}
}

func TestBuildThreadGroup_UTGPrefix(t *testing.T) {
	steps := nonUniformSteps()
	_, sr := closedScenarioResult("API call", steps)
	// UltimateTG with closed model should get UTG_ prefix
	tgXML := buildThreadGroup(sr, steps, config.LoadModelClosed)
	if !strings.Contains(tgXML, `testname="UTG_API call"`) {
		t.Error("UltimateThreadGroup should have UTG_ prefix")
	}
}

func TestGenerateJMX_MultipleScenarios(t *testing.T) {
	steps := uniformSteps()
	sc1, sr1 := closedScenarioResult("Main page", steps)
	sc2, sr2 := openScenarioResult("API health", steps)
	plan := &config.TestPlan{
		Version:        "1.0",
		GlobalDefaults: config.GlobalDefaults{Tool: config.ToolJMeter, LoadModel: config.LoadModelClosed},
		Scenarios:      []config.Scenario{sc1, sc2},
	}
	results := engine.CalculationResults{
		Plan:            plan,
		ScenarioResults: []engine.ScenarioResult{sr1, sr2},
		Steps:           steps,
	}

	data, err := GenerateJMX(results)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "Main page") {
		t.Error("should contain Main page")
	}
	if !strings.Contains(s, "FFATG_API health") {
		t.Error("should contain FFATG_API health")
	}
}
