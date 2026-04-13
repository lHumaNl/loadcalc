# loadcalc

**[Русская версия](README_RU.md)**

Load test parameter calculator for **LRE Performance Center** and **JMeter**.

Calculates optimal thread counts, pacing, and throughput from target intensity — so you don't have to do it by hand.

---

## The Problem

Load testing tools need integer thread counts and fixed pacing values. When your test has multiple intensity steps (100% → 150% → 200%), rounding errors accumulate. Adding 1 thread can overshoot the target by more than the step increment.

**loadcalc** finds the optimal pacing that minimizes deviation across all steps.

---

## Install

### Binary

Download from [Releases](https://github.com/lHumaNl/loadcalc/releases) for your platform.

### Build from source

```bash
git clone https://github.com/lHumaNl/loadcalc.git
cd loadcalc
make build
# binary: ./bin/loadcalc
```

---

## Quick Start

**1. Create a config:**

```bash
loadcalc template --format yaml -o config.yaml
```

**2. Edit it** — set your scenarios, target intensity, and test profile.

**3. Calculate:**

```bash
loadcalc calculate -i config.yaml -o results.xlsx
```

That's it. Open `results.xlsx` — you'll see threads, pacing, deviations, and a timeline chart.

---

## Commands

| Command | What it does |
|---------|-------------|
| `calculate` | Run calculations, output XLSX / JSON / table |
| `validate` | Check config for errors |
| `template` | Generate blank config (YAML / CSV / XLSX) |
| `tui` | Interactive terminal UI |
| `diff` | Compare two configs side-by-side |
| `what-if` | Tweak a parameter, see the impact |
| `merge` | Combine multiple configs into one |
| `jmx generate` | Create a JMeter .jmx from scratch |
| `jmx inject` | Add ThreadGroups into existing .jmx |
| `lre push` | Push results to LRE Performance Center |
| `lre list-tests` | List tests in LRE PC |
| `lre list-scripts` | List scripts in LRE PC |

---

## Config Example

```yaml
version: "1.0"

global:
  tool: jmeter              # jmeter | lre_pc
  load_model: closed         # closed | open
  pacing_multiplier: 3.0
  deviation_tolerance: 2.5   # max allowed deviation, %
  generators_count: 3        # JMeter only

scenarios:
  Main page:
    script_id: 101           # LRE PC only
    target_intensity: 720000
    intensity_unit: ops_h    # ops_h | ops_m | ops_s
    max_script_time_ms: 1100

  Background load:
    target_intensity: 90000
    intensity_unit: ops_h
    max_script_time_ms: 200
    background: true
    background_percent: 100

profile:
  type: capacity            # stability | capacity | custom | spike
  start_percent: 50
  step_increment: 25
  num_steps: 5               # steps: 50%, 75%, 100%, 125%, 150%
  default_rampup_sec: 60
  default_stability_sec: 300
```

### Scenarios from CSV / XLSX

Scenarios can also be loaded from external files — easier to manage in Excel or copy from Confluence:

```bash
loadcalc calculate -i config.yaml --scenarios scenarios.csv
loadcalc calculate -i config.yaml --scenarios uc_web.csv --scenarios uc_api.csv
loadcalc calculate -i config.yaml --scenarios-dir ./scenarios/
```

YAML scenarios and file scenarios are **concatenated** — you can keep "always-on" scenarios in YAML and vary the rest via CSV.

Generate a blank template:

```bash
loadcalc template --format csv -o scenarios.csv
```

CSV delimiter is `;` by default (override with `--csv-delimiter ","`).
XLSX uses the same columns on a sheet named "Scenarios".
Empty cells use global defaults. Column order doesn't matter.

**Example** (how it looks in a spreadsheet or CSV):

| name | script_id | target_intensity | intensity_unit | max_script_time_ms | background | background_percent | load_model | pacing_multiplier |
|------|-----------|-----------------|---------------|-------------------|------------|-------------------|------------|------------------|
| Main page | 101 | 720000 | ops_h | 1100 | | | | |
| Test page | 102 | 1500 | ops_m | 1000 | | | | 4.0 |
| 404 page | 103 | 90000 | ops_h | 200 | true | 100 | | |
| API health | | 75 | ops_h | 50 | | | open | |

**Column reference:**

| Column | Required | Default | Description |
|--------|----------|---------|-------------|
| `name` | yes | — | Scenario name (LRE PC: group name in test) |
| `script_id` | LRE PC only | — | Script ID in Performance Center (ignored for JMeter) |
| `target_intensity` | yes | — | Target load value |
| `intensity_unit` | no | `ops_h` | `ops_h` / `ops_m` / `ops_s` |
| `max_script_time_ms` | yes | — | Max script execution time in milliseconds |
| `background` | no | `false` | `true` = fixed load, ignores step scaling |
| `background_percent` | no | `100` | % of target intensity for background scenarios |
| `load_model` | no | from global | `closed` / `open` |
| `pacing_multiplier` | no | from global | Override pacing multiplier for this scenario |
| `deviation_tolerance` | no | from global | Override max allowed deviation (%) |
| `spike_participate` | no | from global | `true` / `false` — participate in spike phases |

---

## Test Profiles

| Profile | Use case |
|---------|----------|
| **stability** | Single step at fixed % of target |
| **capacity** | Incrementing steps to find system max (supports `fine_tune` for two-range increments) |
| **custom** | Arbitrary step list in any order, repeats allowed |
| **spike** | Base load + growing spikes to test resilience |

---

## Key Features

- **Pacing optimizer** — brute-force search across ±25% range, tries ceil/floor/round, minimizes worst-case deviation
- **Multi-tool** — LRE PC (closed model) and JMeter (closed + open models)
- **Flexible input** — YAML config + CSV/XLSX scenarios, multiple files, directory loading
- **JMeter .jmx** — generate from scratch, inject into existing, update ThreadGroups by name (STG\_/UTG\_/FFATG\_ prefix matching)
- **LRE PC API** — push groups, pacing, and scheduler directly
- **What-if analysis** — change any parameter with `--set`, see before/after
- **Single binary** — no runtime dependencies, cross-platform

---

## Usage Examples

```bash
# Calculate and export to XLSX
loadcalc calculate -i config.yaml -o results.xlsx

# JSON output to stdout
loadcalc calculate -i config.yaml --format json

# Interactive TUI
loadcalc tui -i config.yaml

# Compare configs
loadcalc diff old.yaml new.yaml

# What if pacing multiplier was 4?
loadcalc what-if -i config.yaml --set global.pacing_multiplier=4.0

# Merge team configs
loadcalc merge web.yaml api.yaml -o combined.yaml

# Generate JMeter test plan
loadcalc jmx generate -i config.yaml -o test.jmx

# Inject into existing .jmx
loadcalc jmx inject -i config.yaml --jmx-template base.jmx -o test.jmx

# Update existing ThreadGroups in .jmx
loadcalc jmx inject -i config.yaml --jmx-template base.jmx -o test.jmx --update-existing

# Push to existing LRE PC test
loadcalc lre push -i config.yaml --server https://lre.company.com \
  --domain PERF --project MyProject --test-id 123

# Create a new test in LRE PC and push
loadcalc lre push -i config.yaml --server https://lre.company.com \
  --domain PERF --project MyProject --test-name "Release 2.1 Perf Test" --test-folder "Subject/Perf"

# Dry run (see what would be pushed without making changes)
loadcalc lre push -i config.yaml --server https://lre.company.com \
  --domain PERF --project MyProject --test-id 123 --dry-run
```

---

## License

MIT
