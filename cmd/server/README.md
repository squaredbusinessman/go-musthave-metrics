# cmd/server

Сервер принимает, хранит и отдаёт метрики. HTTP и gRPC используют общий слой
`service.MetricsService`.

Запуск только HTTP-сервера:

```bash
go run ./cmd/server -a :8080
```

Запуск HTTP и gRPC серверов:

```bash
go run ./cmd/server -a :8080 -g :3200
```

Основные параметры:

| Флаг | ENV | Назначение |
| --- | --- | --- |
| `-a` | `ADDRESS` | HTTP-адрес сервера |
| `-g` | `GRPC_ADDRESS` | gRPC-адрес сервера |
| `-l` | `LOG_LEVEL` | уровень логирования |
| `-k` | `KEY` | ключ подписи `HashSHA256` |
| `-t` | `TRUSTED_SUBNET` | доверенная подсеть в CIDR |
| `-i` | `STORE_INTERVAL` | интервал сохранения в файл |
| `-f` | `FILE_STORAGE_PATH` | путь к файлу хранения |
| `-r` | `RESTORE` | восстановление из файла при старте |
| `-d` | `DATABASE_DSN` | строка подключения к PostgreSQL |
| `-crypto-key` | `CRYPTO_KEY` | приватный ключ для HTTP-расшифровки |
| `--audit-file` | `AUDIT_FILE` | файл аудита |
| `--audit-url` | `AUDIT_URL` | URL внешнего приёмника аудита |

Если задан `TRUSTED_SUBNET`, HTTP middleware проверяет заголовок `X-Real-IP`, а gRPC
interceptor проверяет metadata `x-real-ip`. Запросы на запись метрик вне подсети
отклоняются.

Если `GRPC_ADDRESS` не задан, gRPC listener не запускается.
