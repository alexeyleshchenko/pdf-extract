# Key components

| Area | Location |
|------|----------|
| Entry | [cmd/pdf-extract/main.go](../../cmd/pdf-extract/main.go) |
| Logging | [internal/logging/](../../internal/logging/) — `LOG_LEVEL`, JSON stdout |
| HTTP | [internal/httpserver/](../../internal/httpserver/) — access `http_request`, process `process_ok` / `process_failed` |
| Config | [internal/config/config.go](../../internal/config/config.go) |
| CI | [.github/workflows/deploy.yml](../../.github/workflows/deploy.yml) |
| Compose | [docker-compose.yml](../../docker-compose.yml) |
| VDS layout | `/root/services/pdf-extract` (compose + `.env`) |
| Sync helper | [scripts/sync-vds-service.sh](../../scripts/sync-vds-service.sh) |
