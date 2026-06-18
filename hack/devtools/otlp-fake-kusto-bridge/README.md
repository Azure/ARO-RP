# OTLP Fake Kusto Bridge

Step-2 dev tool for gateway testing. It imports NDJSON produced by `hack/devtools/otlp-fake-receiver`, decodes OTLP log payloads, and writes Kusto-ready NDJSON rows.

## Behavior

- Listens on `LISTEN_ADDR` (default `:8081`)
- `GET /healthz/ready` returns `200 ok`
- `POST /import/received-logs` accepts NDJSON from the first fake receiver
- Decodes OTLP payloads (protobuf or JSON) and writes one row per log record
- Optionally forwards those rows directly to Kusto over HTTP from gateway

## Storage

- Default output file: `/tmp/otlp-fake-kusto-bridge/kusto-ready-logs.ndjson`
- Override with `KUSTO_READY_LOGS_PATH`

Each output row includes:

- `receivedAt`, `clusterResourceID`
- `resourceAttributes`, `scope*`, `logAttributes`
- `body`, `severity*`, `timestamp`, `observedTimestamp`
- `traceID`, `spanID`

## Environment variables

- `LISTEN_ADDR` (default `:8081`)
- `KUSTO_READY_LOGS_PATH` (default `/tmp/otlp-fake-kusto-bridge/kusto-ready-logs.ndjson`)
- `KUSTO_INGEST_URL` (optional; if set, bridge will POST NDJSON to this URL)
- `KUSTO_AUTH_HEADER` (optional; value used as `Authorization` header when forwarding)

## Manual step-2 flow (network-constrained environments)

1. Collect OTLP data in the cluster-connected environment with `otlp-fake-receiver`.
2. Manually copy `received-logs.ndjson` to gateway (this is the manual step).
3. Start this bridge fake in gateway.
4. Import logs:

```bash
curl -sS -X POST --data-binary @received-logs.ndjson http://127.0.0.1:8081/import/received-logs
```

5. If `KUSTO_INGEST_URL` is configured, the bridge forwards to Kusto directly from gateway.
6. If `KUSTO_INGEST_URL` is not configured, use `kusto-ready-logs.ndjson` for manual gateway-side ingestion.
