# Example HEARTBEAT tasks for AIlert

Merge these into your PicoClaw workspace `HEARTBEAT.md` if you want periodic alerting checks. Adjust paths to match your workspace (e.g. config and snapshot dir).

## Quick check (run + suggest, then summarize)

- Run the AIlert pipeline and get rule suggestions, then summarize for the user:
  1. exec: `ailert run -config ./config.yaml -save-snapshot ./.ailert/snapshots`
  2. exec: `ailert suggest-rules -config ./config.yaml -snapshot-dir ./.ailert/snapshots`
  3. From the suggest-rules output, decide which patterns to suppress vs alert on; summarize "new patterns", "suggested suppressions", "suggested alerts" and optionally apply rules (apply_rule suppress/alert) or notify the user.

## What changed (no run, diff only)

- Report what changed since last run: exec `ailert detect-changes -config ./config.yaml -snapshot-dir ./.ailert/snapshots`, then summarize new/gone/count deltas for the user.

## Full training loop (run → detect → suggest → apply or notify)

- Run pipeline, detect changes, suggest rules; for each suggestion use the LLM to decide suppress vs alert; apply suppressions/alerts or send a message to the user with the summary and recommendations.
