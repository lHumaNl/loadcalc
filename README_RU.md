# loadcalc

**[English version](README.md)**

Калькулятор параметров нагрузочного тестирования для **LRE Performance Center** и **JMeter**.

Рассчитывает оптимальное количество потоков, пейсинг и пропускную способность по заданной целевой интенсивности — чтобы не считать вручную.

---

## Проблема

Инструменты нагрузочного тестирования требуют целое число потоков и фиксированный пейсинг. Когда в тесте несколько ступеней интенсивности (100% → 150% → 200%), ошибки округления накапливаются. Добавление всего одного потока может превысить цель сильнее, чем шаг ступени.

**loadcalc** подбирает пейсинг, минимизирующий отклонение по всем ступеням, и выдаёт готовые числа для JMeter или LRE PC.

---

## Установка

### Бинарник

Скачайте из [Releases](https://github.com/lHumaNl/loadcalc/releases) для вашей платформы.

### Сборка из исходников

```bash
git clone https://github.com/lHumaNl/loadcalc.git
cd loadcalc
make build
# бинарник: ./loadcalc
```

---

## Быстрый старт

### 1. Интерактивный калькулятор (самый простой путь)

Просто запустите бинарник без аргументов:

```bash
./loadcalc
```

Откроется **интерактивный TUI-калькулятор** — форма, в которой вы редактируете параметры, а результаты пересчитываются на лету.

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

**Возможности:**
- **Мгновенный пересчёт** — любое изменение поля сразу обновляет таблицу результатов
- **Умные подсказки** — если текущий диапазон multiplier не позволяет достичь 0% отклонения, TUI предложит лучший multiplier и точное значение `Range down` / `Range up`, которое нужно для его достижения
- **Пошаговый план Vusers и ramp-up** для LRE PC: сколько Vusers добавлять на каждой ступени, размер батча, интервал в секундах — не нужно считать вручную
- **Все единицы сразу** — ops/h, ops/m, ops/s показываются параллельно
- **Контекстные поля** — Generators скрыт для LRE PC, Multiplier/Range скрыты для open model и т.д.
- **Индикаторы отклонения** — `✓` / `⚠` / `✗` плюс цвет

**Навигация:**

| Клавиша | Действие |
|---------|----------|
| `Tab` / `↓` | следующее поле |
| `Shift+Tab` / `↑` | предыдущее поле |
| `Space` / `←` / `→` | переключение опций (Tool, Unit, Model) |
| любой символ | ввод в текстовое поле |
| `Backspace` | удалить последний символ |
| `Ctrl+C` | выход |

### 2. Разовый CLI-расчёт

Когда числа уже известны и нужен только ответ:

```bash
# 720000 ops/h, скрипт 1100 мс, JMeter
loadcalc quick 720000 1100 jmeter

# Многоступенчатый capacity-тест на LRE PC
loadcalc quick 720000 1100 lre_pc --steps 50,75,100,125,150
```

Флаги `quick`:

| Флаг | По умолчанию | Описание |
|------|--------------|----------|
| `--multiplier` | `3.0` | Базовый множитель пейсинга |
| `--range-down` | `0.2` | Диапазон поиска множителя вниз от базы |
| `--range-up` | `0.5` | Диапазон поиска множителя вверх от базы |
| `--generators` | `1` | Количество генераторов JMeter (игнорируется для LRE PC) |
| `--model` | `closed` | Модель нагрузки: `closed` или `open` |
| `--unit` | `ops_h` | Единица интенсивности: `ops_h` / `ops_m` / `ops_s` |
| `--tolerance` | `2.5` | Допустимое отклонение, % |
| `--steps` | — | Список процентов через запятую (многоступенчатый режим) |
| `--rampup` | `60` | Разгон на ступень в секундах (LRE PC) |

### 3. Полный сценарий с YAML-конфигом

Для реальных тестов с несколькими сценариями, профилями и воспроизводимым результатом:

```bash
loadcalc template --format yaml -o config.yaml   # создать пустой конфиг
# отредактируйте config.yaml — сценарии, интенсивность, профиль
loadcalc calculate -i config.yaml -o results.xlsx
```

Откройте `results.xlsx` — потоки, пейсинг, отклонения, график таймлайна и (для LRE PC) отдельный лист «LRE PC Config» с Vusers по ступеням и параметрами разгона.

---

## Команды

| Команда | Что делает |
|---------|------------|
| _(без аргументов)_ | Запуск интерактивного TUI-калькулятора |
| `quick` | Разовый расчёт одного сценария из CLI |
| `calculate` | Полный расчёт из конфига, вывод в XLSX / JSON / таблицу |
| `validate` | Проверка конфига на ошибки |
| `template` | Генерация пустого конфига (YAML / CSV) |
| `tui` | Интерактивный просмотр результатов для конфига |
| `diff` | Сравнение двух конфигов |
| `what-if` | Изменить параметр — увидеть эффект |
| `merge` | Объединение нескольких конфигов |
| `jmx generate` | Генерация JMeter .jmx с нуля |
| `jmx inject` | Добавление/обновление ThreadGroup в существующем .jmx |
| `lre push` | Отправка результатов в LRE Performance Center |
| `lre list-tests` | Список тестов в LRE PC |
| `lre list-scripts` | Список скриптов в LRE PC |

---

## YAML-конфиг

```yaml
version: "1.0"

global:
  tool: jmeter              # jmeter | lre_pc
  load_model: closed        # closed | open
  pacing_multiplier: 3.0
  deviation_tolerance: 2.5  # макс. допустимое отклонение, %
  generators_count: 3       # только JMeter
  range_down: 0.2           # диапазон поиска множителя вниз
  range_up: 0.5             # диапазон поиска множителя вверх

scenarios:                  # MAP — ключ является именем сценария (должен быть уникален)
  Main page:
    script_id: 101          # только LRE PC — ID скрипта в Performance Center
    target_intensity: 720000
    intensity_unit: ops_h   # опционально, по умолчанию ops_h
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
  num_steps: 5              # ступени: 50%, 75%, 100%, 125%, 150%
  default_rampup_sec: 60
  default_stability_sec: 300
```

Замечания:
- `scenarios` — это **map** (ключ = имя сценария). Дубликаты имён считаются ошибкой валидации.
- `intensity_unit` опционален, по умолчанию `ops_h`.
- `script_id` нужен только для LRE PC; JMeter его игнорирует.

---

## Сценарии из CSV / XLSX

Сценарии можно загружать из внешних файлов — удобнее редактировать в Excel или копировать из Confluence:

```bash
loadcalc calculate -i config.yaml --scenarios scenarios.csv
loadcalc calculate -i config.yaml --scenarios uc_web.csv --scenarios uc_api.csv
loadcalc calculate -i config.yaml --scenarios-dir ./scenarios/
```

Сценарии из YAML и из файлов **объединяются** — постоянные сценарии удобно держать в YAML, а остальные варьировать через CSV. Совпадение имён по всем источникам считается ошибкой валидации.

Создать пустой шаблон:

```bash
loadcalc template --format csv -o scenarios.csv
```

Разделитель CSV — `;` по умолчанию (переопределяется `--csv-delimiter ","`). XLSX использует те же колонки на листе «Scenarios». Пустые ячейки наследуют глобальные значения. Порядок колонок не важен.

**Пример:**

| name | script_id | target_intensity | intensity_unit | max_script_time_ms | background | background_percent | load_model | pacing_multiplier |
|------|-----------|------------------|----------------|--------------------|------------|--------------------|------------|-------------------|
| Main page | 101 | 720000 | ops_h | 1100 | | | | |
| Test page | 102 | 1500 | ops_m | 1000 | | | | 4.0 |
| 404 page | 103 | 90000 | ops_h | 200 | true | 100 | | |
| API health | | 75 | ops_h | 50 | | | open | |

**Описание колонок:**

| Колонка | Обязательна | По умолчанию | Описание |
|---------|-------------|--------------|----------|
| `name` | да | — | Имя сценария (также имя группы в LRE PC). Должно быть уникальным. |
| `script_id` | только LRE PC | — | ID скрипта в Performance Center (JMeter игнорирует) |
| `target_intensity` | да | — | Целевая интенсивность |
| `intensity_unit` | нет | `ops_h` | `ops_h` / `ops_m` / `ops_s` |
| `max_script_time_ms` | да | — | Макс. время выполнения скрипта (мс) |
| `background` | нет | `false` | `true` = фиксированная нагрузка, не масштабируется по ступеням |
| `background_percent` | нет | `100` | % от целевой интенсивности для фоновых сценариев |
| `load_model` | нет | из global | `closed` / `open` |
| `pacing_multiplier` | нет | из global | Переопределение множителя пейсинга |
| `deviation_tolerance` | нет | из global | Переопределение допустимого отклонения (%) |
| `spike_participate` | нет | из global | Участие в фазах спайков |

---

## Профили теста

| Профиль | Для чего |
|---------|----------|
| **stability** | Одна ступень на фиксированном % от цели |
| **capacity** | Нарастающие ступени для поиска максимума (поддержка `fine_tune` для двух диапазонов инкремента) |
| **custom** | Произвольный список ступеней в любом порядке, повторы допускаются |
| **spike** | Базовая нагрузка + растущие спайки для проверки устойчивости |

---

## Ключевые возможности

- **Интерактивный калькулятор** — `./loadcalc` без аргументов, пересчёт на лету по мере ввода
- **Оптимизатор пейсинга** — перебор в настраиваемом диапазоне множителя, проба ceil/floor/round, минимизация наихудшего отклонения
- **Два инструмента** — LRE PC (closed-модель) и JMeter (closed + open)
- **Гибкий ввод** — YAML-конфиг + CSV/XLSX-сценарии, несколько файлов, загрузка из директории
- **XLSX-вывод** — сценарии, ступени, график таймлайна и отдельный лист «LRE PC Config» с Vusers, размером батча и интервалом
- **JMeter .jmx** — генерация с нуля, инъекция в существующий шаблон, обновление ThreadGroup по имени (префиксы STG\_/UTG\_/FFATG\_)
- **LRE PC API** — обновление существующего теста по `--test-id` или создание нового через `--test-name` + `--test-folder`. Параметры разгона рассчитываются автоматически.
- **What-if анализ** — изменение любого параметра через `--set`, сравнение до/после
- **Один бинарник** — без зависимостей, кросс-платформенный

---

## Примеры использования

```bash
# Интерактивный калькулятор (без аргументов)
./loadcalc

# Разовый CLI-расчёт
loadcalc quick 720000 1100 jmeter --multiplier 3.5
loadcalc quick 720000 1100 lre_pc --steps 50,75,100,125,150 --rampup 120

# Полный расчёт с экспортом в XLSX
loadcalc calculate -i config.yaml -o results.xlsx

# JSON-вывод в stdout
loadcalc calculate -i config.yaml --format json

# Интерактивный просмотр результатов конфига
loadcalc tui -i config.yaml

# Сравнение конфигов
loadcalc diff old.yaml new.yaml

# Что будет, если множитель пейсинга станет 4?
loadcalc what-if -i config.yaml --set global.pacing_multiplier=4.0

# Объединение командных конфигов
loadcalc merge web.yaml api.yaml -o combined.yaml

# JMeter — генерация с нуля
loadcalc jmx generate -i config.yaml -o test.jmx

# JMeter — инъекция в существующий .jmx
loadcalc jmx inject -i config.yaml --jmx-template base.jmx -o test.jmx

# JMeter — обновление существующих ThreadGroup на месте
loadcalc jmx inject -i config.yaml --jmx-template base.jmx -o test.jmx --update-existing

# LRE PC — отправка в существующий тест
loadcalc lre push -i config.yaml --server https://lre.company.com/LoadTest/rest \
  --domain PERF --project MyProject --test-id 123

# LRE PC — создание нового теста и отправка
loadcalc lre push -i config.yaml --server https://lre.company.com/LoadTest/rest \
  --domain PERF --project MyProject \
  --test-name "Release 2.1 Perf Test" --test-folder "Subject/Perf"

# LRE PC — пробный прогон без изменений
loadcalc lre push -i config.yaml --server https://lre.company.com/LoadTest/rest \
  --domain PERF --project MyProject --test-id 123 --dry-run
```

---

## Лицензия

MIT
