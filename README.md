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
  - name: "Main page"
    script_id: 101           # LRE PC only
    target_intensity: 720000
    intensity_unit: ops_h    # ops_h | ops_m | ops_s
    max_script_time_ms: 1100

  - name: "Background load"
    target_intensity: 90000
    intensity_unit: ops_h
    max_script_time_ms: 200
    background: true
    background_percent: 100

profile:
  type: max_search           # stable | max_search | custom | spike
  start_percent: 50
  step_increment: 25
  num_steps: 5               # steps: 50%, 75%, 100%, 125%, 150%
  default_rampup_sec: 60
  default_stability_sec: 300
```

Scenarios can also be loaded from CSV or XLSX files:

```bash
loadcalc calculate -i config.yaml --scenarios scenarios.csv
loadcalc calculate -i config.yaml --scenarios-dir ./scenarios/
```

---

## Test Profiles

| Profile | Use case |
|---------|----------|
| **stable** | Single step at fixed % of target |
| **max_search** | Incrementing steps to find system max (supports `fine_tune` for two-range increments) |
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

# Push to LRE Performance Center
loadcalc lre push -i config.yaml --server https://lre.company.com \
  --domain PERF --project MyProject --test-id 123

# Dry run (see what would be pushed)
loadcalc lre push -i config.yaml --server https://lre.company.com \
  --domain PERF --project MyProject --test-id 123 --dry-run
```

---

## License

MIT
