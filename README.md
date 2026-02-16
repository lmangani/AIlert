# AIlert

Log-based alerting that tells you when **something important** shows up: automatic pattern discovery, new vs. known, with optional LLM and one-click suppression. See [docs/PLAN.md](docs/PLAN.md) for the full plan.

## Phase 1 (current)

- **Core library:** Normalized `Record` types, pattern engine (template extraction, hash, new/known), pattern store (in-memory + optional JSON persist), file source.
- **CLI:** `ailert -config config.yaml` streams from configured sources and prints new/known pattern lines and a summary.

### Build and run

```bash
go build -o ailert ./cmd/ailert
```

Copy and edit config:

```bash
cp config.example.yaml config.yaml
# Edit config.yaml: set sources (e.g. file path)
./ailert -config config.yaml
```

### Config example

```yaml
store_path: ".ailert/store.json"   # optional persist
# alertmanager_url: "http://localhost:9093"  # optional; emit alerts to Alertmanager
sources:
  - id: app-log
    type: file
    path: /var/log/app.log
  # - id: metrics
  #   type: prometheus
  #   url: http://localhost:9090/metrics
```

### Tests and CI

```bash
go test ./...
```

CI runs on push/PR via [.github/workflows/ci.yml](.github/workflows/ci.yml): unit tests, integration tests (file + Prometheus source), build, and Alertmanager smoke test (POST alert against real Alertmanager in Docker).

### Project layout

```
cmd/ailert/          CLI entrypoint
internal/
  alertmanager/      Alertmanager API v2 client (POST alerts, POST/GET silences)
  config/            YAML config load
  engine/            Pattern engine (process records → hash, new/known)
  integration/       Integration tests (file, Prometheus /metrics)
  pattern/           Template extraction, level detection
  source/            Source interface, file + Prometheus source
  store/             Seen patterns + suppression store
  testutil/           Test helpers (MetricsServer, WriteLogLines)
  types/             Record, Level
config.example.yaml  Example config
docs/PLAN.md         Full architecture and phased plan
```

## Testing with Alertmanager

```bash
# Start Alertmanager (minimal config: /tmp/am.yml with route.receiver and receivers)
printf 'route:\n  receiver: default\nreceivers:\n  - name: default\n' > /tmp/am.yml
docker run -d --name am -p 9093:9093 -v /tmp/am.yml:/etc/am.yml prom/alertmanager:latest --config.file=/etc/am.yml

# Run ailert with alertmanager_url in config, then check alerts
curl -s http://localhost:9093/api/v2/alerts | jq .
```

## Next (Phase 2+)

LLM module (suppress vs notify), one-click suppress → Alertmanager silence API, then PicoClaw integration and training/rule-definition SKILLs.
