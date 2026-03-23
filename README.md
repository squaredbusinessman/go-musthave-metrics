# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Форматирование goimports

Отформатировать все Go-файлы проекта можно из корня репозитория командой:

```bash
find . -name '*.go' -type f -print0 | xargs -0 goimports -w
```

При последней проверке `goimports` не внёс изменений: все файлы проекта уже были отформатированы.

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Бенчмарки

В проект добавлены бенчмарки для наиболее нагруженных путей:

- `BenchmarkFileStorageSave`
- `BenchmarkFileStorageRestore`
- `BenchmarkMemStorageUpdateMetricsBatch`
- `BenchmarkReportMetricsJSONScheduling`
- `BenchmarkSnapshotToMetrics`

Примеры запуска:

```bash
go test -run '^$' -bench 'Benchmark(FileStorageSave|FileStorageRestore|MemStorageUpdateMetricsBatch)$' -benchmem ./internal/repository
go test -run '^$' -bench 'Benchmark(ReportMetricsJSONScheduling|SnapshotToMetrics)$' -benchmem ./internal/agent
```

## Профилирование памяти

Базовый и итоговый heap profiles сохранены в:

- `profiles/base.pprof`
- `profiles/result.pprof`

Профиль снимался на файловом хранилище, потому что именно этот путь сочетает сериализацию, запись на диск и восстановление из файла:

```bash
go test -run '^$' -bench 'BenchmarkFileStorage(Save|Restore)$' -benchtime=30x -memprofile profiles/base.pprof -memprofilerate=1 ./internal/repository
go test -run '^$' -bench 'BenchmarkFileStorage(Save|Restore)$' -benchtime=30x -memprofile profiles/result.pprof -memprofilerate=1 ./internal/repository
```

Для анализа использовались команды `pprof`:

```bash
pprof -top profiles/base.pprof
pprof -list 'Save$' profiles/base.pprof
pprof -list 'Restore$' profiles/base.pprof
pprof -peek FileStorage profiles/base.pprof
pprof -web profiles/base.pprof
```

Основные выводы из `base.pprof`:

- `FileStorage.Save` держал в памяти сразу три тяжёлых объекта: snapshot map, `[]models.Metrics` и итоговый `[]byte` после `json.MarshalIndent`.
- `FileStorage.Restore` читал весь файл через `os.ReadFile`, затем аллоцировал полный `[]models.Metrics` через `json.Unmarshal`.
- В `pprof -top` доминировали `reflect.growslice`, `encoding/json.MarshalIndent`, `os.readFileContents`, `encoding/json.Unmarshal`.

## Выполненная оптимизация

- `FileStorage.Save` переписан на потоковую запись JSON напрямую в temp-файл без промежуточного `[]models.Metrics` и без большого `[]byte` буфера.
- `FileStorage.Restore` переписан на потоковое чтение через `json.Decoder`, без `os.ReadFile` и без десериализации всего массива метрик целиком.
- Для восстановления введена внутренняя DTO `storageMetricRecord` с обычными `float64`/`int64`, чтобы не плодить лишние pointer-аллокации.
- Добавлены unit-тесты для `internal/service`, `internal/apperr`, `internal/logger`, `cmd/agent`, `migrations`.

## Результат

Бенчмарки для `internal/repository` до/после оптимизации:

| Benchmark | До | После |
| --- | --- | --- |
| `BenchmarkFileStorageSave` | `3173263 B/op`, `10073 allocs/op` | `504307 B/op`, `54 allocs/op` |
| `BenchmarkFileStorageRestore` | `4803726 B/op`, `30127 allocs/op` | `1560941 B/op`, `30111 allocs/op` |

Проверка diff profile:

```bash
pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
```

Полученный вывод:

```text
File: repository.test
Type: alloc_space
Time: 2026-03-23 11:22:34 MSK
Showing nodes accounting for -185950.58kB, 73.35% of 253521.15kB total
Dropped 129 nodes (cum <= 1267.61kB)
      flat  flat%   sum%        cum   cum%
-86755.44kB 34.22% 34.22% -86755.44kB 34.22%  reflect.growslice
  -32472kB 12.81% 47.03% -66329.14kB 26.16%  encoding/json.MarshalIndent
-23434.99kB  9.24% 56.27% -87647.84kB 34.57%  github.com/squaredbusinessman/go-musthave-metrics/internal/repository.(*FileStorage).Save
  -22568kB  8.90% 65.17%   -22568kB  8.90%  os.readFileContents
-17405.88kB  6.87% 72.04% -17405.88kB  6.87%  bytes.growSlice
  -16368kB  6.46% 78.50% -33827.21kB 13.34%  encoding/json.Marshal
14530.52kB  5.73% 72.76% -98183.37kB 38.73%  github.com/squaredbusinessman/go-musthave-metrics/internal/repository.(*FileStorage).Restore
-4763.70kB  1.88% 74.64% -4763.70kB  1.88%  reflect.New
 2112.09kB  0.83% 73.81%  2112.09kB  0.83%  bufio.NewWriterSize (inline)
 1176.11kB  0.46% 73.35% -3587.59kB  1.42%  encoding/json.(*decodeState).literalStore
   -4.36kB 0.0017% 73.35% -96402.75kB 38.03%  encoding/json.Unmarshal
    4.12kB 0.0016% 73.35%  2116.22kB  0.83%  github.com/squaredbusinessman/go-musthave-metrics/internal/repository.writeMetricsJSON
   -1.06kB 0.00042% 73.35% -17406.94kB  6.87%  bytes.(*Buffer).grow
         0     0% 73.35% -9322.88kB  3.68%  bytes.(*Buffer).Write
         0     0% 73.35% -1674.06kB  0.66%  bytes.(*Buffer).WriteByte
         0     0% 73.35%    -6410kB  2.53%  bytes.(*Buffer).WriteString
         0     0% 73.35%  6104.42kB  2.41%  encoding/json.(*Decoder).Decode
         0     0% 73.35% -96397.50kB 38.02%  encoding/json.(*decodeState).array
         0     0% 73.35% -3584.62kB  1.41%  encoding/json.(*decodeState).object
         0     0% 73.35% -90339.72kB 35.63%  encoding/json.(*decodeState).unmarshal
         0     0% 73.35% -90340.06kB 35.63%  encoding/json.(*decodeState).value
         0     0% 73.35% -17414.13kB  6.87%  encoding/json.(*encodeState).marshal
         0     0% 73.35% -17414.13kB  6.87%  encoding/json.(*encodeState).reflectValue
         0     0% 73.35% -17408.96kB  6.87%  encoding/json.arrayEncoder.encode
         0     0% 73.35%  -968.14kB  0.38%  encoding/json.floatEncoder.encode
         0     0% 73.35% -4763.70kB  1.88%  encoding/json.indirect
         0     0% 73.35%  -968.14kB  0.38%  encoding/json.ptrEncoder.encode
         0     0% 73.35% -17408.96kB  6.87%  encoding/json.sliceEncoder.encode
         0     0% 73.35% -8356.76kB  3.30%  encoding/json.stringEncoder
         0     0% 73.35% -17141.90kB  6.76%  encoding/json.structEncoder.encode
         0     0% 73.35% -104490.94kB 41.22%  github.com/squaredbusinessman/go-musthave-metrics/internal/repository.BenchmarkFileStorageRestore
         0     0% 73.35% -81340.53kB 32.08%  github.com/squaredbusinessman/go-musthave-metrics/internal/repository.BenchmarkFileStorageSave
         0     0% 73.35% -22580.84kB  8.91%  os.ReadFile
         0     0% 73.35% -86755.44kB 34.22%  reflect.Value.Grow
         0     0% 73.35% -86755.44kB 34.22%  reflect.Value.grow
         0     0% 73.35% -176354.42kB 69.56%  testing.(*B).launch
         0     0% 73.35% -9477.05kB  3.74%  testing.(*B).run1.func1
         0     0% 73.35% -185797.83kB 73.29%  testing.(*B).runN
```

## Покрытие тестами

Общий coverage profile собран командой:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -n 1
```

Итоговое покрытие:

```text
total:													(statements)		43.6%
```
