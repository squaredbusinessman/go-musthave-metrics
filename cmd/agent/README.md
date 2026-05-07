# cmd/agent

Агент собирает runtime/gopsutil-метрики и отправляет их на сервер.

Запуск через HTTP:

```bash
go run ./cmd/agent -a localhost:8080
```

Запуск через gRPC:

```bash
go run ./cmd/agent -g localhost:3200
```

Основные параметры:

| Флаг | ENV | Назначение |
| --- | --- | --- |
| `-a` | `ADDRESS` | HTTP-адрес сервера |
| `-g` | `GRPC_ADDRESS` | gRPC-адрес сервера |
| `-p` | `POLL_INTERVAL` | интервал сбора метрик |
| `-r` | `REPORT_INTERVAL` | интервал отправки |
| `-f` | `REPORT_FORMAT` | HTTP-формат отправки: `plain` или `json` |
| `-k` | `KEY` | ключ подписи `HashSHA256` |
| `-l` | `RATE_LIMIT` | лимит параллельной отправки |
| `-crypto-key` | `CRYPTO_KEY` | публичный ключ для HTTP-шифрования |

Если `GRPC_ADDRESS` не задан, агент использует HTTP. Если задан, метрики отправляются
батчами через gRPC `Metrics.UpdateMetrics`.
