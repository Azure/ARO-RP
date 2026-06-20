# Gateway OTEL batching performance reasoning

This note captures the mathematical reasoning behind the gateway OTEL batch tuning:

- `timeout: 30s`
- `send_batch_size: 4096`
- `send_batch_max_size: 8192`

## Why this is performant with thousands of sources

Let:

- \(R\) = total incoming log records per second across all clusters
- \(B\) = batch size trigger (`send_batch_size`)
- \(T\) = timeout trigger (`timeout`)

The batch flush rate is approximated by:

\[
\lambda_{\text{flush}}=\max\left(\frac{R}{B},\frac{1}{T}\right)
\]

Compared with old settings (\(B=1024, T=10s\)):

- Increasing \(B\) to 4096 can reduce size-triggered flushes by up to \(4\times\)
- Increasing \(T\) to 30s can reduce timeout-triggered flushes by up to \(3\times\)

A simple CPU/work model:

\[
\text{CPU/sec} = aR + c\lambda_{\text{flush}}
\]

- \(aR\): unavoidable per-record processing
- \(c\lambda_{\text{flush}}\): per-batch fixed overhead (serialization boundaries, RPC setup, TLS framing, headers, queue bookkeeping)

Larger batches reduce \(\lambda_{\text{flush}}\), so the fixed-overhead term falls nearly linearly.

Queueing view with per-batch service time \(S(B)=s_0+s_1B\):

\[
\rho=\lambda_{\text{flush}}S(B)\approx \frac{R}{B}(s_0+s_1B)=Rs_1+\frac{Rs_0}{B}
\]

The fixed-cost portion \(\frac{Rs_0}{B}\) drops as \(B\) grows; going 1024 \(\rightarrow\) 4096 reduces this term by 75%.

With many independent sources (\(R=\sum r_i\)), aggregate arrival variance relative to mean decreases (roughly \(\propto 1/\sqrt{N}\)), so size-triggered batching becomes more stable. This increases batching efficiency and aligns with Kusto guidance favoring fewer, larger ingestion batches for throughput/cost efficiency.

## How many simultaneous batches might we see?

For the current gateway logs pipeline, the batch processor keeps one aggregate batcher because `metadata_keys` is not configured.

Useful estimate:

\[
\lambda_{\text{flush}}=\max\left(\frac{R}{4096},\frac{1}{30}\right), \qquad
N_{\text{inflight}} \approx \lambda_{\text{flush}} \cdot W
\]

- \(W\): average export completion time per flushed batch (seconds)
- \(N_{\text{inflight}}\): approximate simultaneous in-flight outbound batches

Examples:

- \(R=20{,}000,\ W=0.5s \Rightarrow \lambda\approx 4.88/s,\ N_{\text{inflight}}\approx 2.4\)
- \(R=100{,}000,\ W=0.5s \Rightarrow \lambda\approx 24.4/s,\ N_{\text{inflight}}\approx 12.2\)
- If \(R<136.5\), timeout dominates: \(\lambda=1/30\), so in-flight is typically near 0-1 for normal \(W\)

So concurrency scales primarily with total incoming rate and exporter latency, not directly with source count.

## Expected memory utilization

Memory used by batching can be estimated as:

\[
M_{\text{batch}} \approx N_{\text{resident}} \cdot B_{\text{eff}} \cdot S_{\text{rec}} \cdot (1+\alpha)
\]

- \(N_{\text{resident}}\): number of resident batches in memory (active + in-flight + queue-backed unsent)
- \(B_{\text{eff}}\): average records per resident batch (bounded by `send_batch_max_size` = 8192)
- \(S_{\text{rec}}\): average in-memory bytes per log record after decode + attributes
- \(\alpha\): allocator/object overhead factor

For no metadata partitioning, a practical decomposition is:

\[
N_{\text{resident}} \approx 1 + N_{\text{inflight}} + N_{\text{queue}}
\]

- `1`: active accumulating batch
- \(N_{\text{inflight}}\): batches currently being exported
- \(N_{\text{queue}}\): additional batches buffered when exporter is slower than ingress

Under stable operation (export keeps up), \(N_{\text{queue}}\approx 0\), so:

\[
M_{\text{batch,steady}} \approx (1+N_{\text{inflight}})\cdot B_{\text{eff}}\cdot S_{\text{rec}}\cdot(1+\alpha)
\]

Since \(N_{\text{inflight}} \propto \lambda_{\text{flush}}\), reducing flush frequency (larger \(B,T\)) lowers the number of concurrent per-batch structures and associated fixed memory churn.

### Back-of-envelope examples

Assume:

- \(S_{\text{rec}}=1.5\text{ KiB}\)
- \(\alpha=0.3\)
- \(B_{\text{eff}}=4096\)

Then one resident full batch is:

\[
4096 \cdot 1.5\text{ KiB} \cdot 1.3 \approx 8.0\text{ MiB}
\]

If \(N_{\text{inflight}}=2.4\), steady-state batching memory is roughly:

\[
(1+2.4)\cdot 8.0 \approx 27\text{ MiB}
\]

If \(N_{\text{inflight}}=12.2\), it is roughly:

\[
(1+12.2)\cdot 8.0 \approx 106\text{ MiB}
\]

Actual total process memory remains higher due to receivers, auth extension state, exporter internals, and Go runtime heap fragmentation, but the formulas above show how batching-specific memory scales with rate and export latency.
