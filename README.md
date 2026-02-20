# AIlert

Traditional alerting forces you to write queries and thresholds by hand. You either miss real issues or drown in noise. AIlert takes a different approach: it discovers patterns in your **log stream** or **Prometheus metrics**, tracks what it has seen before, and surfaces **new** patterns so you notice when something actually changes. No query to write—just point it at a source and run.

AIlert has two parallel modules sharing the same engine and Alertmanager integration:

| Module | Sources | What it detects |
|--------|---------|-----------------|
| **LOGS** | `file`, `http` | New log patterns (ERROR, WARN, INFO, DEBUG) |
| **METRICS** | `prometheus` / `metrics` | New or changed metric series from a `/metrics` endpoint |

Both modules emit alerts to Alertmanager using identical label schemas and routing rules. You can run either standalone (no LLM, no agent). Optional: [PicoClaw](https://github.com/sipeed/picoclaw) for LLM-driven "suppress vs notify" and [DuckDB](https://duckdb.org/) for storing state and querying it back as a datasource. See [docs/PLAN.md](docs/PLAN.md) for the full picture.

---

## Build

```bash
go build -o ailert ./cmd/ailert
```

---

## LOGS module

Point AIlert at a log file (or an HTTP log stream). It turns each line into a pattern (template + hash), deduplicates by that hash, and labels each occurrence as **new** (first time) or **known**.

### How it works

Each log line is normalized into a **template**: variable parts like numbers, IPs, UUIDs, and quoted strings are stripped, then the remainder is hashed. The engine keeps a store of `(level, hash)` with a sample and count. If the hash is in the store the line is **known**; otherwise **new**. Levels (ERROR, WARN, INFO, DEBUG) are inferred from the message content automatically.

### Quickstart — log file

```bash
# Generate sample log entries
echo 'ERROR connection refused to 10.0.0.1:5432' >> /tmp/app.log
echo 'WARN timeout after 30s'                     >> /tmp/app.log
echo 'ERROR connection refused to 10.0.0.2:5432' >> /tmp/app.log
```

`config.yaml`:

```yaml
store_path: ".ailert/store.json"
sources:
  - id: app-logs
    type: file
    path: /tmp/app.log
```

```bash
./ailert run -config config.yaml
```

Output:

```
[ERROR] new  f03a7e9d (count=1) ERROR connection refused to 10.0.0.1:5432
[WARN]  new  1f214d10 (count=1) WARN timeout after 30s
[ERROR] known f03a7e9d (count=2) ERROR connection refused to 10.0.0.2:5432
```

The two ERROR lines share the same pattern (only the IP differs), so the second is **known**. The WARN is **new**. On the next run, all three are known unless a new pattern appears.

### Quickstart — HTTP log stream

```yaml
store_path: ".ailert/store.json"
sources:
  - id: remote-logs
    type: http
    url: https://example.com/logs.txt
```

AIlert fetches the URL once, emits one Record per non-empty line, and processes it through the same pattern engine.

---

## METRICS module

Point AIlert at a Prometheus `/metrics` endpoint. It treats every non-comment metric line as a Record, extracts a pattern from the metric name (labels and values stripped), and tracks which metric series are **new** or **known**.

### How it works

Each metric line such as `http_requests_total{code="500",method="get"} 42` is normalized by the same pattern engine: label values (inside `{}`) and the numeric sample value are stripped, leaving a template like `http_requests_total`. The level is inferred from keywords in the metric name — `error`, `fail`, `fatal` map to ERROR; `warn` maps to WARN; everything else to UNKNOWN. This means a **new error-class metric appearing in your endpoint** surfaces as a new ERROR pattern and fires an alert.

### Quickstart — Prometheus metrics

```yaml
store_path: ".ailert/store.json"
sources:
  - id: app-metrics
    type: prometheus
    url: http://localhost:9090/metrics
```

```bash
./ailert run -config config.yaml
```

Output (first run, all metrics are new):

```
[UNKNOWN] new  3a1c8f2b (count=1) go_goroutines 42
[ERROR]   new  d4e9a017 (count=1) http_errors_total{handler="/api"} 3
[UNKNOWN] new  8b2d1e44 (count=1) process_cpu_seconds_total 1.5
```

On subsequent runs, only genuinely new metric names surface. Existing series that disappear show up in `detect-changes` as **gone** patterns.

You can use `type: metrics` as an alias for `type: prometheus`.

### Mixing LOGS and METRICS sources

You can combine both modules in a single config:

```yaml
store_path: ".ailert/store.json"
alertmanager_url: "http://localhost:9093"
sources:
  - id: app-logs
    type: file
    path: /var/log/app.log
  - id: app-metrics
    type: prometheus
    url: http://localhost:9090/metrics
  - id: sidecar-logs
    type: http
    url: http://sidecar:8080/logs
```

All sources share the same store and Alertmanager connection, so alert routing rules in Alertmanager apply equally to log patterns and metric patterns via the `source` label.

---

## Typical workflow

Run once and save a snapshot so you can compare later (works for both LOGS and METRICS):

```bash
./ailert run -config config.yaml -save-snapshot .ailert/snapshots
```

See what changed (new patterns, gone patterns, count deltas):

```bash
./ailert detect-changes -config config.yaml -snapshot-dir .ailert/snapshots
```

Get heuristic suggestions (e.g. new ERROR -> alert, new INFO with high count -> consider suppress):

```bash
./ailert suggest-rules -config config.yaml -snapshot-dir .ailert/snapshots
```

Suppress a noisy pattern and, if you use Alertmanager, create a silence in one go:

```bash
# suppress by sample log line
./ailert suppress -config config.yaml -pattern "WARN timeout after 30s" -reason "expected" -create-silence

# suppress by hash (works for both log and metric patterns)
./ailert suppress -config config.yaml -hash d4e9a017 -reason "known flap" -create-silence
```

Other commands: `apply-rule suppress <hash>` / `apply-rule alert <hash>`, and `-metrics-addr :9090` on `run` to expose Prometheus metrics about AIlert itself.

---

## Alertmanager

Set `alertmanager_url` in config and AIlert will POST new (non-suppressed) patterns as alerts to Alertmanager. Grafana Alerting uses the same API. The same Alertmanager rules apply to both log and metric alerts. With `-create-silence`, suppressions become silences visible in the AM/Grafana UI.

Every alert carries these labels:

| Label | Value |
|-------|-------|
| `alertname` | `ailert` |
| `pattern_hash` | stable hash for deduplication |
| `level` | ERROR / WARN / INFO / DEBUG / UNKNOWN |
| `source` | source ID from config |

Quick local check:

```bash
printf 'route:\n  receiver: default\nreceivers:\n  - name: default\n' > /tmp/am.yml
docker run -d --name am -p 9093:9093 \
  -v /tmp/am.yml:/etc/am.yml \
  prom/alertmanager:latest --config.file=/etc/am.yml

./ailert run -config config.yaml
curl -s http://localhost:9093/api/v2/alerts | jq .
```

Example Alertmanager route that handles LOGS and METRICS differently:

```yaml
route:
  receiver: default
  routes:
    - match:
        level: ERROR
        source: app-logs
      receiver: pagerduty
    - match:
        level: ERROR
        source: app-metrics
      receiver: slack-metrics
receivers:
  - name: pagerduty
    # ...
  - name: slack-metrics
    # ...
  - name: default
    # ...
```

---

## DuckDB (optional)

Two separate uses, same components:

- **Storage** — Set `duckdb_path` in config. Patterns, suppressions, an append-only `records` table, and snapshots live in one DuckDB file. `run` writes records and a snapshot; `detect-changes` and `suggest-rules` use the latest snapshot from the DB (no `snapshot_dir` needed).
- **Datasource** — Add a source `type: duckdb` with an optional `query` to stream rows (e.g. from `records`) into the pipeline. Often the same DB as `duckdb_path`, but storage and datasource are independent.

```yaml
duckdb_path: ".ailert/ailert.duckdb"
alertmanager_url: "http://localhost:9093"
sources:
  - id: app-logs
    type: file
    path: /var/log/app.log
  - id: app-metrics
    type: prometheus
    url: http://localhost:9090/metrics
  - id: history
    type: duckdb
    query: "SELECT timestamp, level, message, labels, source_id FROM records WHERE timestamp > now() - interval '1 day'"
```

Requires CGO. See `config.example.yaml` for a full example.

---

## PicoClaw (optional)

For LLM-based "should we suppress or notify?" and periodic review, you can wire [PicoClaw](https://github.com/sipeed/picoclaw) to run the `ailert` CLI via its **exec** tool. No code changes in PicoClaw -- just workspace files (skill, TOOLS/HEARTBEAT snippets). [docs/picoclaw/README.md](docs/picoclaw/README.md) has the setup. Everything above works without it.

---

## Reference

**Commands:** `run`, `suppress`, `detect-changes`, `suggest-rules`, `apply-rule`. Run `./ailert` with no args for the list; `./ailert run -h` (and same for others) for flags.

**Config fields:**

| Field | Description |
|-------|-------------|
| `store_path` | JSON file for pattern store (ignored when `duckdb_path` is set) |
| `duckdb_path` | DuckDB file for store, records, and snapshots |
| `alertmanager_url` | Alertmanager or Grafana Alerting endpoint |
| `snapshot_dir` | Directory for file snapshots (only when not using DuckDB) |
| `sources[].type` | `file`, `http`, `prometheus` (alias: `metrics`), `duckdb` |
| `sources[].path` | File path (type=file) |
| `sources[].url` | HTTP or Prometheus URL (type=http, prometheus) |
| `sources[].query` | SQL query (type=duckdb) |

Full example: [config.example.yaml](config.example.yaml).

**Tests:** `go test ./...`. CI runs tests, DuckDB unit/integration and E2E, and Alertmanager integration.

**Layout:** `cmd/ailert` (CLI), `internal/` -- `engine`, `pattern`, `store`, `snapshot`, `changes`, `source` (file, http, prometheus, duckdb), `alertmanager`, `duckdb`, `config`, `metrics`, `types`. See [docs/PLAN.md](docs/PLAN.md) for architecture.
