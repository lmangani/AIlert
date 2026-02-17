# AIlert

Most log alerting forces you to write queries and thresholds. You either miss real issues or drown in noise. AIlert takes a different approach: it discovers patterns in your log stream, tracks what it has seen before, and surfaces **new** patterns so you notice when something actually changes. No query to write—just point it at a file, URL, or Prometheus metrics and run.

You can run it standalone (no LLM, no agent). Optional: [PicoClaw](https://github.com/sipeed/picoclaw) for LLM-driven “suppress vs notify” and [DuckDB](https://duckdb.org/) for storing state and querying it back as a datasource. See [docs/PLAN.md](docs/PLAN.md) for the full picture.

---

## What you see

Point AIlert at a log file and run. It turns each line into a pattern (template + hash), deduplicates by that hash, and labels each occurrence as **new** (first time) or **known**:

```bash
go build -o ailert ./cmd/ailert

# sample log
echo 'ERROR connection refused to 10.0.0.1:5432' >> /tmp/app.log
echo 'WARN timeout after 30s' >> /tmp/app.log
echo 'ERROR connection refused to 10.0.0.2:5432' >> /tmp/app.log
```

Config (`config.yaml`):

```yaml
store_path: ".ailert/store.json"
sources:
  - id: app
    type: file
    path: /tmp/app.log
```

```bash
./ailert run -config config.yaml
```

Example output:

```
[ERROR] new f03a7e9d... (count=1) ERROR connection refused to 10.0.0.1:5432
[WARN]  new 1f214d10... (count=1) WARN timeout after 30s
[ERROR] known f03a7e9d... (count=2) ERROR connection refused to 10.0.0.2:5432
```

The two ERROR lines share the same pattern (only the IP/port differ), so the second one is **known**. The WARN is **new**. Next run, all three would be known unless a new pattern appears.

---

## How it works

Each log line is normalized into a **template** (variable bits like numbers, UUIDs, IPs are stripped), then hashed. The engine keeps a store of (level, hash) with a sample and count. If the hash is in the store, the line is **known**; otherwise **new**. Levels (ERROR, WARN, INFO, DEBUG) are inferred from the message if not provided. You can **suppress** a pattern by hash or by a sample line so it no longer counts as alertable; optionally that suppression is mirrored as an Alertmanager silence so it shows up in Grafana.

Data can come from a **file**, an **HTTP** URL (GET, line-by-line), **Prometheus** `/metrics` (each line as a record), or a **DuckDB** query. State can live in a JSON file or in DuckDB (patterns, suppressions, an append-only `records` table, and snapshots for change detection).

---

## Typical workflow

Run once and save a snapshot so you can compare later:

```bash
./ailert run -config config.yaml -save-snapshot .ailert/snapshots
```

Later, see what changed (new patterns, gone patterns, count deltas):

```bash
./ailert detect-changes -config config.yaml -snapshot-dir .ailert/snapshots
```

Get heuristic suggestions (e.g. new ERROR → alert, new INFO with high count → consider suppress):

```bash
./ailert suggest-rules -config config.yaml -snapshot-dir .ailert/snapshots
```

Suppress a noisy pattern and, if you use Alertmanager, create a silence in one go:

```bash
./ailert suppress -config config.yaml -pattern "WARN timeout after 30s" -reason "expected" -create-silence
```

Other commands: `apply-rule suppress <hash>` / `apply-rule alert <hash>`, and `-metrics-addr :9090` on `run` to expose Prometheus metrics.

---

## Alertmanager

Set `alertmanager_url` in config and AIlert will POST new (non-suppressed) patterns as alerts to Alertmanager. Grafana Alerting uses the same API, so you don’t need a custom UI. With `-create-silence`, suppressions are turned into silences so they appear in the AM/Grafana UI.

Quick local check:

```bash
printf 'route:\n  receiver: default\nreceivers:\n  - name: default\n' > /tmp/am.yml
docker run -d --name am -p 9093:9093 -v /tmp/am.yml:/etc/am.yml prom/alertmanager:latest --config.file=/etc/am.yml
# in config.yaml set alertmanager_url: "http://localhost:9093"
./ailert run -config config.yaml
curl -s http://localhost:9093/api/v2/alerts | jq .
```

---

## DuckDB (optional)

Two separate uses, same components:

- **Storage** — Set `duckdb_path` in config. Patterns, suppressions, an append-only `records` table, and snapshots live in one DuckDB file. `run` writes records and a snapshot; `detect-changes` and `suggest-rules` use the latest snapshot from the DB (no `snapshot_dir`).
- **Datasource** — Add a source `type: duckdb` with an optional `query` to stream rows (e.g. from `records`) into the pipeline. Often the same DB as `duckdb_path`, but storage and datasource are independent.

Requires CGO. See `config.example.yaml` for a `duckdb_path` and `type: duckdb` source.

---

## PicoClaw (optional)

For LLM-based “should we suppress or notify?” and periodic review, you can wire [PicoClaw](https://github.com/sipeed/picoclaw) to run the `ailert` CLI via its **exec** tool. No code changes in PicoClaw—just workspace files (skill, TOOLS/HEARTBEAT snippets). [docs/picoclaw/README.md](docs/picoclaw/README.md) has the setup. Everything above works without it.

---

## Reference

**Commands:** `run`, `suppress`, `detect-changes`, `suggest-rules`, `apply-rule`. Run `./ailert` with no args for the list; `./ailert run -h` (and same for others) for flags.

**Config:** `store_path` (JSON) or `duckdb_path` (DuckDB), `alertmanager_url`, `snapshot_dir` (for file snapshots when not using DuckDB). Under `sources`: `type` + `path` (file), `url` (http/prometheus), or `query` (duckdb). Full example: [config.example.yaml](config.example.yaml).

**Tests:** `go test ./...`. CI runs tests, DuckDB unit/integration and E2E, and Alertmanager integration.

**Layout:** `cmd/ailert` (CLI), `internal/` — `engine`, `pattern`, `store`, `snapshot`, `changes`, `source` (file, http, prometheus, duckdb), `alertmanager`, `duckdb`, `config`, `metrics`, `types`. See [docs/PLAN.md](docs/PLAN.md) for architecture.
