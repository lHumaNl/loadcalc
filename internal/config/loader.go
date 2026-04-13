package config

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"loadcalc/pkg/units"

	"github.com/xuri/excelize/v2"
	"gopkg.in/yaml.v3"
)

const defaultVersion = "1.0"

// LoadFromYAML parses a YAML file into a TestPlan struct.
func LoadFromYAML(path string) (*TestPlan, error) {
	data, err := os.ReadFile(path) //nolint:gosec // user-provided file path
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var plan TestPlan
	if err := yaml.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if plan.Version == "" {
		plan.Version = defaultVersion
	}

	return &plan, nil
}

// LoadScenariosFromFile loads scenarios from a CSV or XLSX file based on extension.
func LoadScenariosFromFile(path string, csvDelimiter rune) ([]Scenario, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".csv":
		return LoadScenariosFromCSV(path, csvDelimiter)
	case ".xlsx":
		return LoadScenariosFromXLSX(path)
	default:
		return nil, fmt.Errorf("unsupported scenario file extension: %s", ext)
	}
}

// LoadScenariosFromCSV parses a CSV file into a slice of Scenario structs.
func LoadScenariosFromCSV(path string, delimiter rune) ([]Scenario, error) {
	f, err := os.Open(path) //nolint:gosec // user-provided file path
	if err != nil {
		return nil, fmt.Errorf("opening CSV file: %w", err)
	}
	defer func() { _ = f.Close() }()

	reader := csv.NewReader(f)
	reader.Comma = delimiter
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading CSV: %w", err)
	}

	if len(records) < 1 {
		return nil, nil
	}

	// Build column index from header row
	header := records[0]
	colIdx := make(map[string]int)
	for i, h := range header {
		colIdx[strings.TrimSpace(strings.ToLower(h))] = i
	}

	var scenarios []Scenario
	for _, row := range records[1:] {
		s, err := rowToScenario(row, colIdx)
		if err != nil {
			return nil, fmt.Errorf("parsing CSV row: %w", err)
		}
		scenarios = append(scenarios, s)
	}

	return scenarios, nil
}

// LoadScenariosFromXLSX reads scenarios from the first sheet or a sheet named "Scenarios".
func LoadScenariosFromXLSX(path string) ([]Scenario, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("opening XLSX file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Try "Scenarios" sheet first, fall back to first sheet
	sheetName := "Scenarios"
	sheets := f.GetSheetList()
	found := false
	for _, s := range sheets {
		if s == sheetName {
			found = true
			break
		}
	}
	if !found && len(sheets) > 0 {
		sheetName = sheets[0]
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("reading XLSX sheet %q: %w", sheetName, err)
	}

	if len(rows) < 1 {
		return nil, nil
	}

	header := rows[0]
	colIdx := make(map[string]int)
	for i, h := range header {
		colIdx[strings.TrimSpace(strings.ToLower(h))] = i
	}

	var scenarios []Scenario
	for _, row := range rows[1:] {
		s, err := rowToScenario(row, colIdx)
		if err != nil {
			return nil, fmt.Errorf("parsing XLSX row: %w", err)
		}
		scenarios = append(scenarios, s)
	}

	return scenarios, nil
}

// LoadScenariosFromDir loads all .csv and .xlsx files from a directory, sorted by name.
func LoadScenariosFromDir(dir string, csvDelimiter rune) ([]Scenario, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".csv" || ext == ".xlsx" {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	var all []Scenario
	for _, name := range files {
		path := filepath.Join(dir, name)
		scenarios, err := LoadScenariosFromFile(path, csvDelimiter)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", name, err)
		}
		all = append(all, scenarios...)
	}

	return all, nil
}

func rowToScenario(row []string, colIdx map[string]int) (Scenario, error) {
	var s Scenario

	s.Name = getCell(row, colIdx, "name")
	if s.Name == "" {
		return s, fmt.Errorf("scenario name is empty")
	}
	s.ScriptID = getIntCell(row, colIdx, "script_id")
	s.TargetIntensity = getFloatCell(row, colIdx, "target_intensity")
	s.IntensityUnit = units.IntensityUnit(getCell(row, colIdx, "intensity_unit"))
	s.MaxScriptTimeMs = getIntCell(row, colIdx, "max_script_time_ms")
	s.Background = getBoolCell(row, colIdx, "background")
	s.BackgroundPercent = getFloatCell(row, colIdx, "background_percent")

	if lm := getCell(row, colIdx, "load_model"); lm != "" {
		model := LoadModel(lm)
		s.LoadModel = &model
	}
	if pm := getCell(row, colIdx, "pacing_multiplier"); pm != "" {
		v, _ := strconv.ParseFloat(pm, 64)
		s.PacingMultiplier = &v
	}
	if dt := getCell(row, colIdx, "deviation_tolerance"); dt != "" {
		v, _ := strconv.ParseFloat(dt, 64)
		s.DeviationTolerance = &v
	}
	if sp := getCell(row, colIdx, "spike_participate"); sp != "" {
		v := parseBool(sp)
		s.SpikeParticipate = &v
	}

	return s, nil
}

func getCell(row []string, colIdx map[string]int, name string) string {
	idx, ok := colIdx[name]
	if !ok || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func getIntCell(row []string, colIdx map[string]int, name string) int {
	s := getCell(row, colIdx, name)
	if s == "" {
		return 0
	}
	// Handle float strings like "1100.0" from XLSX
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return int(f)
	}
	v, _ := strconv.Atoi(s)
	return v
}

func getFloatCell(row []string, colIdx map[string]int, name string) float64 {
	s := getCell(row, colIdx, name)
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func getBoolCell(row []string, colIdx map[string]int, name string) bool {
	return parseBool(getCell(row, colIdx, name))
}

func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes"
}
