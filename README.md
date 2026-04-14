# loadcalc

**[Русская версия](README_RU.md)**

Load test parameter calculator for **LRE Performance Center** and **JMeter**.

Calculates optimal thread counts, pacing, and throughput from target intensity — so you don't have to do it by hand.

---

## The Problem

Load testing tools need integer thread counts and fixed pacing values. When a test has multiple intensity steps (100% → 150% → 200%), rounding errors accumulate. Adding a single thread can overshoot the target by more than a full step.

**loadcalc** finds the pacing that minimizes deviation across all steps, then prints the numbers you need to plug into JMeter or LRE PC.

---

## Install

### Binary

Download from [Releases](https://github.com/lHumaNl/loadcalc/releases) for your platform.

### Build from source

```bash
git clone https://github.com/lHumaNl/loadcalc.git
cd loadcalc
make build
# binary: ./loadcalc
```

---

## Quick Start

### 1. Interactive calculator (the easiest way)

Just run the binary with no arguments:

```bash
./loadcalc
```

This launches the **quick calculator TUI** — an interactive form where you edit parameters and results recalculate live.

```
  loadcalc quick calculator

  Intensity:      13044
  Unit:           ops/h
  Script time:    550 ms
  Tool:           LRE PC
  Multiplier:     3.0
  Range down:     0.2 -
  Range up:       2.0 +
  Steps:          50,75,100,125,150
  Rampup:         60 s

  ── Results ────────────────────────────────────────
  Pacing: 2,208 ms
  ╭──────┬──────┬────────┬───────┬─────────┬──────────┬────────┬───────────┬────────┬───────┬─────────╮
  │ Step │    % │ Vusers │ Delta │ VUBatch │ Every(s) │ Rampup │     ops/h │  ops/m │ ops/s │     Dev │
  ├──────┼──────┼────────┼───────┼─────────┼──────────┼────────┼───────────┼────────┼───────┼─────────┤
  │    1 │  50% │      4 │    +4 │       1 │      15s │    60s │  6,521.74 │  108.7 │  1.81 │ 0.00% ✓ │
  │    2 │  75% │      6 │    +2 │       1 │      30s │    60s │  9,782.61 │ 163.04 │  2.72 │ 0.00% ✓ │
  │    3 │ 100% │      8 │    +2 │       1 │      30s │    60s │ 13,043.48 │ 217.39 │  3.62 │ 0.00% ✓ │
  │    4 │ 125% │     10 │    +2 │       1 │      30s │    60s │ 16,304.35 │ 271.74 │  4.53 │ 0.00% ✓ │
  │    5 │ 150% │     12 │    +2 │       1 │      30s │    60s │ 19,565.22 │ 326.09 │  5.44 │ 0.00% ✓ │
  ╰──────┴──────┴────────┴───────┴─────────┴──────────┴────────┴───────────┴────────┴───────┴─────────╯

  [Tab] next field  [Space/←/→] cycle  [Ctrl+C] quit
```

**Features:**
- **Live recalculation** — any field change instantly updates the result table
- **Smart hints** — if the current multiplier range can't achieve 0% deviation, the TUI suggests a better multiplier and the exact `Range down` / `Range up` value needed to reach it
- **Per-step Vusers + ramp-up** for LRE PC: how many Vusers to add at each step, batch size, interval in seconds (you don't have to calculate this yourself)
- **Multi-unit rates** — ops/h, ops/m, ops/s shown simultaneously
- **Context-aware fields** — Generators hidden for LRE PC, Multiplier/Range hidden for open model, etc.
- **Deviation indicators** — `✓` / `⚠` / `✗` plus colors

**Navigation:**

| Key | Action |
|-----|--------|
| `Tab` / `↓` | next field |
| `Shift+Tab` / `↑` | previous field |
| `Space` / `←` / `→` | cycle options (for Tool, Unit, Model) |
| Any character | type into text field |
| `Backspace` | delete last character |
| `Ctrl+C` | quit |

### 2. One-shot CLI calculation

When you already know the numbers and just want the answer:

```bash
# 720000 ops/h, 1100 ms script, JMeter
loadcalc quick 720000 1100 jmeter

# Multi-step capacity run on LRE PC
loadcalc quick 720000 1100 lre_pc --steps 50,75,100,125,150
```

Flags for `quick`:

| Flag | Default | Description |
|------|---------|-------------|
| `--multiplier` | `3.0` | Pacing multiplier (base) |
| `--range-down` | `0.2` | Multiplier search range below base |
| `--range-up` | `0.5` | Multiplier search range above base |
| `--generators` | `1` | JMeter generator count (ignored for LRE PC) |
| `--model` | `closed` | Load model: `closed` or `open` |
| `--unit` | `ops_h` | Intensity unit: `ops_h` / `ops_m` / `ops_s` |
| `--tolerance` | `2.5` | Max allowed deviation, % |
| `--steps` | — | Comma-separated percentages for multi-step |
| `--rampup` | `60` | Ramp-up seconds per step (LRE PC) |

### 3. Full workflow with YAML config

For real tests with multiple scenarios, profiles, and reproducible output:

```bash
loadcalc template --format yaml -o config.yaml   # generate blank config
# edit config.yaml — scenarios, target intensity, profile
loadcalc calculate -i config.yaml -o results.xlsx
```

Open `results.xlsx` — you'll see threads, pacing, deviations, a timeline chart, and (for LRE PC) an "LRE PC Config" sheet with per-step Vusers and ramp-up batches.

---

## Commands

| Command | What it does |
|---------|--------------|
| _(no args)_ | Launch interactive quick calculator TUI |
| `quick` | One-shot CLI calculation for a single scenario |
| `calculate` | Run full calculation from config, output XLSX / JSON / table |
| `validate` | Check config for errors |
| `template` | Generate blank config (YAML / CSV) |
| `tui` | Interactive results viewer for a config file |
| `diff` | Compare two configs side-by-side |
| `what-if` | Tweak a parameter, see the impact |
| `merge` | Combine multiple configs into one |
| `jmx generate` | Create a JMeter .jmx from scratch |
| `jmx inject` | Add/update ThreadGroups in an existing .jmx |
| `lre push` | Push results to LRE Performance Center |
| `lre list-tests` | List tests in LRE PC |
| `lre list-scripts` | List scripts in LRE PC |

---

## YAML Config

```yaml
version: "1.0"

global:
  tool: jmeter              # jmeter | lre_pc
  load_model: closed        # closed | open
  pacing_multiplier: 3.0
  deviation_tolerance: 2.5  # max allowed deviation, %
  generators_count: 3       # JMeter only
  range_down: 0.2           # multiplier search range below base
  range_up: 0.5             # multiplier search range above base

scenarios:                  # MAP — key is the scenario name (must be unique)
  Main page:
    script_id: 101          # LRE PC only — script ID in Performance Center
    target_intensity: 720000
    intensity_unit: ops_h   # optional, defaults to ops_h
    max_script_time_ms: 1100

  Background load:
    target_intensity: 90000
    max_script_time_ms: 200
    background: true
    background_percent: 100

profile:
  type: capacity            # stability | capacity | custom | spike
  start_percent: 50
  step_increment: 25
  num_steps: 5              # steps: 50%, 75%, 100%, 125%, 150%
  default_rampup_sec: 60
  default_stability_sec: 300
```

Notes:
- `scenarios` is a **map** (key = scenario name). Duplicate names are a validation error.
- `intensity_unit` is optional and defaults to `ops_h`.
- `script_id` is only needed for LRE PC; JMeter ignores it.

---

## Scenarios from CSV / XLSX

Scenarios can also be loaded from external files — easier to edit in Excel or copy from Confluence:

```bash
loadcalc calculate -i config.yaml --scenarios scenarios.csv
loadcalc calculate -i config.yaml --scenarios uc_web.csv --scenarios uc_api.csv
loadcalc calculate -i config.yaml --scenarios-dir ./scenarios/
```

YAML scenarios and file scenarios are **concatenated** — keep "always-on" scenarios in YAML, vary the rest via CSV. Duplicate names across all sources are a validation error.

Generate a blank template:

```bash
loadcalc template --format csv -o scenarios.csv
```

CSV delimiter defaults to `;` (override with `--csv-delimiter ","`). XLSX uses the same columns on a sheet named "Scenarios". Empty cells inherit global defaults. Column order does not matter.

**Example:**

| name | script_id | target_intensity | intensity_unit | max_script_time_ms | background | background_percent | load_model | pacing_multiplier |
|------|-----------|------------------|----------------|--------------------|------------|--------------------|------------|-------------------|
| Main page | 101 | 720000 | ops_h | 1100 | | | | |
| Test page | 102 | 1500 | ops_m | 1000 | | | | 4.0 |
| 404 page | 103 | 90000 | ops_h | 200 | true | 100 | | |
| API health | | 75 | ops_h | 50 | | | open | |

**Column reference:**

| Column | Required | Default | Description |
|--------|----------|---------|-------------|
| `name` | yes | — | Scenario name (also LRE PC group name). Must be unique. |
| `script_id` | LRE PC only | — | Script ID in Performance Center (ignored for JMeter) |
| `target_intensity` | yes | — | Target load value |
| `intensity_unit` | no | `ops_h` | `ops_h` / `ops_m` / `ops_s` |
| `max_script_time_ms` | yes | — | Max script execution time (ms) |
| `background` | no | `false` | `true` = fixed load, ignores step scaling |
| `background_percent` | no | `100` | % of target intensity for background scenarios |
| `load_model` | no | from global | `closed` / `open` |
| `pacing_multiplier` | no | from global | Override pacing multiplier |
| `deviation_tolerance` | no | from global | Override max allowed deviation (%) |
| `spike_participate` | no | from global | Participate in spike phases |

---

## Test Profiles

| Profile | Use case |
|---------|----------|
| **stability** | Single step at a fixed % of target |
| **capacity** | Incrementing steps to find system max (supports `fine_tune` for two-range increments) |
| **custom** | Arbitrary step list in any order, repeats allowed |
| **spike** | Base load + growing spikes to test resilience |

---

## Key Features

- **Interactive quick calculator** — `./loadcalc` with no args, live recalculation as you type
- **Pacing optimizer** — brute-force search across a configurable multiplier range, tries ceil/floor/round, minimizes worst-case deviation
- **Multi-tool** — LRE PC (closed model) and JMeter (closed + open models)
- **Flexible input** — YAML config + CSV/XLSX scenarios, multiple files, directory loading
- **XLSX output** — scenarios, steps, timeline chart, and a dedicated "LRE PC Config" sheet with Vusers per step, batch size, and interval
- **JMeter .jmx** — generate from scratch, inject into existing templates, update ThreadGroups by name (STG\_/UTG\_/FFATG\_ prefixes)
- **LRE PC API** — update existing tests by `--test-id`, or create a new test with `--test-name` + `--test-folder`. Ramp-up batch/interval is calculated automatically.
- **What-if analysis** — change any parameter with `--set`, compare before/after
- **Single binary** — no runtime dependencies, cross-platform

---

## Usage Examples

```bash
# Interactive quick calculator (no args)
./loadcalc

# One-shot CLI calc
loadcalc quick 720000 1100 jmeter --multiplier 3.5
loadcalc quick 720000 1100 lre_pc --steps 50,75,100,125,150 --rampup 120

# Full calculation, export to XLSX
loadcalc calculate -i config.yaml -o results.xlsx

# JSON output to stdout
loadcalc calculate -i config.yaml --format json

# Interactive results viewer for a config
loadcalc tui -i config.yaml

# Compare configs
loadcalc diff old.yaml new.yaml

# What if pacing multiplier was 4?
loadcalc what-if -i config.yaml --set global.pacing_multiplier=4.0

# Merge team configs
loadcalc merge web.yaml api.yaml -o combined.yaml

# JMeter — generate from scratch
loadcalc jmx generate -i config.yaml -o test.jmx

# JMeter — inject into existing .jmx
loadcalc jmx inject -i config.yaml --jmx-template base.jmx -o test.jmx

# JMeter — update existing ThreadGroups in-place
loadcalc jmx inject -i config.yaml --jmx-template base.jmx -o test.jmx --update-existing

# LRE PC — push to an existing test
loadcalc lre push -i config.yaml --server https://lre.company.com/LoadTest/rest \
  --domain PERF --project MyProject --test-id 123

# LRE PC — create a new test and push
loadcalc lre push -i config.yaml --server https://lre.company.com/LoadTest/rest \
  --domain PERF --project MyProject \
  --test-name "Release 2.1 Perf Test" --test-folder "Subject/Perf"

# LRE PC — dry run
loadcalc lre push -i config.yaml --server https://lre.company.com/LoadTest/rest \
  --domain PERF --project MyProject --test-id 123 --dry-run
```

---

## License

MIT
