# AIlert

Log-based alerting that tells you when **something important** shows up: automatic pattern discovery, new vs. known, with optional LLM and one-click suppression. See [docs/PLAN.md](docs/PLAN.md) for architecture and roadmap.

**AIlert works standalone.** All core features (run, suppress, detect-changes, suggest-rules, apply-rule, Alertmanager, metrics) work without any agent or LLM. PicoClaw is an **optional** add-on for users who want an LLM to decide suppress vs notify and to drive periodic or on-request workflows.

## Features

- **Pattern engine** — Extracts log templates (variable parts stripped), deduplicates by hash, and classifies each line as **new** (first time) or **known**. Level detection (ERROR, WARN, INFO, DEBUG) from content.
- **Data sources** — **File**: read log files line-by-line. **Prometheus**: scrape `/metrics`, each line as a record. **HTTP**: GET a URL, each line as a record. **DuckDB**: run a SQL query (e.g. against a `records` table) to stream records. Storage and datasource are separate features; they just share the same DuckDB components when both are used.
- **Pattern store** — In-memory seen patterns and counts; optional JSON persist, or **DuckDB** for patterns, suppressions, records, and snapshots in one file.
- **CLI** — Subcommands: `run`, `suppress`, `detect-changes`, `suggest-rules`, `apply-rule`.
- **Alertmanager** — Emit alerts (POST `/api/v2/alerts`). One-click suppress creates a silence (POST `/api/v2/silences`) so suppressions appear in Grafana/AM UI.
- **Change detection** — Save snapshots after a run; compare current store to last snapshot (new/gone/count changes).
- **Rule suggestions** — Heuristic suggestions from changes (e.g. new ERROR → alert; new INFO with high count → suppress). No LLM required.
- **Metrics** — Optional Prometheus metrics server (records processed, new/known/suppressed, alerts emitted).

## Build and run

```bash
go build -o ailert ./cmd/ailert
```

## Commands

- **`ailert run`** — Stream sources, detect patterns, print new/known, optionally emit to Alertmanager.
  - `-config` path to config (default `config.yaml`)
  - `-save-snapshot` directory to write `snapshot_latest.json` after run (for detect-changes)
  - `-metrics-addr` e.g. `:9090` to serve Prometheus metrics
- **`ailert suppress`** — Add suppression by hash or pattern sample; optionally create Alertmanager silence.
  - `-hash` pattern hash, or `-pattern` "sample log line" (hash computed)
  - `-reason` reason (default `one-click`)
  - `-create-silence` create silence in Alertmanager (requires `alertmanager_url` in config)
- **`ailert detect-changes`** — Compare current store to last snapshot; print new/gone/count deltas.
  - `-snapshot-dir` or set `snapshot_dir` in config
- **`ailert suggest-rules`** — From current store vs last snapshot, print heuristic rule suggestions (suppress/alert).
  - `-suppress-threshold` suggest suppress for new INFO/DEBUG when count >= N (default 5)
- **`ailert apply-rule suppress <hash>`** — Apply suppression (store + optional AM silence).
- **`ailert apply-rule alert <hash>`** — Send one alert to Alertmanager for that pattern.

## Config example

```yaml
store_path: ".ailert/store.json"
# duckdb_path: ".ailert/ailert.duckdb"   # optional: use DuckDB for store, records, snapshots
# alertmanager_url: "http://localhost:9093"
# snapshot_dir: ".ailert/snapshots"

sources:
  - id: app-log
    type: file
    path: /var/log/app.log
  # type: prometheus; url: http://localhost:9090/metrics
  # type: http; url: https://example.com/logs.txt
  # type: duckdb; query: "SELECT * FROM records ORDER BY timestamp"  # optional; uses same DB as duckdb_path if set
```

## Example workflow

```bash
# Run once, save snapshot
./ailert run -config config.yaml -save-snapshot .ailert/snapshots

# Later: see what changed
./ailert detect-changes -config config.yaml -snapshot-dir .ailert/snapshots

# Get rule suggestions (no LLM)
./ailert suggest-rules -config config.yaml -snapshot-dir .ailert/snapshots

# One-click suppress a pattern and create Alertmanager silence
./ailert suppress -config config.yaml -pattern "WARN noisy message 123" -create-silence
```

## Tests and CI

```bash
go test ./...
```

CI: unit tests, integration tests (file, Prometheus, HTTP sources; simulated datasets), build, Alertmanager integration test.

## Testing with Alertmanager locally

```bash
printf 'route:\n  receiver: default\nreceivers:\n  - name: default\n' > /tmp/am.yml
docker run -d --name am -p 9093:9093 -v /tmp/am.yml:/etc/am.yml prom/alertmanager:latest --config.file=/etc/am.yml

./ailert run -config config.yaml
curl -s http://localhost:9093/api/v2/alerts | jq .
```

## Project layout

```
cmd/ailert/          CLI (run, suppress, detect-changes, suggest-rules, apply-rule)
internal/
  alertmanager/     Alertmanager API v2 (alerts, silences)
  changes/          Change detection and heuristic rule suggestions
  config/           YAML config
  duckdb/            DuckDB backend (store, records, snapshots) and DuckDB source type; same components, independent features
  engine/           Pattern engine (hash, new/known)
  metrics/          Optional Prometheus metrics
  integration/       Pipeline tests
  pattern/          Template extraction, level detection
  snapshot/         Snapshot save/load (file or DuckDB)
  source/           File, Prometheus, HTTP, DuckDB sources
  store/            Seen patterns and suppression store (memory + JSON or DuckDB)
  testutil/         Test helpers (MetricsServer, datasets)
  types/            Record, Level
```

## DuckDB (optional)

DuckDB is used in two independent ways; they share the same components but are not tied to each other.

1. **Storage** — Set **`duckdb_path`** in config to use DuckDB for state: patterns, suppressions, **records** (append-only log of ingested lines), and snapshots. With storage enabled: **run** appends each record to the `records` table and saves a snapshot in the DB (no `-save-snapshot` dir needed); **detect-changes** and **suggest-rules** use the latest snapshot from the DB (no `snapshot_dir` needed).

2. **Datasource** — Add a source **`type: duckdb`** with an optional **`query`** to stream records from a DuckDB database (e.g. `SELECT * FROM records ORDER BY timestamp`). Often you use the same DB as `duckdb_path`, but the two features are separate.

Build requires CGO (DuckDB embeds a native library). See `config.example.yaml` for examples.

## Optional: PicoClaw integration

If you want an LLM agent to decide **suppress vs notify**, summarize changes, or run periodic checks, you can use [PicoClaw](https://github.com/sipeed/picoclaw). The agent runs the `ailert` CLI via **exec**; no PicoClaw code changes. Setup and workspace files: **[docs/picoclaw/README.md](docs/picoclaw/README.md)**. This is optional — basic features work regardless.

## Roadmap

Planned: optional native PicoClaw tool (if accepted upstream). Current: exec-based integration and alerting skill.
