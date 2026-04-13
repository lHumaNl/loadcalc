# loadcalc — Load Test Parameter Calculator

## Specification v1.0

---

## 1. Project Overview

### 1.1 Purpose

`loadcalc` is a CLI/TUI utility for calculating load test parameters (threads, pacing, throughput-per-thread) based on target intensity, script execution time, and test profile configuration. It supports multiple load testing tools (LRE Performance Center, JMeter) and multiple test scenarios with per-scenario overrides.

### 1.2 Core Problem

Load testing tools (LRE PC, JMeter) require integer thread counts and fixed pacing/throughput-per-thread values. When a test has multiple intensity steps (e.g., 100% → 200% → 210%), the constraint that **pacing is set once and only thread count changes per step** creates rounding errors. Adding 1 thread may overshoot the target by more than the desired step increment. `loadcalc` solves this by finding optimal pacing that minimizes cumulative deviation across all steps.

### 1.3 Technology Choice

**Language:** Go

**Rationale:**
- Single static binary, no runtime dependencies — critical for air-gapped/restricted environments
- Strong standard library for YAML parsing, CLI, math
- Mature ecosystem: `excelize` (XLSX), `gopkg.in/yaml.v3` (YAML), `bubbletea`/`tview` (TUI)
- Native cross-compilation (linux/darwin/windows, amd64/arm64)

### 1.4 Interface Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| CLI  | Non-interactive, pipe-friendly, structured output | CI/CD, scripting, automated pipelines |
| TUI  | Interactive terminal UI with tables, navigation, color | Manual use, parameter exploration |

### 1.5 Naming

- Binary: `loadcalc`
- Repository: `loadcalc`

---

## 2. Domain Model

### 2.1 Core Concepts

```
TestPlan
├── GlobalDefaults
│   ├── tool: lre_pc | jmeter
│   ├── load_model: closed | open          (default for all scenarios)
│   ├── pacing_multiplier: float           (default: 3.0)
│   ├── deviation_tolerance: float         (default: 2.5%)
│   ├── spike_participate: bool            (default for all scenarios)
│   └── generators_count: int              (JMeter only, default: 1)
│
├── Scenarios{}                            (map keyed by name — names must be unique)
│   ├── name: string                       (required; LRE PC: group name in test; JMeter: display name)
│   ├── script_id: int                     (LRE PC only, required: script ID in Performance Center; ignored for JMeter)
│   ├── target_intensity: float            (target value)
│   ├── intensity_unit: ops_h | ops_m | ops_s
│   ├── max_script_time_ms: int            (max execution time in milliseconds)
│   ├── background: bool                   (default: false)
│   ├── background_percent: float          (% of target, used if background=true)
│   │
│   │── [Per-scenario overrides — if omitted, global default applies]
│   ├── load_model: closed | open          (override)
│   ├── pacing_multiplier: float           (override)
│   ├── deviation_tolerance: float         (override)
│   └── spike_participate: bool            (override)
│
├── TestProfile
│   ├── type: stability | capacity | custom | spike
│   ├── steps[]                            (for capacity/custom, per-step overrides)
│   │   ├── percent_of_target: float
│   │   ├── rampup_sec: int               (optional, override default)
│   │   ├── impact_sec: int               (optional, override default)
│   │   ├── stability_sec: int            (optional, override default)
│   │   └── rampdown_sec: int             (optional, override default; 0 = no rampdown)
│   │
│   ├── [capacity specific — step generation]
│   ├── start_percent: float               (e.g., 50%)
│   ├── step_increment: float              (e.g., 10%)
│   ├── num_steps: int
│   ├── fine_tune:                         (optional — second range with different increment)
│   │   ├── after_percent: float           (switch increment after this step, e.g., 300%)
│   │   ├── step_increment: float          (new increment, e.g., 10%)
│   │   └── num_steps: int                 (number of fine-tune steps)
│   │
│   ├── [custom specific — explicit step list]
│   ├── steps[]                            (arbitrary percent values, any order, repeats allowed)
│   │   └── percent: float                 (e.g., 100, 200, 100, 300, 200)
│   │
│   ├── [spike specific]
│   ├── base_percent: float                (e.g., 70%)
│   ├── spike_start_increment: float       (e.g., 50%)
│   ├── spike_increment_growth: float      (e.g., 10% — each subsequent spike adds this)
│   ├── num_spikes: int
│   ├── spiketime_sec: int                 (duration of each spike)
│   ├── cooldown_sec: int                  (time between spikes at base level)
│   │
│   └── [step timing defaults — applied if step-level timing omitted]
│       ├── default_rampup_sec: int
│       ├── default_impact_sec: int
│       ├── default_stability_sec: int
│       └── default_rampdown_sec: int      (0 = no rampdown phase)
│
└── Output
    └── format: xlsx
```

### 2.2 Calculation Engine

#### 2.2.1 Unit Normalization

All intensities are internally converted to **ops/sec** for calculation, then converted back to the user's preferred unit for display.

```
ops_h → ops/sec: value / 3600
ops_m → ops/sec: value / 60
ops_s → ops/sec: value
```

#### 2.2.2 Closed Model — LRE Performance Center

**Inputs:** `target_rps`, `max_script_time_ms`, `pacing_multiplier`

**Step 1 — Base pacing:**
```
pacing_ms = max_script_time_ms × pacing_multiplier
```

**Step 2 — Ideal threads (float):**
```
ideal_threads = target_rps × (pacing_ms / 1000)
```

**Step 3 — Round threads to integer (minimum 1):**
```
threads = ceil(ideal_threads)   // for single-step profiles
// For multi-step profiles, the optimizer (§2.2.5) tries ceil, floor, and round
// for each step independently and selects the combination minimizing deviation.
// threads is always >= 1 (integer, never zero).
```

**Step 4 — Correct pacing for exact target (single-step only):**
```
corrected_pacing_ms = (threads / target_rps) × 1000
```

> **Note:** For multi-step profiles, Step 4 is replaced by the multi-step pacing optimizer (§2.2.5),
> which finds optimal pacing across all steps simultaneously. The corrected pacing calculation
> is only used for single-step profiles (e.g., `stability` with one step).

**Step 5 — Verify deviation:**
```
actual_rps = threads / (corrected_pacing_ms / 1000)
deviation_percent = |actual_rps - target_rps| / target_rps × 100
```

**Key constraint:** Pacing (or `corrected_pacing_ms`) is fixed once. On subsequent steps, only `threads` changes.

#### 2.2.3 Closed Model — JMeter

**Same as LRE PC for thread calculation, plus:**

**Step 6 — Ops/min per thread (for Constant Throughput Timer):**
```
ops_per_min_per_thread = (target_rps / threads) × 60
```

**Step 7 — Per-generator split:**
```
target_rps_per_generator = target_rps / generators_count
// Recalculate threads and ops_per_min_per_thread with this value
```

**Key constraint:** `ops_per_min_per_thread` is fixed once (same as pacing in LRE PC). On subsequent steps, only `threads` changes.

#### 2.2.4 Open Model — JMeter (Free-Form Arrivals Thread Group)

No thread/pacing calculation needed. Only unit conversion:

```
If target_rps >= 0.01:
    output_unit = ops/sec, value = target_rps
Else:
    output_unit = ops/min, value = target_rps × 60
```

**Rationale:** JMeter's Free-Form Arrivals Thread Group glitches when ops/sec < 0.01. Switching to ops/min avoids this.

Per-generator split also applies: `value_per_generator = value / generators_count`

**Step scaling:** Open model scenarios participate in step/spike scaling the same way as closed model — their `target_rps` is multiplied by the step's `percent / 100`. The only difference from closed model is the calculation method (no threads/pacing), not the scaling behavior.

#### 2.2.5 Multi-Step Pacing Optimization

**Problem:** Pacing (or ops/min/thread) is set once. At each step, `threads = round(step_target_rps × pacing / 1000)`. Due to integer rounding, actual intensity deviates from target. For some pacing values, the deviation is minimal at step 1 but terrible at step 3.

**Algorithm — Brute-force search with scoring:**

```
1. Define base_pacing = max_script_time_ms × pacing_multiplier
2. Define search range: [base_pacing - delta, base_pacing + delta]
   where delta = base_pacing × 0.25  (25% exploration window)
3. For each candidate_pacing in range (step = 1ms):
   a. For each step in test profile:
      - step_target_rps = base_target_rps × step.percent / 100
      - ideal_threads = step_target_rps × candidate_pacing / 1000
      - actual_threads = round(ideal_threads)  // try ceil, floor, round
      - actual_rps = actual_threads / (candidate_pacing / 1000)
      - deviation = |actual_rps - step_target_rps| / step_target_rps × 100
   b. score = max(all step deviations)  // minimize worst-case
      // Alternative: score = sum(all step deviations)  // minimize total
   c. Reject if any step deviation > deviation_tolerance
4. Select candidate_pacing with lowest score
5. If no candidate passes tolerance — report warning, use best-effort
```

**For JMeter closed model:** Same algorithm, but optimizing `ops_per_min_per_thread` instead of `pacing`.

**Complexity note:** For a typical test with 5-10 steps and a search range of ~1000ms, this is ~10,000 iterations × 10 steps = 100K operations. Trivial for modern hardware, no need for sophisticated optimization.

### 2.3 Background Scenarios

Background scenarios run at a fixed percentage of their target intensity throughout the test, ignoring step progression.

```
background_rps = target_rps × background_percent / 100
```

Their threads/pacing are calculated once using the standard formulas. They do not participate in step scaling.

**Priority rule:** `background: true` always takes precedence over `spike_participate`. A background scenario never changes its intensity, even if `spike_participate: true` is set (explicitly or via global default). The `spike_participate` flag is ignored for background scenarios.

### 2.4 Spike Scenarios

In spike test profile:
- Scenarios with `spike_participate = true` (resolved via global default + per-scenario override): oscillate between base_percent and spike_percent
- Scenarios with `spike_participate = false`: stay at base_percent throughout

Spike amplitude grows with each spike:
```
spike_1_percent = base_percent + spike_start_increment          // e.g., 70% + 50% = 120%
spike_2_percent = base_percent + spike_start_increment + spike_increment_growth  // 70% + 60% = 130%
spike_n_percent = base_percent + spike_start_increment + (n-1) × spike_increment_growth
```

**Timeline for spike test:**

Each spike is a full phase cycle: `rampup → impact → stability → [rampdown]` (rampdown is optional, may be 0).

```
[rampup to base] → [impact at base] → [stability at base] →
[rampup to spike_1] → [impact at spike_1] → [stability at spike_1] → [rampdown to base] → [cooldown at base] →
[rampup to spike_2] → [impact at spike_2] → [stability at spike_2] → [rampdown to base] → [cooldown at base] →
...
[rampdown to 0]  (optional)
```

---

## 3. Input Format

### 3.1 Primary: YAML

YAML is the primary input format — easy to version in Git, human-readable, schema-validatable.

```yaml
# loadcalc input configuration
version: "1.0"

global:
  tool: jmeter                    # lre_pc | jmeter
  load_model: closed              # closed | open — default for all scenarios
  pacing_multiplier: 3.0          # default multiplier for pacing calculation
  deviation_tolerance: 2.5        # max allowed deviation from target, %
  spike_participate: true         # default: scenarios participate in spikes
  generators_count: 3             # JMeter only: number of load generators
  range_down: 0.2                 # multiplier search range below base (default 0.2)
  range_up: 0.5                   # multiplier search range above base (default 0.5)

# Scenarios can be defined here (inline) and/or loaded from external CSV/XLSX files.
# When both are present, they are concatenated (YAML scenarios first, then external files).
scenarios:
  Main page:
    script_id: 101                 # LRE PC only: script ID in Performance Center
    target_intensity: 720000
    intensity_unit: ops_h          # ops_h | ops_m | ops_s
    max_script_time_ms: 1100
    # All other fields use global defaults

  Test page:
    script_id: 102
    target_intensity: 1500
    intensity_unit: ops_m
    max_script_time_ms: 1000
    pacing_multiplier: 4.0         # override: use x4 for this scenario

  404 page:
    script_id: 103
    target_intensity: 90000
    intensity_unit: ops_h
    max_script_time_ms: 200
    background: true               # background scenario
    background_percent: 100        # runs at 100% of target always

  API health check:
    target_intensity: 75
    intensity_unit: ops_h
    max_script_time_ms: 50
    load_model: open               # override: uses Free-Form Arrivals TG
    spike_participate: false       # does not participate in spikes

profile:
  type: capacity                 # stability | capacity | custom | spike

  # Default timing for all steps (overrideable per step)
  default_rampup_sec: 60
  default_impact_sec: 120
  default_stability_sec: 300
  default_rampdown_sec: 60         # 0 = no rampdown phase

  # capacity specific — Mechanism A: uniform increment
  start_percent: 50
  step_increment: 25
  num_steps: 5
  # Generates steps: 50%, 75%, 100%, 125%, 150%

  # Per-step timing overrides (optional)
  # Matched by percent value. If percent doesn't match any generated step — validation warning.
  steps:
    - percent: 50
      stability_sec: 600           # longer warmup at first step
    - percent: 100
      stability_sec: 900           # longer measurement at target
    # Steps without overrides use defaults

# --- Max search with fine_tune example (Mechanism C) ---
# Use case: coarse steps to find the range, then fine steps to find exact max.
# Example: 100%, 200%, 300%, 310%, 320%, 330%
#
# profile:
#   type: capacity
#   start_percent: 100
#   step_increment: 100
#   num_steps: 3                   # → [100, 200, 300]
#   fine_tune:
#     after_percent: 300           # after which step to switch increment
#     step_increment: 10           # new smaller increment
#     num_steps: 3                 # → [310, 320, 330]
#   # Combined result: [100, 200, 300, 310, 320, 330]
#   default_rampup_sec: 60
#   default_impact_sec: 120
#   default_stability_sec: 300
#   default_rampdown_sec: 60

# --- Custom profile example (Mechanism B) ---
# Arbitrary percent values in any order, repeats allowed.
# Use case: non-uniform step patterns, re-testing specific intensities.
#
# profile:
#   type: custom
#   steps:
#     - percent: 100
#     - percent: 200
#     - percent: 100               # repeat is allowed
#       stability_sec: 600         # per-step timing override
#     - percent: 300
#     - percent: 200
#   default_rampup_sec: 60
#   default_impact_sec: 120
#   default_stability_sec: 300
#   default_rampdown_sec: 0        # no rampdown between steps

# --- Spike profile example ---
# profile:
#   type: spike
#   base_percent: 70
#   spike_start_increment: 50      # first spike: 70% + 50% = 120%
#   spike_increment_growth: 10     # each next spike: +10% more
#   num_spikes: 3                  # spikes at 120%, 130%, 140%
#   spiketime_sec: 120
#   cooldown_sec: 300
#   default_rampup_sec: 30
#   default_impact_sec: 60
#   default_stability_sec: 300
#   default_rampdown_sec: 30

# --- Stable profile example ---
# Single step at specified percent of target intensity.
# Background scenarios use their own background_percent independently.
#
# profile:
#   type: stability
#   percent: 100                   # can be any value, e.g., 25 for quarter-load test
#   default_rampup_sec: 120
#   default_impact_sec: 180
#   default_stability_sec: 1800
#   default_rampdown_sec: 120      # 0 = no rampdown
```

### 3.2 External Scenario Sources: CSV / XLSX

Scenarios can be loaded from external CSV or XLSX files in addition to (or instead of) inline YAML scenarios. This is the preferred workflow when scenarios are managed in spreadsheets or copied from Confluence.

**CLI flags:**
```bash
loadcalc calculate -i config.yaml --scenarios scenarios.csv                    # single file
loadcalc calculate -i config.yaml --scenarios uc_web.csv --scenarios uc_api.csv  # multiple files
loadcalc calculate -i config.yaml --scenarios-dir ./scenarios/                 # all CSV/XLSX in directory
loadcalc calculate -i config.yaml --csv-delimiter ","                          # custom delimiter (default: ";")
```

**Concatenation rule:** Scenarios from YAML (if any) come first, then scenarios from `--scenarios` files (in order specified), then from `--scenarios-dir` (sorted by filename). This allows keeping "always-on" scenarios (e.g., background load) in YAML while varying the rest via external files.

#### CSV format

Semicolon-separated by default (`;`), customizable via `--csv-delimiter`. First row is header. Column names match YAML scenario field names:

```csv
name;script_id;target_intensity;intensity_unit;max_script_time_ms;background;background_percent;load_model;pacing_multiplier;deviation_tolerance;spike_participate
Main page;101;720000;ops_h;1100;;;;;
Test page;102;1500;ops_m;1000;;;;4.0;;
404 page;103;90000;ops_h;200;true;100;;;;
API health check;;75;ops_h;50;;;open;;;false
```

- Empty cells use global defaults (same as omitting fields in YAML)
- `script_id` is required for LRE PC, empty for JMeter
- Column order does not matter; unknown columns are ignored
- Minimum required columns: `name`, `target_intensity`, `intensity_unit`, `max_script_time_ms`

#### XLSX scenario format

Sheet "Scenarios" with the same column structure as CSV. Multiple sheets are ignored — only the first sheet (or a sheet named "Scenarios") is read.

#### Blank template generation

```bash
loadcalc template --format yaml -o config.yaml          # YAML config template
loadcalc template --format csv -o scenarios.csv          # CSV scenario template with headers
loadcalc template --format xlsx -o scenarios.xlsx        # XLSX scenario template
```

### 3.3 Input Validation Rules

| Rule | Condition | Severity |
|------|-----------|----------|
| `max_script_time_ms` | > 0 | Error |
| `pacing_multiplier` | >= 1.0 | Error |
| `deviation_tolerance` | >= 0 | Error |
| `target_intensity` | > 0 | Error |
| `generators_count` | >= 1 | Error |
| `background_percent` | 0..100 | Error |
| `step percent` | > 0 (for capacity/stability); any > 0 for custom | Error |
| `name` | Non-empty for all scenarios | Error |
| `script_id` for LRE PC | Required (> 0) when `tool: lre_pc` | Error |
| `script_id` for JMeter | Ignored silently | — |
| Pacing > script time | `pacing_ms >= max_script_time_ms` | Warning (auto-satisfied by multiplier >= 1) |
| Open model + LRE PC | Not supported | Error |
| Open model threshold | ops/sec < 0.01 → auto-switch to ops/min | Warning |
| Deviation exceeded | Any scenario/step exceeds tolerance | Warning (highlighted in output) |
| Zero threads | Calculated threads rounds to 0 | Error (minimum 1 thread) |
| Step override mismatch | `steps[].percent` in capacity doesn't match any generated step | Warning (override ignored) |
| `generators_count` + LRE PC | `generators_count` specified when `tool: lre_pc` | Ignored silently |
| `fine_tune.after_percent` | Must match last generated step from base range | Error |
| Custom steps empty | `type: custom` with empty `steps[]` | Error |
| Background + spike_participate | `background: true` with `spike_participate: true` | Ignored (`background` wins) |

---

## 4. Output Format

### 4.1 XLSX Output

The output workbook contains the following sheets:

#### Sheet 1: "Summary"

High-level overview of all scenarios across all steps.

| Column | Description |
|--------|-------------|
| # | Row number (sequential) |
| Scenario Name | Name (LRE PC: group name; JMeter: display name) |
| Script ID | LRE PC only: script ID in Performance Center |
| Load Model | closed / open |
| Target Intensity | Original target with unit |
| Pacing (ms) | Calculated/optimized pacing (closed model, LRE PC) |
| Ops/min/thread | Throughput per thread (closed model, JMeter) |
| Base Threads | Threads at 100% target |
| Is Background | Yes/No |
| Spike Participate | Yes/No |
| Max Deviation % | Worst deviation across all steps |

Conditional formatting: rows where `Max Deviation %` exceeds tolerance are highlighted in red/orange.

#### Sheet 2: "Steps Detail"

Per-scenario, per-step breakdown.

| Column | Description |
|--------|-------------|
| Step # | Step number |
| Step % | Percent of target |
| Scenario Name | Scenario name |
| Target RPS | Target ops/sec for this step |
| Threads | Integer thread count |
| Actual RPS | Actual ops/sec with integer threads |
| Deviation % | Deviation from target |
| Rampup (s) | Rampup duration |
| Impact (s) | Stabilization duration |
| Stability (s) | Stable load duration |
| Rampdown (s) | Rampdown duration |

Conditional formatting on Deviation % column.

#### Sheet 3: "Timeline"

Visual timeline of the entire test with aggregated thread counts and timing. Includes an embedded XLSX chart showing thread count over time for visual clarity (far more readable than ASCII for complex profiles).

| Column | Description |
|--------|-------------|
| Phase | "Step 1 Rampup", "Step 1 Impact", "Step 1 Stability", "Spike 2 Rampup", "Spike 2 Impact", etc. |
| Start Time (s) | Absolute start time from test beginning |
| Duration (s) | Phase duration |
| End Time (s) | Absolute end time |
| Total Threads | Sum of all scenario threads at this phase |
| Per-scenario columns | Thread count for each scenario |

**Phase sequence per step:** Rampup → Impact → Stability → Rampdown (rampdown omitted if duration = 0).

#### Sheet 4: "Input Parameters"

Full dump of input configuration for reproducibility.

#### Sheet 5: "JMeter Config" (JMeter only)

Per-generator configuration ready for JMeter setup.

| Column | Description |
|--------|-------------|
| Scenario Name | Scenario name |
| Thread Group Type | ThreadGroup / FreeFormArrivalsThreadGroup |
| Threads (per generator) | Thread count divided by generators |
| CTT ops/min (per thread) | Constant Throughput Timer value |
| Intensity unit | ops/sec or ops/min (for open model) |
| Intensity value (per generator) | For open model scenarios |

### 4.2 TUI Output

Interactive terminal view with:
- Summary table with colored deviation indicators (green < 1%, yellow 1-2.5%, red > tolerance) plus symbols for accessibility: `✓` (ok), `⚠` (warning), `✗` (exceeded)
- Switchable column groups via Tab key (tool-specific columns shown on demand to fit terminal width)
- Step navigation (arrow keys to browse steps)
- Per-scenario detail view with timing breakdown shown below the table
- Export to XLSX from TUI

**Note:** Timeline visualization is provided as a chart in the XLSX output (§4.1, Sheet "Timeline"), not in TUI — a chart is far more readable for complex profiles with many steps/spikes.

### 4.3 CLI Output

Structured text output (table or JSON) suitable for piping:
```bash
loadcalc calculate -i config.yaml -o results.xlsx
loadcalc calculate -i config.yaml --format json    # JSON to stdout
loadcalc calculate -i config.yaml --format table    # ASCII table to stdout
```

---

## 5. Architecture

### 5.1 Package Structure

```
loadcalc/
├── cmd/
│   └── loadcalc/
│       └── main.go                  # Entry point, CLI flag parsing
│
├── internal/
│   ├── config/
│   │   ├── model.go                 # Domain structs (TestPlan, Scenario, Profile, etc.)
│   │   ├── loader.go                # Load from YAML or XLSX
│   │   ├── validator.go             # Input validation
│   │   └── defaults.go              # Default resolution (global → per-scenario)
│   │
│   ├── engine/
│   │   ├── calculator.go            # Core calculation interface + factory
│   │   ├── lrepc.go                 # LRE PC calculation strategy
│   │   ├── jmeter_closed.go         # JMeter closed model strategy
│   │   ├── jmeter_open.go           # JMeter open model strategy
│   │   ├── optimizer.go             # Multi-step pacing optimization
│   │   └── types.go                 # Calculation result types
│   │
│   ├── profile/
│   │   ├── builder.go               # Profile → step list builder interface
│   │   ├── stability.go                # Stable profile step generation
│   │   ├── capacity.go            # Max search step generation (incl. fine_tune)
│   │   ├── custom.go                # Custom profile — explicit step list
│   │   ├── spike.go                 # Spike profile step + timeline generation
│   │   └── timeline.go              # Timeline construction from steps
│   │
│   ├── output/
│   │   ├── xlsx_writer.go           # XLSX output generation
│   │   ├── json_writer.go           # JSON output
│   │   ├── table_writer.go          # ASCII table output
│   │   └── formatter.go             # Unit formatting, rounding display
│   │
│   ├── tui/
│   │   ├── app.go                   # TUI application (bubbletea)
│   │   ├── views/
│   │   │   ├── summary.go           # Summary table view
│   │   │   ├── steps.go             # Step detail view
│   │   │   ├── timeline.go          # Timeline view
│   │   │   └── scenario.go          # Single scenario detail
│   │   └── styles.go                # Color schemes, formatting
│   │
│   └── integration/                 # Future: LRE PC API, JMeter XML
│       ├── jmeter_xml.go            # JMeter test plan XML generation
│       └── lrepc_api.go             # LRE PC API client
│
├── pkg/
│   └── units/
│       └── convert.go               # Unit conversion utilities (ops/h ↔ ops/m ↔ ops/s)
│
├── testdata/
│   ├── valid_config.yaml
│   ├── valid_config.xlsx
│   ├── invalid_configs/             # Various invalid configs for testing
│   └── expected_outputs/            # Golden files for e2e tests
│
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

### 5.2 Key Interfaces

```go
// Calculator computes load parameters for a specific tool + load model combination.
type Calculator interface {
    // Calculate computes parameters for a single scenario at a single intensity level.
    Calculate(scenario config.Scenario, targetRPS float64) (Result, error)
    // Optimize finds optimal pacing/throughput across all steps for a scenario.
    Optimize(scenario config.Scenario, steps []Step) (OptimizedResult, error)
}

// ProfileBuilder generates the list of steps with timing from a test profile config.
type ProfileBuilder interface {
    // BuildSteps generates ordered steps with their percent and timing.
    BuildSteps(profile config.TestProfile) ([]Step, error)
    // BuildTimeline generates full test timeline from steps.
    BuildTimeline(steps []Step, scenarios []config.Scenario) (Timeline, error)
}

// OutputWriter writes calculation results in a specific format.
type OutputWriter interface {
    Write(results CalculationResults, timeline Timeline, dest string) error
}
```

### 5.3 Design Patterns

| Pattern | Where | Why |
|---------|-------|-----|
| Strategy | `Calculator` interface with LRE/JMeter implementations | Different tools have different calculation logic |
| Strategy | `ProfileBuilder` interface with Stable/MaxSearch/Custom/Spike | Different profiles generate different step sequences |
| Strategy | `OutputWriter` interface with XLSX/JSON/Table | Multiple output formats |
| Factory | `NewCalculator(tool, loadModel)` | Select strategy based on config |
| Builder | `ProfileBuilder.BuildSteps()` → `BuildTimeline()` | Complex object construction |
| Value Object | `Intensity`, `Pacing`, `Duration` types with unit info | Type-safe unit handling |
| Result | Go-style `(value, error)` returns everywhere | Explicit error handling |

---

## 6. Development Phases

### Phase 0: Project Scaffold & CI

**Goal:** Repository structure, build system, linting, CI pipeline.

**Tasks:**
1. Initialize Go module (`go mod init`)
2. Create directory structure per §5.1
3. Set up `Makefile` (build, test, lint, fmt)
4. Configure linter (`golangci-lint` with strict config)
5. Create `README.md` with project description
6. Add `.gitignore`

**Agent workflow:**
- `SubAgent-Scaffold` → creates structure
- `ValidatorAgent-Scaffold` → verifies build, lint passes, directories exist
- `FixAgent-Scaffold` → fixes issues if any

**Tests:** Build compiles, `make lint` passes.

---

### Phase 1: Domain Model & Unit Conversion

**Goal:** Core data structures and unit conversion with 100% test coverage.

**Tasks:**
1. Define all domain structs in `internal/config/model.go`
2. Implement unit conversion in `pkg/units/convert.go`
3. Write comprehensive unit tests for conversions (edge cases: 0, very small values, very large values)

**TDD cycle:**
1. Write tests for `OpsPerHourToPerSec`, `OpsPerMinToPerSec`, `NormalizeIntensity`
2. Implement
3. Write tests for edge cases (0 intensity, negative — should error)
4. Implement validation

**Agent workflow:**
- `SubAgent-Model` → writes structs + conversion + tests
- `ValidatorAgent-Model` → runs tests, checks coverage, verifies edge cases handled, checks for false positives in tests
- `FixAgent-Model` → if needed

**Tests:** Unit tests for all conversions. Property: `ToPerSec(ToPerHour(x)) == x` within float precision.

---

### Phase 2: Input Loading & Validation

**Goal:** Parse YAML and XLSX inputs, resolve defaults, validate.

**Tasks:**
1. Implement YAML loader (`internal/config/loader.go`)
2. Implement default resolution: global → per-scenario merge (`internal/config/defaults.go`)
3. Implement validation rules from §3.3 (`internal/config/validator.go`)
4. Implement XLSX loader (read template format)
5. Implement `loadcalc template` command to generate blank XLSX

**TDD cycle:**
1. Write tests: valid YAML → parsed struct matches expectations
2. Write tests: missing fields → global defaults applied
3. Write tests: each validation rule (invalid input → specific error)
4. Write tests: XLSX input → same result as equivalent YAML
5. Implement

**Agent workflow:**
- `SubAgent-Loader` → YAML loader + defaults + tests
- `ValidatorAgent-Loader` → runs tests, checks that invalid configs produce clear errors, not panics
- `FixAgent-Loader` → if needed
- `SubAgent-XLSXLoader` → XLSX loader + template generator + tests
- `ValidatorAgent-XLSXLoader` → validates
- `FixAgent-XLSXLoader` → if needed

**Tests:** Unit tests for each loader. Integration test: load file → validate → get config. Negative tests for every validation rule.

---

### Phase 3: Calculation Engine — Single Step

**Goal:** Calculate threads/pacing/throughput for a single scenario at a single intensity.

**Tasks:**
1. Implement `Calculator` interface
2. Implement `LREPCCalculator` (§2.2.2)
3. Implement `JMeterClosedCalculator` (§2.2.3)
4. Implement `JMeterOpenCalculator` (§2.2.4)
5. Implement calculator factory `NewCalculator(tool, loadModel)`
6. Write unit tests with known expected values (from manual calculations and the reference Excel)

**TDD cycle:**
1. Write tests from the reference Excel data:
   - UC01: 720000 ops/h, 1100ms, x3, 3 generators → expect threads=220, round=220, CTT=18.18...
   - UC02: 90000 ops/h, 1000ms, x3 → expect ...
   - UC03: 90000 ops/h, 200ms, x3 → expect ...
   - UC04: 90000 ops/h, 1100ms, x3 → expect ...
2. Write tests from the manual calculations in our conversation:
   - 13044 ops/h, 550ms, x3 → 6 threads, pacing 1656ms
   - 75 ops/h, 50ms, x3 → 1 thread, pacing 48000ms
3. Implement calculators
4. Write edge case tests: very low intensity, very high intensity, 1 thread minimum

**Agent workflow:**
- `SubAgent-Engine` → implements calculators + tests
- `ValidatorAgent-Engine` → runs tests, cross-validates with Excel data, checks float precision, checks for false positives
- `FixAgent-Engine` → if needed

**Tests:** Unit tests per calculator. Cross-validation with reference Excel. Property: `actual_threads >= 1`, `pacing > 0`, `deviation computable`.

---

### Phase 4: Profile Builders — Step Generation

**Goal:** Generate step sequences from test profile configuration.

**Tasks:**
1. Implement `ProfileBuilder` interface
2. Implement `StableProfileBuilder` — single step at configured percent
3. Implement `MaxSearchProfileBuilder` — generates N steps from start_percent with increment, supports `fine_tune` for two-range increment
4. Implement `CustomProfileBuilder` — takes explicit step list, any order, repeats allowed
5. Implement `SpikeProfileBuilder` — generates base + spike alternation with growing amplitude (each spike = rampup→impact→stability→[rampdown])
6. Handle background scenarios (fixed, not affected by steps; `background: true` overrides `spike_participate`)
7. Handle spike participation flags
8. Handle optional rampdown (0 = no rampdown phase)

**TDD cycle:**
1. Write tests: stable profile → 1 step at percent with correct timing
2. Write tests: capacity 50%/25%/5 steps → [50, 75, 100, 125, 150] with timing
3. Write tests: capacity with fine_tune → [100, 200, 300, 310, 320, 330]
4. Write tests: custom profile → arbitrary list [100, 200, 100, 300, 200] preserved as-is
5. Write tests: custom profile with per-step timing overrides
6. Write tests: spike with base 70%, start +50%, growth +10%, 3 spikes → correct sequence
7. Write tests: background scenarios excluded from step scaling (background wins over spike_participate)
8. Write tests: spike_participate flag respected
9. Write tests: rampdown_sec = 0 → no rampdown phase generated
10. Implement

**Agent workflow:**
- `SubAgent-Profile` → implements builders + tests
- `ValidatorAgent-Profile` → validates step sequences, checks timing math, checks edge cases (1 step, 0 increment)
- `FixAgent-Profile` → if needed

**Tests:** Unit tests per builder. Integration: config → builder → steps list matches expected.

---

### Phase 5: Multi-Step Pacing Optimizer

**Goal:** Find optimal pacing/throughput that minimizes deviation across all steps.

**Tasks:**
1. Implement `Optimizer` (§2.2.5)
2. Search range: base_pacing ± 25%
3. Scoring: minimize worst-case deviation across steps
4. Constraint: reject candidates where any step exceeds tolerance
5. Fallback: if no candidate passes, use best-effort + warning
6. For JMeter: optimize `ops_per_min_per_thread` instead of pacing

**TDD cycle:**
1. Write tests: single step → optimizer returns base pacing (no optimization needed)
2. Write tests: known multi-step case where base pacing produces bad deviation at step 3, optimizer finds better
3. Write tests: all candidates exceed tolerance → warning returned, best-effort used
4. Write tests: optimizer result satisfies `for each step: deviation <= tolerance`
5. Implement

**Key test case from our conversation:**
- Base: 100% = 6 threads, Step: 210% → naive would give 12.6 → 13 threads = 216.7%
- Optimizer should find pacing where 210% maps cleanly (e.g., threads that give closer to 210%)

**Agent workflow:**
- `SubAgent-Optimizer` → implements optimizer + tests
- `ValidatorAgent-Optimizer` → runs tests, verifies scoring logic, checks that optimizer actually improves over base pacing, checks for false positives
- `FixAgent-Optimizer` → if needed

**Tests:** Unit tests, property tests (optimizer result deviation ≤ base pacing deviation).

---

### Phase 6: XLSX Output

**Goal:** Generate output spreadsheet with all sheets per §4.1.

**Tasks:**
1. Implement `XLSXWriter` using `excelize` library
2. Generate Summary sheet with conditional formatting
3. Generate Steps Detail sheet
4. Generate Timeline sheet
5. Generate Input Parameters sheet
6. Generate JMeter Config sheet (conditional on tool)
7. Apply styles: headers, column widths, number formatting, deviation highlighting

**TDD cycle:**
1. Write tests: generate XLSX → read back → data matches expected
2. Write tests: conditional formatting applied on deviation column
3. Write tests: JMeter Config sheet present only when tool=jmeter
4. Implement

**Agent workflow:**
- `SubAgent-XLSX` → implements writer + tests
- `ValidatorAgent-XLSX` → verifies generated file opens correctly, data accuracy, formatting
- `FixAgent-XLSX` → if needed
- `UIDesignAgent-XLSX` → reviews spreadsheet layout, column order, readability, formatting conventions

**Tests:** Integration tests: full pipeline config → calculate → write XLSX → read back and verify.

---

### Phase 7: CLI Interface

**Goal:** Command-line interface with all commands.

**Commands:**
```
loadcalc calculate -i <config.yaml> -o <output.xlsx> [--scenarios <file.csv|xlsx>]... [--scenarios-dir <dir>] [--csv-delimiter ";"] [--format table|json|xlsx]
loadcalc template --format <yaml|csv|xlsx> -o <output>
loadcalc validate -i <config.yaml> [--scenarios <file.csv|xlsx>]... [--scenarios-dir <dir>]
loadcalc version
```

**Tasks:**
1. Implement CLI using `cobra` library
2. `calculate` command: load → validate → calculate → optimize → build timeline → output
3. `template` command: generate blank input file
4. `validate` command: load + validate only, report issues
5. Structured error output (exit codes, clear messages)

**TDD cycle:**
1. Write e2e tests: `loadcalc calculate -i testdata/valid_config.yaml -o /tmp/out.xlsx` → exit 0, file exists
2. Write e2e tests: `loadcalc calculate -i testdata/invalid_config.yaml` → exit 1, error message contains reason
3. Write e2e tests: `loadcalc validate -i testdata/valid_config.yaml` → exit 0
4. Write e2e tests: `loadcalc template --format yaml -o /tmp/template.yaml` → valid YAML written
5. Implement

**Agent workflow:**
- `SubAgent-CLI` → implements commands + tests
- `ValidatorAgent-CLI` → runs e2e tests, checks error messages are user-friendly, verifies exit codes
- `FixAgent-CLI` → if needed

**Tests:** E2E tests using `os/exec` to run the binary. Golden file comparison for table/JSON output.

---

### Phase 8: TUI Interface

**Goal:** Interactive terminal UI for exploring results.

**Tasks:**
1. Set up `bubbletea` application structure
2. Summary view: table with all scenarios, colored deviations
3. Step navigation: browse steps with arrow keys, see per-step data
4. Scenario detail view: drill into one scenario
5. Timeline view: ASCII timeline with phases
6. Export action: save to XLSX from TUI
7. Deviation color coding: green (< 1%), yellow (1–2.5%), red (> tolerance)

**TDD cycle:**
1. Write unit tests for TUI models (state transitions, key handling)
2. Write tests for view rendering (expected output strings)
3. Implement views

**Agent workflow:**
- `UIDesignAgent-TUI` → designs layout, navigation flow, color scheme, information hierarchy
- `SubAgent-TUI` → implements TUI + tests
- `ValidatorAgent-TUI` → tests key navigation, rendering correctness, edge cases (empty config, 1 scenario)
- `FixAgent-TUI` → if needed

**Tests:** Model unit tests. Snapshot tests for view output.

---

### Phase 9: Integration Testing & Polish

**Goal:** End-to-end validation, cross-validation with reference Excel, edge cases.

**Tasks:**
1. E2E test: reference Excel data → loadcalc → verify output matches Excel calculations
2. E2E test: all three profile types with realistic configs
3. E2E test: mixed open/closed model scenarios
4. E2E test: background scenarios + spike test
5. E2E test: edge cases (1 scenario, 100 scenarios, very low/high intensity)
6. Performance test: 100 scenarios × 20 steps — must complete in < 1 second
7. Cross-platform build verification (linux/darwin/windows × amd64/arm64)
8. Polish: README, usage examples, sample configs

**Agent workflow:**
- `SubAgent-E2E` → writes comprehensive e2e tests
- `ValidatorAgent-E2E` → runs all tests, reviews coverage report, identifies untested paths
- `FixAgent-E2E` → if needed
- `SubAgent-Docs` → README, examples, sample configs

**Tests:** Full pipeline e2e tests. Coverage target: 85%+ for engine/config/profile packages.

---

### Phase 10: JMeter XML Generation

**Goal:** Generate and inject JMeter `.jmx` test plan elements from calculation results.

#### 10.1 Commands

```bash
# Generate new .jmx from scratch (skeleton with all ThreadGroups)
loadcalc jmx generate -i config.yaml -o test.jmx [--scenarios ...]

# Inject ThreadGroups into existing .jmx template (append to end)
loadcalc jmx inject -i config.yaml --jmx-template existing.jmx -o result.jmx [--scenarios ...]

# Inject + update existing ThreadGroups in-place (dangerous, explicit flag)
loadcalc jmx inject -i config.yaml --jmx-template existing.jmx -o result.jmx --update-existing [--scenarios ...]
```

#### 10.2 Thread Group Type Selection

| Condition | Thread Group Type |
|-----------|-------------------|
| Closed model, uniform steps (equal increment) | `kg.apc.jmeter.threads.SteppingThreadGroup` |
| Closed model, non-uniform steps (varying increments) | `kg.apc.jmeter.threads.UltimateThreadGroup` |
| Open model | `com.blazemeter.jmeter.threads.arrivals.FreeFormArrivalsThreadGroup` |

#### 10.3 Generated Elements per Scenario

**Closed model (Stepping/Ultimate TG):**
- Thread Group with calculated thread counts per step, rampup/hold times
- Constant Throughput Timer (`ConstantThroughputTimer`) with `ops_per_min_per_thread` value
- Module Controller referencing a Test Fragment by scenario `name`

**Open model (Free-Form Arrivals TG):**
- FreeFormArrivalsThreadGroup with intensity schedule (rate per step, rampup, hold)
- Module Controller referencing a Test Fragment by scenario `name`

#### 10.4 Module Controller + Test Fragment Matching

Each generated ThreadGroup contains a Module Controller that references a Test Fragment. Matching is by scenario `name`:
- Search the `.jmx` (in inject mode) or the generated plan for a `TestFragmentController` whose `testname` attribute matches the scenario `name`
- If found: Module Controller points to it
- If not found: Module Controller is created with a placeholder comment, and a **warning is logged**: `"Test Fragment not found for scenario: <name>"`

**Logging:** At INFO level, log a matching report:
```
Scenario "Main page" → Test Fragment "Main page" [MATCHED]
Scenario "API health" → Test Fragment [NOT FOUND — placeholder created]
```

#### 10.5 `--update-existing` Mode

When `--update-existing` is set, instead of appending new ThreadGroups:
1. Search existing ThreadGroups by name (including `FFATG_*` prefix pattern)
2. If found: update thread count, CTT value, stepping/schedule config in-place
3. If not found: append as new (same as default mode)
4. Log all matches and updates at INFO level

**Warning:** This mode modifies existing ThreadGroup structure. Always use with `-o` to write to a new file, preserving the original.

#### 10.6 Generate Mode (Variant A)

Generates a complete `.jmx` with:
- `TestPlan` root element
- All ThreadGroups with calculated values
- Module Controllers referencing Test Fragments
- Empty Test Fragments (as placeholders) for each scenario — user fills in the actual samplers

#### 10.7 Tasks

1. Implement JMeter XML builder in `internal/integration/jmeter_xml.go`
2. Implement `.jmx` parser/injector for template mode
3. Implement `--update-existing` ThreadGroup matching (by name, `FFATG_*` prefix)
4. Implement Module Controller + Test Fragment matching/creation
5. Add `jmx generate` and `jmx inject` CLI subcommands
6. Comprehensive tests with sample `.jmx` files

**TDD cycle:**
1. Write tests: generate .jmx → parse XML → verify ThreadGroup types, thread counts, CTT values
2. Write tests: inject into template → verify original content preserved + new TGs appended
3. Write tests: --update-existing → verify existing TGs updated, unmatched TGs appended
4. Write tests: Module Controller → Test Fragment matching
5. Implement

---

### Phase 11: LRE PC API Integration

**Goal:** Push calculation results to LoadRunner Enterprise (Performance Center) via REST API — create/update test groups, set Vuser counts, pacing, and scheduler.

#### 11.1 LRE PC API Overview

**Base URL:** `https://{server}:{port}/LoadTest/rest/domains/{domain}/projects/{project}`

**Authentication:**
- `POST /LoadTest/rest/authentication-point/authenticate` with Basic Auth header
- Returns `LWSSO_COOKIE_KEY` session cookie for subsequent requests
- Logout: `DELETE /LoadTest/rest/authentication-point/authenticate`

**Key endpoints:**

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/tests` | GET | List tests |
| `/tests/{testId}` | GET/PUT | Get/update test |
| `/tests/{testId}/groups` | GET/POST | List/create groups |
| `/tests/{testId}/groups/{groupId}` | PUT | Update group |
| `/tests/{testId}/groups/{groupId}/runtime-settings` | GET/PUT | Pacing, think time |
| `/tests/{testId}/scheduler` | GET/PUT | Ramp-up, duration, ramp-down |
| `/scripts` | GET | List available scripts |

#### 11.2 Concept Mapping

| loadcalc | LRE PC API |
|----------|------------|
| Scenario `name` | Group `Name` |
| Scenario `script_id` | Group `ScriptId` |
| Calculated threads | Group `VuserCount` |
| Calculated pacing_ms | RuntimeSettings → Pacing (`MinDelay`/`MaxDelay` in ms) |
| Step rampup_sec | Scheduler → `RampUpAmount`, `RampUpInterval` |
| Step stability_sec | Scheduler → `Duration` |
| Step rampdown_sec | Scheduler → `RampDownAmount`, `RampDownInterval` |

#### 11.3 Commands

```bash
# Push calculation results to LRE PC test
loadcalc lre push -i config.yaml --server https://lre.company.com --domain PERF --project MyProject --test-id 123 [--scenarios ...]

# List available tests
loadcalc lre list-tests --server https://lre.company.com --domain PERF --project MyProject

# List scripts
loadcalc lre list-scripts --server https://lre.company.com --domain PERF --project MyProject

# Dry-run: show what would be pushed without making changes
loadcalc lre push ... --dry-run
```

**Credentials:** `--user` and `--password` flags, or `LOADCALC_LRE_USER` / `LOADCALC_LRE_PASSWORD` env vars.

#### 11.4 Push Logic

1. Authenticate with LRE PC
2. Run full calculation pipeline (load → validate → calculate → optimize)
3. For each scenario:
   a. Find existing group in test by `Name` match, or create new group
   b. Set `VuserCount` to calculated threads (for the target step, default 100%)
   c. Set `ScriptId` from scenario `script_id`
   d. Update runtime settings: set pacing to calculated `corrected_pacing_ms`
4. Update scheduler with test profile timing (ramp-up, duration, ramp-down)
5. Log all actions: group matched/created, Vusers set, pacing set
6. `--dry-run`: log what would be done without making API calls

#### 11.5 Tasks

1. Implement LRE PC API client in `internal/integration/lrepc_api.go`
2. Authentication (Basic Auth → session cookie)
3. CRUD for groups (list, create, update)
4. Runtime settings update (pacing)
5. Scheduler update
6. Push orchestration: match groups by name, create missing, update existing
7. CLI commands: `lre push`, `lre list-tests`, `lre list-scripts`
8. Dry-run mode
9. Tests with HTTP mock server

---

## 7. Agent Orchestration Protocol

### 7.1 Agent Types

| Agent | Role | Input | Output |
|-------|------|-------|--------|
| `SubAgent` | Implements a task (code + tests) | Task spec from phase | Code files, test files |
| `ValidatorAgent` | Reviews implementation, runs tests, checks quality | Code from SubAgent | Pass/fail report with issues list |
| `FixAgent` | Fixes issues found by Validator | Issues from Validator | Fixed code |
| `UIDesignAgent` | Designs UI/UX for TUI and XLSX output | Feature requirements | Layout specs, mockups, design decisions |

### 7.2 Execution Flow

```
For each phase:
  1. [UIDesignAgent] (if phase has UI work) → design spec
  2. [SubAgent] → implement code + tests (following TDD: tests first)
  3. [ValidatorAgent] → run tests, review code, check:
     a. All tests pass
     b. No false positives (tests actually test what they claim)
     c. Edge cases covered
     d. Code follows Go conventions (error handling, naming)
     e. No data races / concurrency issues
     f. Logging present at appropriate levels
  4. If issues found:
     [FixAgent] → fix issues
     [ValidatorAgent] → re-validate (max 3 fix cycles, then escalate)
  5. If all pass → proceed to next phase
```

### 7.3 Parallel Execution Rules

Phases execute **sequentially** by default. Exception: phases may run in parallel if **all** conditions are met:
- No shared files between tasks
- No dependency on output of the other task
- Different packages/directories

**Parallelizable pairs:**
- Phase 1 (model) tasks are independent internally
- Phase 6 (XLSX output) and Phase 7 (CLI) can start simultaneously after Phase 5
- Phase 8 (TUI) can start after Phase 7 (CLI), but can be parallel with Phase 6

### 7.4 TDD Protocol

Every SubAgent follows this protocol:

1. **Read the spec** for the current task
2. **Write test file first** with all test cases (including edge cases)
3. **Run tests — verify they fail** (red phase)
4. **Implement minimum code** to pass tests (green phase)
5. **Refactor** while keeping tests green
6. **Run linter** — fix all warnings
7. **Commit** (in real workflow) with descriptive message

### 7.5 Code Quality Standards

- All exported functions have godoc comments (English)
- All error paths handled explicitly (no `_` for errors)
- `context.Context` propagated where appropriate
- Structured logging via `log/slog`
- No `fmt.Println` in library code (only in CLI/TUI)
- Table-driven tests preferred
- Test names: `TestCalculator_LRE_SingleStep_BasicCase`

---

## 8. Logging

Use `log/slog` with structured fields:

```go
slog.Info("calculation complete",
    "scenario", scenario.ID,
    "threads", result.Threads,
    "pacing_ms", result.PacingMS,
    "deviation_percent", result.DeviationPercent,
)
```

Log levels:
- `DEBUG`: calculation intermediates, optimizer iterations
- `INFO`: scenario results, warnings about deviations
- `WARN`: deviation exceeds tolerance, fallback used
- `ERROR`: validation failures, I/O errors

CLI flag: `--log-level debug|info|warn|error` (default: `warn`)

---

## 9. Configuration Reference

### 9.1 Global Defaults Table

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `tool` | enum | *required* | `lre_pc` or `jmeter` |
| `load_model` | enum | `closed` | Default load model for scenarios |
| `pacing_multiplier` | float | `3.0` | Default pacing multiplier |
| `deviation_tolerance` | float | `2.5` | Max allowed deviation (%) |
| `spike_participate` | bool | `true` | Default spike participation |
| `generators_count` | int | `1` | JMeter load generator count |

### 9.2 Scenario Fields Table

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `name` | string | *required* | LRE PC: group name in test; JMeter: display name |
| `script_id` | int | *required for LRE PC* | LRE PC: script ID in Performance Center; ignored for JMeter |
| `target_intensity` | float | *required* | Target value |
| `intensity_unit` | enum | `ops_h` | `ops_h`, `ops_m`, `ops_s` |
| `max_script_time_ms` | int | *required* | Max script execution time |
| `background` | bool | `false` | Is background scenario |
| `background_percent` | float | `100` | % of target for background |
| `load_model` | enum | *global* | Override load model |
| `pacing_multiplier` | float | *global* | Override multiplier |
| `deviation_tolerance` | float | *global* | Override tolerance |
| `spike_participate` | bool | *global* | Override spike participation |

---

## 10. Future Considerations

1. **JMeter XML generation** (Phase 10): direct `.jmx` output
2. **LRE PC API** (Phase 11): push test configuration via REST
3. **Configuration diffing** (Phase 12)
4. **What-if mode** (Phase 13)
5. **Multi-config merge** (Phase 14)

---

### Phase 12: Configuration Diffing

**Goal:** Compare two configurations and display parameter differences.

**Command:**
```bash
loadcalc diff old.yaml new.yaml [--scenarios old_sc.csv --scenarios new_sc.csv] [--format table|json]
```

**Output:** A structured diff showing:
- **Global changes:** fields that changed (e.g., `tool: jmeter → lre_pc`, `pacing_multiplier: 3.0 → 4.0`)
- **Profile changes:** type changed, steps added/removed/modified, timing changes
- **Scenario changes:**
  - Added scenarios (in new but not in old, matched by `name`)
  - Removed scenarios (in old but not in new)
  - Modified scenarios: per-field diff (target_intensity, max_script_time_ms, etc.)
- **Calculation impact:** for modified scenarios, show old vs new threads/pacing/deviation (requires running calculation on both)

**Matching:** Scenarios are matched by `name` field. If names are not unique, match by position within same-name groups.

**Output formats:**
- `table` (default): colored terminal table with `+`/`-`/`~` indicators
- `json`: structured JSON diff for automation

---

### Phase 13: What-If Mode

**Goal:** Interactively tweak parameters and see cascading effects without editing config files.

**Command:**
```bash
loadcalc what-if -i config.yaml [--scenarios ...] [--set global.pacing_multiplier=4.0] [--set "scenarios[0].target_intensity=500000"]
```

**Behavior:**
1. Load and calculate with original config → baseline results
2. Apply `--set` overrides to config (dot-notation path to any field)
3. Recalculate with modified config → modified results
4. Display side-by-side comparison: baseline vs modified for each affected scenario

**Path syntax for `--set`:**
- `global.pacing_multiplier=4.0` — change global field
- `global.generators_count=5` — change generator count
- `scenarios[0].target_intensity=500000` — change first scenario's intensity
- `scenarios[Main page].max_script_time_ms=2000` — change by scenario name
- `profile.num_steps=10` — change profile field

**Output:** Table showing per-scenario: old threads → new threads, old pacing → new pacing, old deviation → new deviation. Highlight improvements in green, regressions in red.

**Multiple overrides:** `--set` can be specified multiple times to apply several changes at once.

---

### Phase 14: Multi-Config Merge

**Goal:** Combine multiple config files into a single test plan.

**Command:**
```bash
loadcalc merge config_web.yaml config_api.yaml -o combined.yaml [--scenarios web.csv --scenarios api.csv]
loadcalc merge --dir ./configs/ -o combined.yaml
```

**Merge rules:**
- **Global settings:** taken from the first config file. If subsequent configs have conflicting globals, emit a warning and keep the first. User can override with `--set global.field=value`.
- **Profile:** taken from the first config file. Conflict → warning.
- **Scenarios:** concatenated from all configs (same as `--scenarios` concatenation). Scenarios from external CSV/XLSX files associated with each config are included.
- **Duplicate scenario names:** allowed (they are a list, not a map).

**`--dir` mode:** Load all `.yaml` files from a directory (sorted by name), merge in order.

**Output:** A single merged YAML config file that can be used with `loadcalc calculate`.

---

## Appendix A: Reference Calculations

### From Reference Excel (JMeter, closed model, 3 generators)

| Scenario | Target ops/h | Script ms | Pacing ms | Thread throughput ops/min | Threads (float) | Threads (ceil) | CTT ops/min/thread |
|----------|-------------|-----------|-----------|--------------------------|-----------------|---------------|-------------------|
| UC01 Main page | 720,000 | 1,100 | 3,300 | 18.18 | 220.0 | 220 | 18.18 |
| UC02 Test page | 90,000 | 1,000 | 3,000 | 20.0 | 25.0 | 25 | 20.0 |
| UC03 404 page | 90,000 | 200 | 600 | 100.0 | 5.0 | 5 | 100.0 |
| UC04 Main page eng | 90,000 | 1,100 | 3,300 | 18.18 | 27.5 | 28 | 17.86 |

*Note: Target ops/min per generator = (target_ops_h / 60) / 3 generators*

### From Conversation (LRE PC, closed model)

**Case 1:** 13,044 ops/h, 550ms, x3
- Pacing: 1,650ms → target_rps: 3.623 → threads: 5.98 → 6
- Corrected pacing: 1,656ms → actual: 3.623 rps = 13,044/h ✓

**Case 2:** 75 ops/h, 50ms, x3
- Pacing: 150ms → target_rps: 0.02083 → threads: 0.003 → 1
- Corrected pacing: 48,000ms → actual: 0.02083 rps = 75/h ✓
