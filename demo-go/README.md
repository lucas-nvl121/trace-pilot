## Demo Go App

A HTTP service exposing Prometheus metrics, built with Gin and Uber Fx. Includes health and readiness endpoints for Kubernetes.

### Features
- **HTTP framework**: Gin (`/` returns JSON)
- **Dependency injection & lifecycle**: Uber Fx (graceful shutdown)
- **Observability**: JSON `slog` logs, OpenTelemetry HTTP traces, and Prometheus metrics at `/metrics`
- **Probes**: `/healthz` (liveness), `/readyz` (readiness)

### Requirements
- Go 1.24+

### Run locally
```bash
cd demo-go
go run .
# in another shell
curl -s http://localhost:8080/            # {"message":"Hello World!"}
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/healthz
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/readyz
curl -s http://localhost:8080/metrics | head
```

By default, traces are exported to an OTLP gRPC collector at
`localhost:4317`. Configure the service with environment variables:

```bash
SERVICE_NAME=demo-go
SERVICE_VERSION=dev
HTTP_ADDR=:8080
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
OTEL_EXPORTER_OTLP_INSECURE=true
```

Request logs are JSON written to stdout. When tracing is active, HTTP request
logs include `trace_id` and `span_id` fields for correlation with exported
spans. The application does not require a collector to start, but spans cannot
be exported until one is available.

Build a local binary:
```bash
go build -o bin/demo-go ./...
./bin/demo-go
```

### Endpoints
- `GET /` → `{ "message": "Hello World!" }`
- `GET /healthz` → 200 when alive
- `GET /readyz` → 200 when ready (503 otherwise)
- `GET /metrics` → Prometheus metrics

### Notes
- The server listens on `:8080` by default.
- Readiness is toggled by the app at startup/shutdown via Fx lifecycle hooks.
