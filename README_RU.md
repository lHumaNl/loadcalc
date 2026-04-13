# loadcalc

**[English version](README.md)**

Калькулятор параметров нагрузочного тестирования для **LRE Performance Center** и **JMeter**.

Рассчитывает оптимальное количество потоков, пейсинг и пропускную способность из целевой интенсивности — чтобы не считать вручную.

---

## Проблема

Инструменты нагрузочного тестирования требуют целое число потоков и фиксированный пейсинг. Когда в тесте несколько ступеней интенсивности (100% → 150% → 200%), ошибки округления накапливаются. Добавление 1 потока может превысить цель больше, чем шаг ступени.

**loadcalc** находит оптимальный пейсинг, минимизирующий отклонение по всем ступеням.

---

## Установка

### Бинарник

Скачайте из [Releases](https://github.com/lHumaNl/loadcalc/releases) для вашей платформы.

### Сборка из исходников

```bash
git clone https://github.com/lHumaNl/loadcalc.git
cd loadcalc
make build
# бинарник: ./bin/loadcalc
```

---

## Быстрый старт

**1. Создайте конфиг:**

```bash
loadcalc template --format yaml -o config.yaml
```

**2. Отредактируйте** — укажите сценарии, целевую интенсивность и профиль теста.

**3. Рассчитайте:**

```bash
loadcalc calculate -i config.yaml -o results.xlsx
```

Готово. Откройте `results.xlsx` — там потоки, пейсинг, отклонения и график таймлайна.

---

## Команды

| Команда | Что делает |
|---------|-----------|
| `calculate` | Расчёт, вывод в XLSX / JSON / таблицу |
| `validate` | Проверка конфига на ошибки |
| `template` | Генерация пустого конфига (YAML / CSV / XLSX) |
| `tui` | Интерактивный терминальный интерфейс |
| `diff` | Сравнение двух конфигов |
| `what-if` | Изменить параметр — увидеть влияние |
| `merge` | Объединение нескольких конфигов в один |
| `jmx generate` | Создание JMeter .jmx с нуля |
| `jmx inject` | Добавление ThreadGroup в существующий .jmx |
| `lre push` | Отправка результатов в LRE Performance Center |
| `lre list-tests` | Список тестов в LRE PC |
| `lre list-scripts` | Список скриптов в LRE PC |

---

## Пример конфига

```yaml
version: "1.0"

global:
  tool: jmeter              # jmeter | lre_pc
  load_model: closed         # closed | open
  pacing_multiplier: 3.0
  deviation_tolerance: 2.5   # макс. допустимое отклонение, %
  generators_count: 3        # только JMeter

scenarios:
  Main page:
    script_id: 101           # только LRE PC
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
  num_steps: 5               # ступени: 50%, 75%, 100%, 125%, 150%
  default_rampup_sec: 60
  default_stability_sec: 300
```

### Сценарии из CSV / XLSX

Сценарии можно загружать из внешних файлов — удобнее заполнять в Excel или копировать из Confluence:

```bash
loadcalc calculate -i config.yaml --scenarios scenarios.csv
loadcalc calculate -i config.yaml --scenarios uc_web.csv --scenarios uc_api.csv
loadcalc calculate -i config.yaml --scenarios-dir ./scenarios/
```

Сценарии из YAML и из файлов **объединяются** — можно держать "постоянные" сценарии в YAML, а остальные менять через CSV.

Генерация пустого шаблона:

```bash
loadcalc template --format csv -o scenarios.csv
```

Разделитель CSV — `;` по умолчанию (переопределение: `--csv-delimiter ","`).
XLSX использует те же колонки на листе "Scenarios".
Пустые ячейки используют глобальные значения. Порядок колонок не важен.

**Пример** (как выглядит в таблице или CSV):

| name | script_id | target_intensity | intensity_unit | max_script_time_ms | background | background_percent | load_model | pacing_multiplier |
|------|-----------|-----------------|---------------|-------------------|------------|-------------------|------------|------------------|
| Main page | 101 | 720000 | ops_h | 1100 | | | | |
| Test page | 102 | 1500 | ops_m | 1000 | | | | 4.0 |
| 404 page | 103 | 90000 | ops_h | 200 | true | 100 | | |
| API health | | 75 | ops_h | 50 | | | open | |

**Описание колонок:**

| Колонка | Обязательна | По умолчанию | Описание |
|---------|-------------|-------------|----------|
| `name` | да | — | Имя сценария (LRE PC: имя группы в тесте) |
| `script_id` | только LRE PC | — | ID скрипта в Performance Center (для JMeter игнорируется) |
| `target_intensity` | да | — | Целевая интенсивность |
| `intensity_unit` | нет | `ops_h` | `ops_h` / `ops_m` / `ops_s` |
| `max_script_time_ms` | да | — | Макс. время выполнения скрипта в миллисекундах |
| `background` | нет | `false` | `true` = фиксированная нагрузка, не зависит от ступеней |
| `background_percent` | нет | `100` | % от целевой интенсивности для фоновых сценариев |
| `load_model` | нет | из global | `closed` / `open` |
| `pacing_multiplier` | нет | из global | Переопределение множителя пейсинга для сценария |
| `deviation_tolerance` | нет | из global | Переопределение макс. допустимого отклонения (%) |
| `spike_participate` | нет | из global | `true` / `false` — участие в фазах спайков |

---

## Профили теста

| Профиль | Для чего |
|---------|---------|
| **stability** | Одна ступень на фиксированном % от цели |
| **capacity** | Нарастающие ступени для поиска максимума (поддержка `fine_tune` для двух диапазонов инкремента) |
| **custom** | Произвольный список ступеней в любом порядке, повторы допустимы |
| **spike** | Базовая нагрузка + растущие спайки для тестирования устойчивости |

---

## Ключевые возможности

- **Оптимизатор пейсинга** — перебор в диапазоне ±25%, проба ceil/floor/round, минимизация наихудшего отклонения
- **Два инструмента** — LRE PC (закрытая модель) и JMeter (закрытая + открытая модели)
- **Гибкий ввод** — YAML конфиг + CSV/XLSX сценарии, несколько файлов, загрузка из директории
- **JMeter .jmx** — генерация с нуля, инъекция в существующий, обновление ThreadGroup по имени (поиск по префиксам STG\_/UTG\_/FFATG\_)
- **LRE PC API** — отправка групп, пейсинга и планировщика напрямую
- **What-if анализ** — изменение любого параметра через `--set`, сравнение до/после
- **Один бинарник** — без зависимостей, кросс-платформенный

---

## Примеры использования

```bash
# Расчёт и экспорт в XLSX
loadcalc calculate -i config.yaml -o results.xlsx

# JSON вывод в stdout
loadcalc calculate -i config.yaml --format json

# Интерактивный TUI
loadcalc tui -i config.yaml

# Сравнение конфигов
loadcalc diff old.yaml new.yaml

# Что если pacing multiplier будет 4?
loadcalc what-if -i config.yaml --set global.pacing_multiplier=4.0

# Объединение конфигов команд
loadcalc merge web.yaml api.yaml -o combined.yaml

# Генерация JMeter тест-плана
loadcalc jmx generate -i config.yaml -o test.jmx

# Инъекция в существующий .jmx
loadcalc jmx inject -i config.yaml --jmx-template base.jmx -o test.jmx

# Обновление существующих ThreadGroup в .jmx
loadcalc jmx inject -i config.yaml --jmx-template base.jmx -o test.jmx --update-existing

# Отправка в существующий тест LRE PC
loadcalc lre push -i config.yaml --server https://lre.company.com \
  --domain PERF --project MyProject --test-id 123

# Создание нового теста в LRE PC и отправка
loadcalc lre push -i config.yaml --server https://lre.company.com \
  --domain PERF --project MyProject --test-name "Release 2.1 Perf Test" --test-folder "Subject/Perf"

# Пробный запуск (посмотреть что будет отправлено, без изменений)
loadcalc lre push -i config.yaml --server https://lre.company.com \
  --domain PERF --project MyProject --test-id 123 --dry-run
```

---

## Лицензия

MIT
