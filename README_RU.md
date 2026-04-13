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
  - name: "Main page"
    script_id: 101           # только LRE PC
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
  num_steps: 5               # ступени: 50%, 75%, 100%, 125%, 150%
  default_rampup_sec: 60
  default_stability_sec: 300
```

Сценарии можно загружать из CSV или XLSX файлов:

```bash
loadcalc calculate -i config.yaml --scenarios scenarios.csv
loadcalc calculate -i config.yaml --scenarios-dir ./scenarios/
```

---

## Профили теста

| Профиль | Для чего |
|---------|---------|
| **stable** | Одна ступень на фиксированном % от цели |
| **max_search** | Нарастающие ступени для поиска максимума (поддержка `fine_tune` для двух диапазонов инкремента) |
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

# Отправка в LRE Performance Center
loadcalc lre push -i config.yaml --server https://lre.company.com \
  --domain PERF --project MyProject --test-id 123

# Пробный запуск (посмотреть что будет отправлено)
loadcalc lre push -i config.yaml --server https://lre.company.com \
  --domain PERF --project MyProject --test-id 123 --dry-run
```

---

## Лицензия

MIT
