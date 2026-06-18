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

## Step-2 bridge (gateway)

For gateway architecture testing where environments are split, use:

- `hack/devtools/otlp-fake-kusto-bridge`

Flow:

1. Collect OTLP requests with this fake receiver.
2. Manually move `received-logs.ndjson` to gateway.
3. Import it into the bridge fake.
4. The bridge can forward directly to Kusto from gateway when `KUSTO_INGEST_URL` is set.
5. If forwarding is disabled, use generated `kusto-ready-logs.ndjson` for manual gateway-side ingestion.
