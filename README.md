# Observability Stack

A multi-service observability demonstration with standardized logging, metrics, and tracing across Go, TypeScript, and Elixir services, all collected via OpenTelemetry Collector.

## Services

- **Go Service** (Port 8080) - Go service with Prometheus metrics and OpenTelemetry tracing
- **TypeScript Service** (Port 3000) - Node.js/Express service with Prometheus metrics and OpenTelemetry tracing
- **Elixir Service** (Port 4000) - Elixir/Phoenix service with Prometheus metrics and OpenTelemetry tracing

## Observability Stack

- **OpenTelemetry Collector** (Ports 4317 gRPC, 4318 HTTP)
  - Unified collection point for all logs, metrics, and traces
  - Exports to Datadog (traces, metrics, logs)
- **Datadog** - Cloud-based observability platform
  - Traces: https://app.datadoghq.com/apm/traces
  - Metrics: https://app.datadoghq.com/metric/explorer
  - Logs: https://app.datadoghq.com/logs

## Quick Start

1. **Start all services:**
   ```bash
   docker compose up -d
   ```

2. **Set Datadog API Key:**
   ```bash
   export DD_API_KEY=your_datadog_api_key_here
   export DD_SITE=datadoghq.com
   ```
   Or create a `.env` file with these variables.

3. **Access dashboards:**
   - Datadog: https://app.datadoghq.com

3. **View logs:**
   ```bash
   docker compose logs -f
   ```

4. **Check service endpoints:**
   - Go: http://localhost:8080/metrics (Prometheus format)
   - TypeScript: http://localhost:3000/metrics (Prometheus format)
   - Elixir: http://localhost:4000/metrics (Prometheus format)

## OpenTelemetry Collector

All services send telemetry data to the OpenTelemetry Collector via OTLP:

- **Traces**: Sent via OTLP gRPC → Collector → Jaeger
- **Metrics**: Sent via OTLP gRPC → Collector → Prometheus
- **Logs**: Structured JSON logs (can be sent via OTLP in future)

### Environment Variables

All services are configured with:
```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
OTEL_SERVICE_NAME=<service-name>
```

## Standardized Observability

All services implement:

1. **Structured JSON Logging** - Consistent log format across all services
2. **Prometheus Metrics** - Standard HTTP metrics (requests, duration)
3. **OpenTelemetry Tracing** - Distributed tracing via OTLP

See [OBSERVABILITY.md](./OBSERVABILITY.md) and [OTEL_COLLECTOR.md](./OTEL_COLLECTOR.md) for detailed documentation.

## Health Checks

All services expose `/health` endpoints that are checked every 1 second:
- http://localhost:8080/health (Go)
- http://localhost:3000/health (TypeScript)
- http://localhost:4000/health (Elixir)

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Go Service  │     │ TypeScript  │     │ Elixir      │
│   :8080    │     │   Service   │     │  Service    │
│            │     │   :3000     │     │   :4000     │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                  │                    │
       └──────────────────┼────────────────────┘
                          │
                          │ OTLP (gRPC)
                          │
                   ┌──────▼──────────┐
                   │  OTEL Collector │
                   │   :4317 (gRPC)  │
                   │   :8889 (Metrics)│
                   └──────┬──────────┘
                          │
        ┌─────────────────┼─────────────────┐
        │
   ┌────▼────┐
   │ Datadog │
   │  Cloud  │
   └─────────┘
```

## Development

### Building Services

```bash
# Build all services
docker compose build

# Build specific service
docker compose build go-service
docker compose build typescript-service
docker compose build elixir-service
```

### Viewing Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f go-service
docker compose logs -f otel-collector
```

### Testing OTLP Connection

```bash
# Check collector is receiving data
docker compose logs otel-collector

# Check collector metrics
curl http://localhost:8889/metrics
```

## Documentation

- [OBSERVABILITY.md](./OBSERVABILITY.md) - Observability standards and practices
- [OTEL_COLLECTOR.md](./OTEL_COLLECTOR.md) - OpenTelemetry Collector configuration
- [DATADOG_SETUP.md](./DATADOG_SETUP.md) - Datadog integration guide

## License

MIT
