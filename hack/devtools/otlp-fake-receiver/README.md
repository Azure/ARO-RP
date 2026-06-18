# OTLP Fake Receiver

Small dev-only HTTP receiver for OTLP log ingestion testing.

## Behavior

- Listens on `LISTEN_ADDR` (default `:4318`)
- `GET /healthz/ready` returns `200 ok`
- `POST /v1/logs` requires `X-ARO-Cluster-Resource-ID`
- Stores accepted requests as NDJSON

## Storage

- Default file: `/tmp/otlp-fake-receiver/received-logs.ndjson`
- Override with `RECEIVED_LOGS_PATH`
- Entries older than **10 minutes** are purged on each write

Each line is JSON with:

- `receivedAt`
- `clusterResourceID`
- `contentType`
- `payloadSize`
- `payloadBase64`

## Environment variables

- `LISTEN_ADDR` (default `:4318`)
- `RECEIVED_LOGS_PATH` (default `/tmp/otlp-fake-receiver/received-logs.ndjson`)
- `DUMP_PAYLOADS` (`true` to print raw payload bytes to logs)
