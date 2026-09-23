# TracePilot

A personal project to build an agentic telemetry investigator that replaces passive dashboards with an active, root-cause analysis engine.

## Goal
Detect, localize, and explain microservice incidents using traces, logs, and metrics. The final system should answer:
- What failed?
- Where did it fail?
- Why did it fail?
- Which service or dependency caused the incident?

## Core idea
A Go demo microservice emits telemetry. An OpenTelemetry Collector ingests it and stores it in ClickHouse. A LangGraph agent runs read-only SQL queries against the telemetry to investigate the incident and generate a root-cause report.

## Recommended architecture
- App service: Go + Gin + Uber Fx
- Traces: OpenTelemetry Go + otelgin middleware
- Metrics: Prometheus `/metrics` endpoint + OTel Collector scraping
- Logs: structured slog JSON with trace/span IDs
- Ingestion: OpenTelemetry Collector
- Storage: ClickHouse
- Analysis: LangGraph agent
- Optional visualization: Grafana later

## Why this hybrid approach
For the MVP, the best practical route is:
- Use OTel for distributed tracing and correlation
- Keep Prometheus metrics for simplicity and compatibility
- Use slog for logs, with trace_id and span_id attached
- Normalize everything through the OTel Collector into ClickHouse

This is fast to implement, realistic, and still aligned with OTel standards.

## Data flow
App -> OTel Collector -> ClickHouse -> LangGraph agent

Optional:
Grafana -> ClickHouse
OpenInference -> ClickHouse for agent self-observability

## MVP plan
1. Build a simple Go app
   - HTTP endpoints
   - healthz, readyz
   - metrics endpoint
   - synthetic failure scenarios
2. Add distributed tracing
   - Gin middleware
   - child spans for payment/checkouts/inventory flows
3. Add metrics and logs
   - Prometheus counters/histograms
   - slog JSON logs with trace_id/span_id
4. Setup OTel Collector
   - OTLP receiver
   - Prometheus receiver
   - filelog or stdout log ingestion
5. Store in ClickHouse
   - raw OTel tables
   - agent-facing views
6. Build the LangGraph agent
   - read-only SQL tools
   - incident investigation workflow
7. End-to-end validation
   - trigger a synthetic outage
   - agent finds root cause with evidence

## Repository naming
Preferred project name:
- TracePilot

Alternative names:
- IncidentPilot
- Telemetry Detective
- RootCause AI

## Final note
This project is not meant to be production-grade from day one. It is a strong portfolio project that demonstrates:
- distributed tracing
- metrics collection
- structured logging
- observability engineering
- AI-driven incident investigation