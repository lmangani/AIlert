# AIlert commands (use via exec)

Use the **exec** tool to run the `ailert` binary. All commands take `-config <path>` (default `config.yaml`). Use the same config and snapshot dir across runs.

- **ailert run -config <path> -save-snapshot <dir>** — Stream sources, detect patterns, update store, write snapshot to `<dir>` for later change detection.
- **ailert detect-changes -config <path> -snapshot-dir <dir>** — Compare current store to last snapshot; prints new patterns, gone patterns, count deltas.
- **ailert suggest-rules -config <path> -snapshot-dir <dir>** — Print heuristic rule suggestions (alert or suppress) with hash, level, sample, reason. You (the LLM) decide which to apply.
- **ailert apply-rule suppress <hash>** — Add suppression for pattern hash; add **-create-silence** to create an Alertmanager silence if configured.
- **ailert apply-rule alert <hash>** — Send one alert to Alertmanager for that pattern.
- **ailert suppress -pattern "<line>" -reason "..." -create-silence** — Suppress by a sample log line (hash is computed); optional **-create-silence** for Alertmanager.

Optional flags: **-suppress-threshold N** for suggest-rules (default 5). Config can set **alertmanager_url**, **snapshot_dir**, **store_path**.
