# TracePilot

TracePilot is a personal project for building an agentic telemetry investigator. It aims to turn traces, logs, and metrics into clear answers about what failed, where it failed, and why.

## Draft Plan

- Build a small Go microservice that emits traces, metrics, and structured logs.
- Collect and correlate telemetry with OpenTelemetry.
- Store telemetry in ClickHouse for investigation.
- Build a LangGraph agent that runs read-only queries and produces root-cause reports.
- Add synthetic failure scenarios for end-to-end testing.

## Proposed Stack

Go, Gin, Uber Fx, OpenTelemetry, Prometheus, slog, OpenTelemetry Collector, ClickHouse, and LangGraph.

## Data Flow

`App -> OpenTelemetry Collector -> ClickHouse -> LangGraph agent`

Grafana and agent self-observability may be added later.

This is an MVP and portfolio project focused on practical observability and AI-assisted incident investigation.
