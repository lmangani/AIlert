---
name: alerting
description: Run AIlert for log-based alerting: run pipeline, detect changes, suggest rules, and decide suppress vs notify using the LLM; apply rules via the exec tool.
---

# Alerting skill (AIlert)

You have access to **AIlert** via the **exec** tool. AIlert is a log-based alerting CLI: it discovers patterns in log streams, marks them as new or known, and can suggest rules (suppress vs alert). You run it with the `exec` tool and then **use the LLM to decide** which suggestions to apply.

## Commands (run with exec)

Use the same config path and snapshot dir for a run; the user may set these in AGENTS.md or USER.md. Example: `config_path` = `./config.yaml`, `snapshot_dir` = `./.ailert/snapshots`.

1. **Run pipeline and save snapshot** (ingest logs, update pattern store, save snapshot for change detection):
   - `ailert run -config <config_path> -save-snapshot <snapshot_dir>`
   - Or if ailert is in workspace: `./ailert run -config ./config.yaml -save-snapshot ./.ailert/snapshots`

2. **Detect changes** (compare current store to last snapshot — new patterns, gone patterns, count deltas):
   - `ailert detect-changes -config <config_path> -snapshot-dir <snapshot_dir>`

3. **Suggest rules** (heuristic suggestions: alert on new ERROR/WARN, suppress on high-count INFO/DEBUG, alert on count spikes):
   - `ailert suggest-rules -config <config_path> -snapshot-dir <snapshot_dir>`
   - Optional: `-suppress-threshold N` (default 5) for when to suggest suppress for INFO/DEBUG.

4. **Apply a suppression** (add to store and optionally create Alertmanager silence):
   - `ailert apply-rule suppress <hash> -create-silence` (if Alertmanager is configured)
   - Or: `ailert suppress -hash <hash> -reason "..." -create-silence`

5. **Apply an alert** (send one alert to Alertmanager for that pattern):
   - `ailert apply-rule alert <hash>`

6. **Suppress by pattern sample** (when you don't have the hash):
   - `ailert suppress -pattern "<log line sample>" -reason "..." -create-silence`

## Your role (LLM)

- **Interpret suggest-rules output**: For each suggested rule (action=alert or suppress, hash, level, sample, reason), decide whether to **apply** it:
  - **Suppress**: Use when the pattern is expected, noisy, or not actionable (e.g. known SSH messages, health-check logs).
  - **Alert**: Use when the pattern is important or unknown and the user should be notified.
- **Summarize**: When the user asks "what changed?" or "summarize new patterns", run `ailert detect-changes` and optionally `ailert suggest-rules`, then summarize in natural language.
- **On request**: If the user says "suggest rules for the last run" or "should we suppress these?", run suggest-rules and either list your recommendations or apply rules (if the user asked to apply).
- **Periodic (HEARTBEAT)**: If HEARTBEAT includes alerting tasks, run the pipeline, then suggest-rules; summarize new/gone and your suppress vs alert recommendations, and optionally apply (e.g. if policy says auto-suppress with high confidence).

## Constraints

- Always pass the same `-config` and `-snapshot-dir` (or config's snapshot_dir) so runs are comparable.
- When applying rules, prefer `apply-rule suppress <hash> -create-silence` when Alertmanager is configured so silences appear in Grafana/AM UI.
- Do not run destructive or arbitrary shell commands; only run the `ailert` subcommands described above.
