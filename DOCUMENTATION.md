# Документация по проекту go-musthave-metrics

## Общее описание
Проект состоит из двух приложений:
- **Агент**: собирает метрики (runtime и gopsutil) и отправляет их на сервер.
- **Сервер**: принимает, хранит и отдаёт метрики (в памяти, файле или PostgreSQL).

Передача метрик от агента к серверу поддерживает gzip-сжатие, подпись `HashSHA256`
и опциональное асимметричное шифрование. Для шифрования используется гибридная
схема: тело запроса шифруется случайным AES-256-GCM ключом, а сам AES-ключ
шифруется публичным RSA-ключом через RSA-OAEP/SHA-256. Такой подход выбран,
потому что RSA не подходит для шифрования больших JSON/gzip payload целиком:
размер RSA-блока ограничен длиной ключа и схемой padding. AES-GCM эффективно
шифрует данные произвольного размера и одновременно проверяет целостность
ciphertext, а RSA-OAEP безопасно передаёт одноразовый симметричный ключ.

Порядок обработки запроса:
- агент формирует JSON payload метрик;
- при наличии `KEY` считает `HashSHA256` по исходному JSON payload;
- сжимает payload через gzip;
- при наличии публичного ключа шифрует gzip-body и ставит заголовок
  `Content-Encryption: rsa-aes-gcm`;
- сервер сначала расшифровывает body, затем распаковывает gzip, затем проверяет
  `HashSHA256` и передаёт JSON в handler.

Ключи передаются через флаг `-crypto-key` или переменную окружения `CRYPTO_KEY`:
агент получает путь к публичному ключу, сервер получает путь к приватному ключу.

---

## cmd/agent
Назначение: запуск агента и загрузка конфигурации.

Ссылка на пакет: `cmd/agent/`

### Структуры
- `type Config` — конфигурация агента. Поля:
  - `Addr` (string): адрес сервера.
  - `PollInterval` (int): интервал опроса runtime/gopsutil метрик в секундах.
  - `ReportInterval` (int): интервал отправки метрик в секундах.
  - `ReportFormat` (string): формат отправки (`plain` или `json`).
  - `Key` (string): ключ подписи (HashSHA256).
  - `RateLimit` (int): лимит параллельных исходящих запросов.
  - `CryptoKey` (string): путь к публичному ключу для шифрования запросов.
  - Код: [cmd/agent/flags.go](cmd/agent/flags.go)

### Функции
- `func main()`
  - Запускает агент: инициализация логгера, парсинг конфигурации, загрузка публичного ключа шифрования, создание worker pool, запуск горутин сбора и отправки.
  - Вход: конфигурация через флаги и ENV.
  - Выход: бесконечная работа агента, ошибки критичны (log.Fatalf).
  - Код: [cmd/agent/main.go](cmd/agent/main.go)

- `func parseConfig() Config`
  - Читает флаги и env, формирует `Config`.
  - Вход: флаги CLI и переменные окружения.
  - Выход: заполненная структура `Config`.
  - Код: [cmd/agent/flags.go](cmd/agent/flags.go)

- `func normalizeReportFormat(value string) string`
  - Приводит формат к `plain` или `json`.
  - Вход: строка формата.
  - Выход: нормализованная строка формата.
  - Код: [cmd/agent/flags.go](cmd/agent/flags.go)

---

## cmd/server
Назначение: запуск сервера и загрузка конфигурации.

Ссылка на пакет: `cmd/server/`

### Структуры
- `type Config` — корневая конфигурация сервера.
  - `Server` (ServerConfig), `Storage` (StorageConfig), `Database` (DBConfig), `Audit` (AuditConfig), `Crypto` (CryptoConfig)
  - Код: [cmd/server/flags.go](cmd/server/flags.go)

- `type ServerConfig`
  - `RunAddr` (string): адрес сервера.
  - `LogLevel` (string): уровень логирования.
  - `Key` (string): ключ подписи (HashSHA256).
  - Код: [cmd/server/flags.go](cmd/server/flags.go)

- `type StorageConfig`
  - `StoreInterval` (int): период записи в файл (сек).
  - `FileStorageEnabled` (bool): активен ли файл.
  - `FileStoragePath` (string): путь к файлу.
  - `Restore` (bool): восстанавливать метрики при старте.
  - Код: [cmd/server/flags.go](cmd/server/flags.go)

- `type DBConfig`
  - `DSN` (string): строка подключения к БД.
  - Код: [cmd/server/flags.go](cmd/server/flags.go)

- `type AuditConfig`
  - `FilePath` (string): путь к файлу аудита.
  - `URL` (string): URL внешнего приёмника аудита.
  - Код: [cmd/server/flags.go](cmd/server/flags.go)

- `type CryptoConfig`
  - `KeyPath` (string): путь к приватному ключу для расшифровки запросов.
  - Код: [cmd/server/flags.go](cmd/server/flags.go)

### Функции
- `func main()`
  - Инициализирует логгер, загружает приватный ключ шифрования, выбирает хранилище (mem/file/pg), поднимает HTTP-сервер, подключает middleware.
  - Вход: конфигурация через флаги и ENV.
  - Выход: слушает HTTP, завершает работу при фатальных ошибках.
  - Код: [cmd/server/main.go](cmd/server/main.go)

- `func buildRouter(h *handler.Handler, key string, privateKey *rsa.PrivateKey) http.Handler`
  - Собирает HTTP router и подключает middleware в порядке `Crypto -> Gzip -> Hash -> Handler`.
  - Вход: handler, ключ подписи, приватный ключ расшифровки.
  - Выход: готовый `http.Handler`.
  - Код: [cmd/server/main.go](cmd/server/main.go)

- `func parseConfig() Config`
  - Читает флаги и env, формирует `Config`.
  - Вход: флаги CLI и переменные окружения.
  - Выход: заполненная структура `Config`.
  - Код: [cmd/server/flags.go](cmd/server/flags.go)

---

## internal/agent
Назначение: сбор метрик, упаковка и отправка на сервер, worker pool.

Ссылка на пакет: `internal/agent/`

### Функции (сбор и отправка)
- `func CollectRuntimeMetrics(store *storage.MemStorage, rnd *rand.Rand)`
  - Снимает runtime метрики и пишет в `MemStorage`.
  - Вход: хранилище, генератор случайных чисел.
  - Выход: обновлённые значения в `store`.
  - Код: [internal/agent/metrics_collector.go](internal/agent/metrics_collector.go)

- `func ColletGopsutilMetrics(store *storage.MemStorage)`
  - Снимает `TotalMemory`, `FreeMemory` и `CPUutilization1..N`, пишет в `MemStorage`.
  - Вход: хранилище.
  - Выход: обновлённые значения в `store`.
  - Код: [internal/agent/metrics_collector_psutil.go](internal/agent/metrics_collector_psutil.go)

- `func SendMetric(client *resty.Client, m models.Metric, key string) error`
  - Отправляет одну метрику в `text/plain` по пути `/update/{type}/{name}/{value}`.
  - Вход: REST клиент, структура `Metric`, ключ подписи.
  - Выход: `error` при сетевых/HTTP ошибках.
  - Код: [internal/agent/report_metrics.go](internal/agent/report_metrics.go)

- `func sendMetricsBatchJSON(client *resty.Client, metrics []models.Metrics, key string, publicKey *rsa.PublicKey) error`
  - Отправляет батч метрик в JSON на `/updates`: сериализует payload, сжимает gzip и при наличии публичного ключа шифрует тело запроса.
  - Вход: REST клиент, список `Metrics`, ключ подписи, публичный ключ шифрования.
  - Выход: `error` при сетевых/HTTP ошибках.
  - Код: [internal/agent/report_metrics.go](internal/agent/report_metrics.go)

- `func ReportMetrics(client *resty.Client, store *storage.MemStorage, reportFormat string, key string, publicKey *rsa.PublicKey, jobs chan<- Job)`
  - Снимает snapshot из `MemStorage` и ставит задачи отправки в канал `jobs`.
  - Если публичный ключ задан, принудительно использует JSON batch-режим, потому что plain-режим передаёт значения метрик в URL и не может быть зашифрован как body.
  - Вход: REST клиент, хранилище, формат, ключ подписи, публичный ключ шифрования, канал задач.
  - Выход: задачи в очередь; ошибки логируются воркерами.
  - Код: [internal/agent/report_metrics.go](internal/agent/report_metrics.go)

- `func snapshotToMetrics(gauges map[string]models.Gauge, counters map[string]models.Counter) []models.Metrics`
  - Конвертирует snapshot в список `Metrics`.
  - Вход: карты gauge/counter.
  - Выход: список метрик.
  - Код: [internal/agent/report_metrics.go](internal/agent/report_metrics.go)

- `func gzipPayload(payload []byte) ([]byte, error)`
  - Сжимает payload в gzip.
  - Вход: байты.
  - Выход: gzip-байты или ошибка.
  - Код: [internal/agent/report_metrics.go](internal/agent/report_metrics.go)

- `func isRetryableNetErr(err error) bool`
  - Определяет, стоит ли ретраить сетевую ошибку.
  - Вход: ошибка.
  - Выход: `true`, если ретраить.
  - Код: [internal/agent/report_metrics.go](internal/agent/report_metrics.go)

- `func normalizeReportFormat(format string) string`
  - Нормализует формат отчёта.
  - Вход: строка формата.
  - Выход: `plain` или `json`.
  - Код: [internal/agent/report_metrics.go](internal/agent/report_metrics.go)

### Шифрование исходящих запросов
- При заданном публичном ключе агент добавляет заголовок `Content-Encryption: rsa-aes-gcm`.
- Тело HTTP-запроса содержит JSON-envelope с полями `key`, `nonce`, `data`.
- Поле `key` содержит RSA-OAEP encrypted AES key в base64.
- Поля `nonce` и `data` содержат параметры AES-GCM в base64.
- Заголовок `Content-Encoding: gzip` сохраняется, потому что после расшифровки сервер получает gzip-body и передаёт его в `GzipMiddleware`.

### Хеширование
- `func sha256hex(body []byte, key string) string`
  - Считает `SHA256(body + key)` и возвращает hex.
  - Вход: payload и ключ.
  - Выход: hex-строка.
  - Код: [internal/agent/hash.go](internal/agent/hash.go)

### Gzip helpers
- `type CompressWriter`
  - Обёртка `http.ResponseWriter`, пишет gzip.
  - Код: [internal/agent/gzip.go](internal/agent/gzip.go)

- `func NewCompressWriter(writer http.ResponseWriter) *CompressWriter`
  - Вход: исходный writer.
  - Выход: gzip writer-обёртка.
  - Код: [internal/agent/gzip.go](internal/agent/gzip.go)

- `func (cw *CompressWriter) Header() http.Header`
- `func (cw *CompressWriter) Write(data []byte) (int, error)`
- `func (cw *CompressWriter) WriteHeader(statusCode int)`
- `func (cw *CompressWriter) Close() error`
  - Реализация `http.ResponseWriter` + управление gzip.
  - Код: [internal/agent/gzip.go](internal/agent/gzip.go)

- `type CompressReader`
  - Обёртка `io.ReadCloser`, читает gzip.
  - Код: [internal/agent/gzip.go](internal/agent/gzip.go)

- `func NewCompressReader(reader io.ReadCloser) (*CompressReader, error)`
- `func (cr *CompressReader) Read(data []byte) (int, error)`
- `func (cr *CompressReader) Close() error`
  - Реализация `io.ReadCloser` для распаковки gzip.
  - Код: [internal/agent/gzip.go](internal/agent/gzip.go)

### Worker pool
- `type Job func() error`
  - Тип задачи для пула воркеров.
  - Код: [internal/agent/worker_pool.go](internal/agent/worker_pool.go)

- `func StartWorkers(n int, jobs <-chan Job)`
  - Запускает `n` воркеров, читающих задачи из канала.
  - Вход: число воркеров, канал задач.
  - Выход: горутины, выполняющие `job()`; ошибки логируются.
  - Код: [internal/agent/worker_pool.go](internal/agent/worker_pool.go)

---

## internal/cryptoutil
Назначение: загрузка RSA-ключей и гибридное шифрование запросов агента.

Ссылка на пакет: `internal/cryptoutil/`

### Структуры
- `type Envelope`
  - JSON-обёртка зашифрованного payload.
  - `EncryptedKey` (`json:"key"`): AES-ключ, зашифрованный публичным RSA-ключом и закодированный в base64.
  - `Nonce` (`json:"nonce"`): nonce для AES-GCM в base64.
  - `Data` (`json:"data"`): ciphertext AES-GCM в base64.
  - Код: [internal/cryptoutil/cryptoutil.go](internal/cryptoutil/cryptoutil.go)

### Функции
- `func LoadPublicKey(path string) (*rsa.PublicKey, error)`
  - Читает публичный RSA-ключ из PEM-файла.
  - Поддерживает блоки `PUBLIC KEY` (PKIX) и `RSA PUBLIC KEY` (PKCS#1).
  - Вход: путь к файлу.
  - Выход: публичный RSA-ключ или ошибка.
  - Код: [internal/cryptoutil/cryptoutil.go](internal/cryptoutil/cryptoutil.go)

- `func LoadPrivateKey(path string) (*rsa.PrivateKey, error)`
  - Читает приватный RSA-ключ из PEM-файла.
  - Поддерживает блоки `PRIVATE KEY` (PKCS#8) и `RSA PRIVATE KEY` (PKCS#1).
  - Вход: путь к файлу.
  - Выход: приватный RSA-ключ или ошибка.
  - Код: [internal/cryptoutil/cryptoutil.go](internal/cryptoutil/cryptoutil.go)

- `func Encrypt(publicKey *rsa.PublicKey, plainText []byte) ([]byte, error)`
  - Генерирует одноразовый AES-256 ключ, шифрует `plainText` через AES-GCM, шифрует AES-ключ через RSA-OAEP/SHA-256 и возвращает JSON-envelope.
  - Вход: публичный RSA-ключ и открытый payload.
  - Выход: JSON-envelope или ошибка.
  - Код: [internal/cryptoutil/cryptoutil.go](internal/cryptoutil/cryptoutil.go)

- `func Decrypt(privateKey *rsa.PrivateKey, payload []byte) ([]byte, error)`
  - Читает JSON-envelope, расшифровывает AES-ключ приватным RSA-ключом через RSA-OAEP/SHA-256 и расшифровывает `data` через AES-GCM.
  - Вход: приватный RSA-ключ и JSON-envelope.
  - Выход: открытый payload или ошибка.
  - Код: [internal/cryptoutil/cryptoutil.go](internal/cryptoutil/cryptoutil.go)

### Почему гибридная схема
- RSA используется только для шифрования короткого одноразового AES-ключа.
- AES-GCM используется для основного тела запроса, потому что размер JSON/gzip payload не ограничен RSA-блоком.
- GCM дополнительно проверяет целостность ciphertext при расшифровке.
- RSA-OAEP с SHA-256 выбран вместо устаревших схем padding, чтобы использовать современную стандартную схему асимметричного шифрования из `crypto/rsa`.

---

## internal/apperr
Назначение: ошибки домена и преобразование в HTTP-ответы.

Ссылка на пакет: `internal/apperr/`

### Переменные
- `ErrUnknownMetricType`, `ErrBadMetricValue`, `ErrMetricNotFound`
  - Доменные ошибки.
  - Код: [internal/apperr/errorCodes.go](internal/apperr/errorCodes.go)

### Функции
- `func WriteServiceError(w http.ResponseWriter, err error)`
  - Маппит доменную ошибку на HTTP-статус и пишет ответ.
  - Вход: writer, ошибка.
  - Выход: HTTP-ответ с нужным кодом.
  - Код: [internal/apperr/handlerErr.go](internal/apperr/handlerErr.go)

---

## internal/handler
Назначение: HTTP-хендлеры сервера.

Ссылка на пакет: `internal/handler/`

### Структуры и интерфейсы
- `type Handler`
  - Поля: `ms` (MetricsService), `db` (DBPinger).
  - Код: [internal/handler/handler.go](internal/handler/handler.go)

- `type DBPinger interface`
  - `Ping(ctx context.Context) error`
  - Код: [internal/handler/dbping.go](internal/handler/dbping.go)

### Функции
- `func New(ms service.MetricsService, db DBPinger) *Handler`
  - Создаёт хендлер.
  - Вход: сервис метрик и DB пингер.
  - Выход: `*Handler`.
  - Код: [internal/handler/handler.go](internal/handler/handler.go)

- `func AcceptMetricsToStorage(w http.ResponseWriter, r *http.Request)`
  - Принимает метрику в `text/plain`.
  - Метод: POST.
  - Путь: `/update/{type}/{name}/{value}`.
  - Выход: HTTP 200 при успехе, иначе ошибка.
  - Код: [internal/handler/handler.go](internal/handler/handler.go)

- `func UpdateMetricJSON(writer http.ResponseWriter, request *http.Request)`
  - Принимает метрику в JSON.
  - Метод: POST.
  - Content-Type: `application/json`.
  - Тело: `Metrics` (`id`, `type`, `delta|value`).
  - Ответ: JSON с сохранённой метрикой.
  - Код: [internal/handler/handler.go](internal/handler/handler.go)

- `func UpdateMetricsBatch(writer http.ResponseWriter, request *http.Request)`
  - Принимает батч метрик.
  - Метод: POST.
  - Content-Type: `application/json`.
  - Тело: массив `[]Metrics`.
  - Ответ: HTTP 200.
  - Код: [internal/handler/handler.go](internal/handler/handler.go)

- `func GetMetric(w http.ResponseWriter, r *http.Request)`
  - Возвращает значение метрики в `text/plain`.
  - Метод: GET.
  - Путь: `/value/{type}/{name}`.
  - Ответ: строковое значение.
  - Код: [internal/handler/handler.go](internal/handler/handler.go)

- `func GetMetricJSON(writer http.ResponseWriter, request *http.Request)`
  - Возвращает значение метрики в JSON.
  - Метод: POST.
  - Content-Type: `application/json`.
  - Тело: `Metrics` с `id` и `type`.
  - Ответ: `Metrics` с заполненным `value|delta`.
  - Код: [internal/handler/handler.go](internal/handler/handler.go)

- `func GetAllMetrics(w http.ResponseWriter, r *http.Request)`
  - Возвращает все метрики в HTML-таблице.
  - Метод: GET.
  - Ответ: HTML.
  - Код: [internal/handler/handler.go](internal/handler/handler.go)

- `func Ping(w http.ResponseWriter, r *http.Request)`
  - Проверка соединения с БД.
  - Метод: GET.
  - Ответ: 200, если БД доступна.
  - Код: [internal/handler/dbping.go](internal/handler/dbping.go)

---

## internal/logger
Назначение: общий логгер и обёртка ResponseWriter.

Ссылка на пакет: `internal/logger/`

### Структуры
- `type LoggingWriter`
  - Поля: `ResponseWriter`, `Status`, `Bytes`.
  - Код: [internal/logger/logger.go](internal/logger/logger.go)

### Функции/методы
- `func (lw *LoggingWriter) WriteHeader(code int)`
  - Запоминает статус и проксирует в оригинальный writer.
  - Код: [internal/logger/logger.go](internal/logger/logger.go)

- `func (lw *LoggingWriter) Write(b []byte) (int, error)`
  - Запоминает количество байт и проксирует запись.
  - Код: [internal/logger/logger.go](internal/logger/logger.go)

- `func Initialize(level string) error`
  - Настраивает глобальный `Log`.
  - Вход: строка уровня.
  - Выход: ошибка инициализации (если есть).
  - Код: [internal/logger/logger.go](internal/logger/logger.go)

---

## internal/middleware
Назначение: middleware HTTP-сервера.

Ссылка на пакет: `internal/middleware/`

### Функции
- `func Conveyor(h http.Handler, middlewares ...Middleware) http.Handler`
  - Последовательно оборачивает хендлер в middleware.
  - Вход: `http.Handler`, список `Middleware`.
  - Выход: обёрнутый handler.
  - Код: [internal/middleware/middleware.go](internal/middleware/middleware.go)

- `func RequestLogger(next http.Handler) http.Handler`
  - Логирует метод/путь/статус/время.
  - Вход: handler.
  - Выход: handler с логированием.
  - Код: [internal/middleware/middleware.go](internal/middleware/middleware.go)

- `func GzipMiddleware(next http.Handler) http.Handler`
  - Если клиент поддерживает gzip — сжимает ответ.
  - Если запрос gzipped — распаковывает тело.
  - Код: [internal/middleware/middleware.go](internal/middleware/middleware.go)

- `func CryptoMiddleware(privateKey *rsa.PrivateKey) Middleware`
  - Если в запросе есть `Content-Encryption: rsa-aes-gcm`, читает encrypted envelope из body, расшифровывает его приватным ключом и заменяет `request.Body` на расшифрованные байты.
  - Должен выполняться до `GzipMiddleware`, потому что агент шифрует уже gzip-сжатое тело.
  - Если заголовок шифрования отсутствует, пропускает запрос без изменений.
  - Код: [internal/middleware/middleware.go](internal/middleware/middleware.go)

- `func HashMiddleware(key string) Middleware`
  - Проверяет подпись запроса (если `HashSHA256` заголовок присутствует).
  - Подписывает ответ `HashSHA256`, если `key` задан.
  - Код: [internal/middleware/middleware.go](internal/middleware/middleware.go)

### Вспомогательные функции/структуры
- `func sha256hex(body []byte, key string) string`
  - Считает `SHA256(body + key)` и возвращает hex.
  - Код: [internal/middleware/hash_utils.go](internal/middleware/hash_utils.go)

- `type ResponseRecorder`
  - Буферизует ответ и заголовки для вычисления подписи.
  - Код: [internal/middleware/hash_utils.go](internal/middleware/hash_utils.go)

---

## internal/model
Назначение: доменные структуры метрик.

Ссылка на пакет: `internal/model/`

### Структуры
- `type Gauge` / `type Counter`
  - Поля: `Value`.
  - Код: [internal/model/metrics.go](internal/model/metrics.go)

- `type Metrics`
  - JSON-структура: `id`, `type`, `delta`, `value`, `hash`.
  - Используется в JSON API.
  - Код: [internal/model/metrics.go](internal/model/metrics.go)

- `type Metric`
  - Структура для plain API: `Type`, `Name`, `Value`.
  - Код: [internal/model/metrics.go](internal/model/metrics.go)

### Методы
- `func (g *Gauge) NewGaugeReplace(val float64)`
  - Устанавливает значение gauge.
  - Код: [internal/model/metrics.go](internal/model/metrics.go)

- `func (c *Counter) NewValueIncrement(val int64)`
  - Инкрементирует counter.
  - Код: [internal/model/metrics.go](internal/model/metrics.go)

---

## internal/repository
Назначение: слой хранения метрик (memory/file/postgres).

Ссылка на пакет: `internal/repository/`

### Интерфейсы и структуры
- `type Storage`
  - Базовый интерфейс хранилища.
  - Код: [internal/repository/server_storage.go](internal/repository/server_storage.go)

- `type MemStorage`
  - Хранилище в памяти, потокобезопасно.
  - Код: [internal/repository/server_storage.go](internal/repository/server_storage.go)

- `type FileStorage`
  - Хранилище в файле (оборачивает Storage).
  - Код: [internal/repository/file_storage.go](internal/repository/file_storage.go)

- `type DBStorage`
  - Хранилище в PostgreSQL.
  - Код: [internal/repository/db_storage.go](internal/repository/db_storage.go)

### Функции и методы MemStorage
- `func NewMemStorage() *MemStorage`
  - Создаёт пустое in-memory хранилище.
  - Код: [internal/repository/server_storage.go](internal/repository/server_storage.go)

- `func (ms *MemStorage) SetGauge(ctx context.Context, name string, value models.Gauge) error`
- `func (ms *MemStorage) AddCounter(ctx context.Context, name string, value int64) error`
- `func (ms *MemStorage) GetGauge(ctx context.Context, name string) (float64, error)`
- `func (ms *MemStorage) GetCounter(ctx context.Context, name string) (int64, error)`
- `func (ms *MemStorage) Snapshot(ctx context.Context) (map[string]models.Gauge, map[string]models.Counter, error)`
- `func (ms *MemStorage) UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error`
  - CRUD над метриками в памяти.
  - Код: [internal/repository/server_storage.go](internal/repository/server_storage.go)

### Функции и методы FileStorage
- `func NewFileStorage(path string, store Storage) *FileStorage`
  - Создаёт file storage поверх `Storage`.
  - Код: [internal/repository/file_storage.go](internal/repository/file_storage.go)

- `func (fs *FileStorage) Save() error`
  - Сериализует метрики в JSON и пишет в файл.
  - Код: [internal/repository/file_storage.go](internal/repository/file_storage.go)

- `func (fs *FileStorage) Restore() error`
  - Читает метрики из файла и записывает в `Storage`.
  - Код: [internal/repository/file_storage.go](internal/repository/file_storage.go)

### Функции и методы DBStorage
- `func NewDBStorage(pool *pgxpool.Pool) *DBStorage`
  - Создаёт storage для PostgreSQL.
  - Код: [internal/repository/db_storage.go](internal/repository/db_storage.go)

- `func (db *DBStorage) SetGauge(ctx context.Context, name string, value models.Gauge) error`
- `func (db *DBStorage) AddCounter(ctx context.Context, name string, value int64) error`
- `func (db *DBStorage) GetGauge(ctx context.Context, name string) (float64, error)`
- `func (db *DBStorage) GetCounter(ctx context.Context, name string) (int64, error)`
- `func (db *DBStorage) Snapshot(ctx context.Context) (map[string]models.Gauge, map[string]models.Counter, error)`
- `func (db *DBStorage) UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error`
  - Полная реализация `Storage` на PostgreSQL.
  - Код: [internal/repository/db_storage.go](internal/repository/db_storage.go)

### Вспомогательные функции DBStorage
- `func buildUpsertGauge(name string, value float64) (string, []interface{}, error)`
- `func buildUpsertCounter(name string, value int64) (string, []interface{}, error)`
- `func buildGetGauge(name string) (string, []interface{}, error)`
- `func buildGetCounter(name string) (string, []interface{}, error)`
- `func buildSnapshotGauges() (string, []interface{}, error)`
- `func buildSnapshotCounters() (string, []interface{}, error)`
- `func isRetryablePGErr(err error) bool`
  - Генерация SQL и политика retry по PG ошибкам.
  - Код: [internal/repository/db_storage.go](internal/repository/db_storage.go)

---

## internal/retry
Назначение: общий механизм retry с backoff.

Ссылка на пакет: `internal/retry/`

### Функции
- `func Do(ctx context.Context, isRetryable func(error) bool, op func() error) error`
  - Запускает операцию с retry по заданной функции `isRetryable`.
  - Вход: контекст, функция проверки ошибки, операция.
  - Выход: ошибка последней попытки.
  - Код: [internal/retry/retry.go](internal/retry/retry.go)

- `func wait(ctx context.Context, d time.Duration) error`
  - Ожидание с учётом отмены контекста.
  - Код: [internal/retry/retry.go](internal/retry/retry.go)

---

## internal/service
Назначение: бизнес-логика метрик (валидация + репозиторий).

Ссылка на пакет: `internal/service/`

### Интерфейсы и структуры
- `type MetricsService`
  - Контракт бизнес-логики.
  - Код: [internal/service/metrics.go](internal/service/metrics.go)

- `type metricsService`
  - Реализация `MetricsService`.
  - Код: [internal/service/metrics.go](internal/service/metrics.go)

- `type MetricsServiceOption`
  - Опция конфигурации сервиса.
  - Код: [internal/service/metrics.go](internal/service/metrics.go)

### Функции и методы
- `func WithAfterUpdate(hook func()) MetricsServiceOption`
  - Устанавливает callback после обновления метрик.
  - Код: [internal/service/metrics.go](internal/service/metrics.go)

- `func NewMetricsService(store repository.Storage, opts ...MetricsServiceOption) MetricsService`
  - Создаёт сервис метрик.
  - Вход: хранилище и опции.
  - Выход: реализация `MetricsService`.
  - Код: [internal/service/metrics.go](internal/service/metrics.go)

- `func (s *metricsService) UpdateMetric(ctx context.Context, m models.Metric) error`
  - Обновление метрики из plain API.
  - Вход: `Metric` с типом/именем/значением.
  - Выход: ошибка валидации или хранения.
  - Код: [internal/service/metrics.go](internal/service/metrics.go)

- `func (s *metricsService) UpdateMetricJSON(ctx context.Context, m models.Metrics) error`
  - Обновление метрики из JSON API.
  - Вход: `Metrics` с `id`, `type`, `delta|value`.
  - Выход: ошибка валидации или хранения.
  - Код: [internal/service/metrics.go](internal/service/metrics.go)

- `func (s *metricsService) UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error`
  - Батч-обновление метрик.
  - Вход: массив `Metrics`.
  - Выход: ошибка валидации/хранения.
  - Код: [internal/service/metrics.go](internal/service/metrics.go)

- `func (s *metricsService) GetMetric(ctx context.Context, m models.Metric) (string, error)`
  - Получение метрики (plain API).
  - Вход: `Metric` с `type`, `name`.
  - Выход: строковое значение или ошибка.
  - Код: [internal/service/metrics.go](internal/service/metrics.go)

- `func (s *metricsService) GetMetricJSON(ctx context.Context, m models.Metrics) (*models.Metrics, error)`
  - Получение метрики (JSON API).
  - Вход: `Metrics` с `id`, `type`.
  - Выход: `Metrics` с `value|delta` или ошибка.
  - Код: [internal/service/metrics.go](internal/service/metrics.go)

- `func (s *metricsService) GetAllMetrics(ctx context.Context) (map[string]models.Gauge, map[string]models.Counter, error)`
  - Возвращает snapshot всех метрик.
  - Код: [internal/service/metrics.go](internal/service/metrics.go)

- `func (s *metricsService) triggerAfterUpdate()`
  - Вызывает callback после обновления метрик.
  - Код: [internal/service/metrics.go](internal/service/metrics.go)

---

## migrations
Назначение: миграции PostgreSQL.

Ссылка на пакет: `migrations/`

### Функции
- `func Up(pool *pgxpool.Pool, dir string) error`
  - Запускает миграции из указанной директории.
  - Вход: пул pgx, путь к миграциям.
  - Выход: ошибка миграции (если есть).
  - Код: [migrations/migrations.go](migrations/migrations.go)

---

## Покрытие тестами
Данные получены из `go test ./... -coverprofile=coverage.out` и `go tool cover -func=coverage.out`.

**Общее покрытие проекта:** 27.4% (statements).

**По пакетам:**
- `cmd/agent`: 0.0%
- `cmd/server`: 0.0%
- `internal/agent`: 55.1%
- `internal/apperr`: 0.0%
- `internal/handler`: 24.1%
- `internal/logger`: 0.0%
- `internal/middleware`: 23.2%
- `internal/model`: 100.0%
- `internal/repository`: 30.8%
- `internal/retry`: 73.7%
- `internal/service`: 0.0%
- `migrations`: 0.0%

---

## TODO
1) **Тесты:**
   - Добавить тесты для `internal/service` (валидация, ошибки, batch).
   - Покрыть `internal/apperr` и `internal/logger`.
   - Добавить интеграционные тесты HTTP‑хендлеров с middleware.
2) **Worker pool:**
   - Добавить graceful shutdown (закрытие канала jobs, ожидание воркеров).
   - Добавить защиту от переполнения очереди (метрика/лог).
3) **gopsutil:**
   - Логировать ошибки при чтении CPU/mem.
   - Добавить поддержку missing permissions в контейнерах.
4) **Хеш‑подпись:**
   - Опционально сделать строгий режим (требовать `HashSHA256` при наличии ключа).
5) **Хранилища:**
   - Тесты для `FileStorage.Save/Restore` на atomic rename.
