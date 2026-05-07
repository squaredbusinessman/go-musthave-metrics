# go-musthave-metrics

Сервис сбора метрик и алертинга для трека Яндекс Практикума.

Проект состоит из двух приложений:

- `cmd/server` — принимает, хранит и отдаёт метрики через HTTP и gRPC.
- `cmd/agent` — собирает runtime/gopsutil-метрики и отправляет их на сервер.

Метрики можно хранить в памяти, файле или PostgreSQL. HTTP-транспорт поддерживает gzip,
`HashSHA256`, RSA/AES-GCM шифрование и проверку доверенной подсети. gRPC-транспорт
отправляет метрики батчами через `Metrics.UpdateMetrics` и передаёт IP агента в metadata
`x-real-ip`.

## Быстрый запуск

Запустить сервер только с HTTP на `:8080`:

```bash
go run ./cmd/server -a :8080
```

Запустить сервер с HTTP на `:8080` и gRPC на `:3200`:

```bash
go run ./cmd/server -a :8080 -g :3200
```

Запустить агент через HTTP:

```bash
go run ./cmd/agent -a localhost:8080
```

Запустить агент через gRPC:

```bash
go run ./cmd/agent -g localhost:3200
```

Если `-g`/`GRPC_ADDRESS` у сервера не задан, gRPC listener не запускается. Если
`-g`/`GRPC_ADDRESS` у агента не задан, используется HTTP-отправка. Если задан,
агент отправляет батчи через gRPC на указанный адрес.

## Конфигурация сервера

Параметры можно задавать флагами, переменными окружения или JSON-конфигом.

| Назначение | Флаг | ENV | JSON | По умолчанию |
| --- | --- | --- | --- | --- |
| HTTP-адрес | `-a` | `ADDRESS` | `address` | `:8080` |
| gRPC-адрес | `-g` | `GRPC_ADDRESS` | `grpc_address` | пусто |
| уровень логирования | `-l` | `LOG_LEVEL` | `log_level` | `info` |
| ключ подписи | `-k` | `KEY` | `key` | пусто |
| доверенная подсеть | `-t` | `TRUSTED_SUBNET` | `trusted_subnet` | пусто |
| интервал сохранения | `-i` | `STORE_INTERVAL` | `store_interval` | `300` |
| файл хранения | `-f` | `FILE_STORAGE_PATH` | `store_file` | `/tmp/devops-metrics-db.json` |
| восстановление из файла | `-r` | `RESTORE` | `restore` | `true` |
| PostgreSQL DSN | `-d` | `DATABASE_DSN` | `database_dsn` | пусто |
| приватный ключ | `-crypto-key` | `CRYPTO_KEY` | `crypto_key` | пусто |
| файл аудита | `--audit-file` | `AUDIT_FILE` | `audit_file` | пусто |
| URL аудита | `--audit-url` | `AUDIT_URL` | `audit_url` | пусто |

## Конфигурация агента

| Назначение | Флаг | ENV | JSON | По умолчанию |
| --- | --- | --- | --- | --- |
| HTTP-адрес сервера | `-a` | `ADDRESS` | `address` | `:8080` |
| gRPC-адрес сервера | `-g` | `GRPC_ADDRESS` | `grpc_address` | пусто |
| интервал сбора | `-p` | `POLL_INTERVAL` | `poll_interval` | `2` |
| интервал отправки | `-r` | `REPORT_INTERVAL` | `report_interval` | `10` |
| HTTP-формат | `-f` | `REPORT_FORMAT` | `report_format` | `plain` |
| ключ подписи | `-k` | `KEY` | `key` | пусто |
| лимит отправки | `-l` | `RATE_LIMIT` | `rate_limit` | `1` |
| публичный ключ | `-crypto-key` | `CRYPTO_KEY` | `crypto_key` | пусто |

## gRPC

Протокол описан в `internal/proto/metrics.proto`. Сгенерированные Go-файлы лежат рядом:

- `internal/proto/metrics.pb.go`
- `internal/proto/metrics_grpc.pb.go`

Если задан `GRPC_ADDRESS`, сервер реализует `Metrics.UpdateMetrics` и сохраняет данные
через общий слой `service.MetricsService`, поэтому HTTP и gRPC используют одну
бизнес-логику.

Для проверки доверенной подсети агент передаёт свой IP в metadata `x-real-ip`.
Если на сервере задан `TRUSTED_SUBNET`, gRPC `UnaryInterceptor` проверяет этот IP.
При запрете сервер возвращает `codes.PermissionDenied`.

Пример запуска с проверкой подсети:

```bash
go run ./cmd/server -g :3200 -t 192.168.1.0/24
go run ./cmd/agent -g localhost:3200
```

## Форматирование goimports

Отформатировать все Go-файлы проекта можно из корня репозитория командой:

```bash
find . -name '*.go' -type f -print0 | xargs -0 goimports -w
```

При последней проверке `goimports` не внёс изменений: все файлы проекта уже были отформатированы.

## Асимметричное шифрование метрик

Агент и сервер поддерживают опциональное шифрование запросов с метриками.

- Агент принимает путь к публичному ключу через флаг `-crypto-key` или переменную окружения `CRYPTO_KEY`.
- Сервер принимает путь к приватному ключу через флаг `-crypto-key` или переменную окружения `CRYPTO_KEY`.
- Если ключ на агенте задан, агент принудительно использует JSON batch-режим `/updates`, потому что plain-режим передаёт значения метрик в URL и не может быть зашифрован как тело запроса.

### Почему RSA + AES-GCM

Для передачи метрик используется гибридная схема шифрования:

1. Агент сериализует метрики в JSON.
2. Если задан `KEY`, агент считает `HashSHA256` по исходному JSON payload.
3. Payload сжимается через gzip.
4. Для каждого запроса генерируется случайный AES-256 ключ.
5. Gzip-body шифруется через AES-GCM.
6. AES-ключ шифруется публичным RSA-ключом сервера через RSA-OAEP/SHA-256.
7. Сервер приватным RSA-ключом расшифровывает AES-ключ, затем расшифровывает body через AES-GCM.
8. После расшифровки обычный pipeline сервера продолжает обработку: gzip -> hash -> handler.

RSA не используется для шифрования всего тела запроса напрямую, потому что размер данных,
которые можно зашифровать одним RSA-блоком, ограничен длиной ключа и схемой padding.
JSON batch с метриками может быть больше этого лимита. AES-GCM подходит для payload
произвольного размера и дополнительно проверяет целостность ciphertext, а RSA-OAEP
используется только для безопасной передачи одноразового AES-ключа.

Зашифрованное тело запроса передаётся как JSON-envelope:

```json
{
  "key": "base64-rsa-encrypted-aes-key",
  "nonce": "base64-aes-gcm-nonce",
  "data": "base64-aes-gcm-ciphertext"
}
```

Агент помечает такие запросы заголовком:

```text
Content-Encryption: rsa-aes-gcm
```

На сервере `CryptoMiddleware` выполняется до `GzipMiddleware`: сначала encrypted envelope
превращается обратно в gzip-body, затем gzip-body распаковывается и передаётся в
`HashMiddleware` и HTTP handler.

### Ручная проверка

Сгенерировать ключи:

```bash
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem
```

Запустить сервер:

```bash
go run ./cmd/server -crypto-key private.pem
```

Запустить агент:

```bash
go run ./cmd/agent -crypto-key public.pem
```

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
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

Актуальный итог:

```text
total:													(statements)					45.7%
```

Пакетная разбивка приведена в [DOCUMENTATION.md](DOCUMENTATION.md#покрытие-тестами).
