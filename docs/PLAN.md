# AIlert: Log-Based Alerting Center — Plan

**Goal:** Build a system that tells you *when something important or critical shows up in the logs* — not “run a query and notify on match,” but automatic pattern discovery, new vs. known, and smart suppression (LLM + one-click).

**Agent role:** The agent doesn't only react — it **trains on evolving datasets**, **detects when they change**, and **defines (or suggests) new rules** to handle them. This runs **periodically** (HEARTBEAT/cron) and **on user request** ("review the logs and suggest new rules," "what's changed in the last week?").

**Stack:** Go, PicoClaw (inspiration or active integration), SKILLs for actions, extensible data sources (SQL, Prometheus, LogQL, etc.) with format mappings, and fast realtime pattern detection.

---

## 1. Problem vs. Traditional Alerting

| Traditional | AIlert |
|-------------|--------|
| You write queries; matches → notify | System discovers patterns and highlights what’s *important* |
| Manual tuning of thresholds and queries | Automatic pattern extraction; “new” vs “already seen” |
| Noisy alerts (e.g. SSH warnings) | LLM + one-click suppression of expected/noisy patterns |
| Tied to one data source/format | Pluggable sources + mappings (SQL, Prometheus, LogQL, etc.) |

[metrico/logparser](https://github.com/metrico/logparser) is a good reference for *pattern extraction* (template mining, level detection, clustering) but is oriented toward batch/summary (e.g. `cat log | logparser`). We need **realtime streaming**, low latency, and integration with an **agent** that can act (suppress, notify, escalate) via SKILLs.

---

## 2. High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         AIlert Alerting Center                           │
├─────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐   ┌──────────────┐   ┌─────────────────────────────┐  │
│  │ Data        │   │ Format       │   │ Pattern Engine               │  │
│  │ Sources     │──▶│ Mappings     │──▶│ (realtime, new/known, level)  │  │
│  │ (pluggable) │   │ (per source) │   │                              │  │
│  └─────────────┘   └──────────────┘   └──────────────┬──────────────┘  │
│         │                   │                         │                 │
│         │                   │                         ▼                 │
│         │                   │              ┌─────────────────────────┐  │
│         │                   │              │ Suppression & Routing   │  │
│         │                   │              │ • LLM (noisy/expected)  │  │
│         │                   │              │ • One-click → AM silence│  │
│         │                   │              │ • Emit → Alertmanager   │  │
│         │                   │              └────────────┬────────────┘  │
│         │                   │                           │               │
│         ▼                   ▼                           ▼               │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │              Agent (PicoClaw-style) + SKILLs                      │  │
│  │  • Query APIs (SQL, Prometheus, LogQL, custom)                    │  │
│  │  • Act: suppress, ack, notify, run playbooks                      │  │
│  │  • Train: learn from evolving data; detect drift; define rules    │  │
│  │  • HEARTBEAT / cron: periodic check + periodic train/rule review  │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
```

- **Data sources:** Pluggable modules (SQL, Prometheus, LogQL, file tail, etc.) producing a unified stream or pull model.
- **Format mappings:** Per-source config to normalize to a common schema (e.g. timestamp, level, message, labels) so the pattern engine and LLM see a consistent view.
- **Pattern engine:** Realtime template extraction (Drain/Spell-style or optimized logparser idea), pattern hash, level, “new” vs “seen before.”
- **Suppression & routing:** LLM to classify noisy/expected; one-click → Alertmanager silence; emit alerts to Alertmanager (Grafana Alerting uses same backend; no custom UI).
- **Agent:** PicoClaw as base or inspiration; SKILLs = tools that implement “alerting center” actions (query, suppress, and **train on data / detect change / define rules**).

---

## 3. Agent as Trainer: Evolving Data, Change Detection, Rule Definition

The agent participates in a **training loop** over evolving datasets: it learns what's normal, notices when the data changes, and defines or suggests new rules. This runs **periodically** (e.g. HEARTBEAT) and **on user request** (e.g. "review logs and suggest rules," "what changed?").

### Train on evolving datasets

- **Continuous learning:** The pattern store and (optionally) lightweight baselines (e.g. pattern counts, level mix per source) are updated as data flows in. The agent can **ingest recent windows** of data (via existing sources) to refresh its view of "what exists" and "what's normal."
- **Training triggers:** (1) **Periodic:** HEARTBEAT/cron runs "train" (e.g. pull last N hours from each source, run pattern engine, update store and baselines). (2) **On request:** User asks "train on the last 24h" or "analyze logs from production" — agent calls `query_logs` / `query_metrics`, runs pattern extraction, updates internal state.
- **No custom ML stack required:** "Training" here = updating the pattern store, optional summary stats (counts, rates), and optionally asking the LLM to summarize "what's new or surprising" for a time window. Future: optional embedding or anomaly model.

### Understand when data changes

- **Change detection:** Compare current pattern set and/or baselines to a previous snapshot (e.g. last run, or last week). Detect: **new patterns**, **disappeared patterns**, **significant count/rate shifts** (e.g. a previously rare pattern spikes), **schema/format drift** (e.g. new fields, changed timestamp format).
- **Output:** Structured "diff" (new / gone / changed) that the agent can reason over and turn into actions (alert, suggest rule, auto-suppress, etc.).

### Define new rules (periodic or on request)

- **Rule types:** (1) **Suppression:** "Treat this pattern as expected" → add to suppression list or create Alertmanager silence. (2) **Alert:** "Treat this pattern (or this change) as critical" → ensure it fires to Alertmanager or create/update an alert rule. (3) **Routing / grouping:** e.g. suggest label or contact-point changes.
- **Agent-driven:** The agent uses LLM + tools to propose rules: e.g. "Pattern X is new and looks like a known SSH warning; suggest adding a suppression." User can approve (one-click) or the agent can apply when configured (e.g. auto-suppress with LLM high confidence).
- **Periodic:** HEARTBEAT runs "review recent changes → suggest rules" and notifies the user ("3 new patterns; I suggest suppressing 2 and alerting on 1") or applies suggested rules if policy allows.
- **On request:** User: "Define rules to handle what you see in the last 24h" or "Should we suppress these?" Agent runs change detection, suggests rules, and optionally applies them (with or without confirmation).

### Summary

| Aspect | Periodic | On user request |
|--------|----------|------------------|
| **Train** | HEARTBEAT: pull recent data, update pattern store & baselines | "Train on last 24h" / "Analyze production logs" |
| **Change detection** | After each train: diff vs previous snapshot | "What changed in the last week?" / "Summarize new patterns" |
| **Define rules** | Suggest (or apply) suppressions/alert rules from changes | "Suggest rules for what you see" / "Suppress these" |

The agent thus keeps the system adapted to evolving data and reduces manual tuning.

---

## 4. PicoClaw: Integration vs. Inspiration

**Option A — Extend PicoClaw**

- Add AIlert as a **custom skill** and **tools** in `workspace/` (e.g. `skills/alerting/`, entries in `TOOLS.md`).
- New tools: `query_logs`, `query_metrics`, `suppress_pattern`, `list_new_patterns`, `ack_alert`, etc.
- HEARTBEAT.md (or cron) triggers periodic “check configured sources → pattern engine → LLM filter → notify if important.”
- Pros: Single binary, existing channels (Telegram, Discord), existing LLM/config. Cons: Tied to PicoClaw release cycle and layout.

**Option B — Standalone agent inspired by PicoClaw**

- Reuse ideas: workspace, `AGENTS.md`-style prompt, SKILLs as named tools, config (providers, tools).
- Implement only what’s needed: config, one LLM provider, “alerting” tools, optional gateway (e.g. webhook or one channel).
- Pros: Full control, minimal surface. Cons: More code to own (auth, channels if needed).

**Recommendation:** Start with **Option A** (extend PicoClaw) for speed: implement **data source adapters** and **pattern engine** as a Go library/CLI, then expose them as PicoClaw tools/skills so the agent can “run alerting” and users get one-click suppression and notifications through existing channels. If we outgrow PicoClaw, we can extract the core into a standalone agent (Option B) later.

---

## 5. Data Sources (Extensible Modules)

Each module is a **source type** + **optional format mapping**.

| Module       | Purpose           | Output shape (after mapping)     |
|-------------|-------------------|-----------------------------------|
| **SQL**     | Query DB for logs/metrics | Rows → timestamp, level, message, labels |
| **Prometheus** | PromQL / query API   | Series + samples → time, value, labels   |
| **LogQL**   | Loki query API     | Log stream → timestamp, line, labels     |
| **File**    | Tail / read file   | Lines → timestamp, level, message        |
| **HTTP**    | Generic REST/JSON | Configurable mapping to common schema    |

- **Interface (Go):** e.g. `Source` with `Stream(ctx) (<-chan Record, error)` or `Poll(ctx) ([]Record, error)`.
- **Format mapping:** YAML/JSON per source: field names, timestamp layout, level detection (regex or keyword), “message” field. Same idea as logparser decoders: normalize so the rest of the pipeline is source-agnostic.
- **Config:** List of sources + mapping names; credentials via env or config (no secrets in repo).

---

## 6. Pattern Engine (Fast, Realtime)

Requirements:

- **Realtime:** Process stream or frequent polls with low latency (no “run at end of file” only).
- **Template extraction:** Turn a log line into a pattern (e.g. replace numbers/hex/UUIDs with placeholders, then hash). logparser’s approach (word tokenization, digit/hex removal, `WeakEqual`) is a good model but may need tuning for speed (e.g. avoid full scan of all patterns on every message).
- **New vs. known:** Maintain a store (in-memory + optional persisted) of seen pattern hashes (and optionally level). First time we see a pattern → “new”; subsequent → “known.” New patterns can be routed to LLM or directly to “notify.”
- **Level:** Reuse or mirror logparser’s level detection (ERROR, WARN, INFO, etc.) so we can focus alerts on ERROR/WARN and optionally treat INFO as lower priority.

Possible implementations:

1. **Optimized logparser fork:** Keep its `Pattern`/`Parser` idea; optimize hot path (e.g. lock-free or sharded map, avoid full `WeakEqual` scan by better indexing).
2. **Drain-style tree (e.g. Drain3):** Template tree by token length and tokens; good for streaming; may need a Go port or bindings.
3. **Spell-style (e.g. [pfeak/spell](https://github.com/pfeak/spell)):** Online LCS-based template extraction in Go; compare throughput vs. our target (e.g. 10k lines/s).

Deliverables:

- A **pattern package** that: `Add(line, level, ts)` → pattern hash, “new”/“known”, and optional sample.
- **Store:** In-memory + optional SQLite/file for “seen patterns” and “suppression list” so restarts don’t re-alert on known patterns.

---

## 7. Suppression & Routing

- **LLM suppression:** For patterns that pass the pattern engine (e.g. new or high count), send a short context (pattern, sample, source) to the LLM with a prompt like: “Is this likely expected/noisy (e.g. common SSH warning, health check)?” If yes → suppress (and optionally add to one-click rules).
- **One-click suppression:** User says “don’t alert on this again” (e.g. from Telegram/Discord or a small UI). Action: add pattern hash (and optionally level/source) to suppression list; persist.
- **Routing:** After suppression: route remaining events to “notify” (channel, webhook, PagerDuty, etc.) or “escalate to agent” (agent decides next step via SKILLs).

---

## 7b. Alertmanager & Grafana Alerting Compatibility (No Custom UI)

To integrate with existing stacks and **avoid building a UI**, AIlert will emit and manage alerts via **Prometheus Alertmanager** and **Grafana Alerting**, so users use Grafana and Alertmanager UIs for visualization, silence management, and routing.

### Outbound: AIlert → Alertmanager

- **Emit in Alertmanager format:** When the pipeline decides “alert” (new/important pattern, not suppressed), produce an alert in the same shape Alertmanager expects: `labels` (e.g. `alertname`, `pattern_hash`, `level`, `source`), `annotations` (e.g. `summary`, `description`, sample log line), `startsAt`, optional `endsAt`, optional `generatorURL`.
- **POST to Alertmanager API:** Push alerts via `POST /api/v2/alerts` (or equivalent). Alertmanager then handles grouping, deduplication, routing to contact points (Slack, PagerDuty, email, etc.), and silence matching. Clients are expected to re-send firing alerts periodically (e.g. every 30s–3m) until resolved.
- **Resolve:** When a pattern is no longer firing (e.g. no new occurrences in a window), send the same alert with `endsAt` set or stop sending it so Alertmanager can mark it resolved (per its semantics).

Result: **No AIlert UI.** Users see and manage alerts in Alertmanager’s UI (or in Grafana when Grafana uses Alertmanager as the alert backend).

### Grafana Alerting

- **Primary path:** Grafana Alerting typically uses **Alertmanager as the state store**. By pushing alerts to Alertmanager, they appear in **Grafana’s Alerting section** (alerts list, history) and respect Grafana’s contact points if configured to use the same Alertmanager. No extra integration needed beyond Alertmanager.
- **Optional:** If you use **Grafana OnCall** or **Grafana IRM** with **incoming webhooks**, AIlert could additionally (or alternatively) POST to Grafana’s incoming webhook URL to create incidents; useful if you want incidents in Grafana IRM without going through Alertmanager.

### One-click suppression → Alertmanager silences

- **Implement “suppress this” as a silence:** When the user says “don’t alert on this again” (chat or API), AIlert can call **Alertmanager’s Silences API** (`POST /api/v2/silences`) to create a silence matching that alert (e.g. by `pattern_hash` and/or other labels). Then Alertmanager will suppress future notifications for that pattern; users manage and expire silences in Alertmanager (or Grafana) as usual.
- **LLM suppression:** Can remain internal to AIlert (so we don’t even send noisy alerts to Alertmanager), or optionally create a short-lived Alertmanager silence so the same pattern is suppressed in one place.

### Config

- **Alertmanager URL** (and optional auth) in AIlert config.
- **Alert label schema:** Convention for `alertname`, `pattern_hash`, `level`, `source`, etc., so silences and routing rules in Alertmanager/Grafana work predictably.

### Summary

| Integration        | Role |
|--------------------|------|
| **Alertmanager**   | Receive alerts from AIlert via API; handle routing, grouping, silencing, contact points; single place for “who gets notified.” |
| **Grafana Alerting** | Use existing Grafana + Alertmanager setup; view alerts and (optionally) manage contact points in Grafana; no AIlert UI. |
| **One-click suppress** | Implemented as “create Alertmanager silence” via API so all suppression lives in Alertmanager/Grafana. |

This keeps compatibility with Alertmanager and Grafana Alerting and avoids building a custom UI.

---

## 8. SKILLs as Alerting Actions

SKILLs = named tools the agent can call. Suggested set:

| Skill / Tool       | Purpose |
|--------------------|--------|
| `query_logs`       | Run a predefined or parameterized log query (LogQL, SQL, file) and return recent lines or summary. |
| `query_metrics`    | Run Prometheus/LogQL metric query; return series or scalar. |
| `list_new_patterns`| Return new patterns since last run (or since N hours). |
| `list_active_alerts`| Return current unsuppressed alerts / patterns above threshold. |
| `suppress_pattern` | Add pattern to suppression list; optionally create Alertmanager silence via API (one-click from chat; visible in Grafana/AM UI). |
| `ack_alert`        | Acknowledge an alert (record who/when; optional TTL). |
| `run_playbook`     | Trigger a script or HTTP hook (e.g. restart service, run playbook). |
| **Training & rules** | |
| `train_on_window`  | Ingest data from sources for a time window (e.g. last 24h); run pattern engine; update pattern store and baselines. |
| `detect_changes`   | Compare current pattern set/baselines to a previous snapshot; return structured diff (new / gone / changed patterns, rate shifts). |
| `suggest_rules`    | Given a change diff (or on request), use LLM to suggest suppressions or alert rules; return list for user approval or auto-apply. |
| `apply_rule`       | Apply a suggested rule: create suppression (or AM silence) or ensure alert fires; optionally call Alertmanager/Grafana APIs. |

Implementation: In PicoClaw, these become tools described in `TOOLS.md` and implemented in the same way as existing tools (e.g. in `pkg/`), calling into the AIlert library (sources + pattern engine + suppression store + Alertmanager client).

---

## 9. Phased Plan

### Phase 1 — Core library (no PicoClaw yet)

- **1.1** Define common types: `Record` (timestamp, level, message, labels, source_id).
- **1.2** Implement 1–2 data sources with format mappings (e.g. **File tail** and **LogQL** or **Prometheus**).
- **1.3** Implement **pattern engine** (optimized logparser-style or Spell/Drain): `Add` → hash, new/known, level.
- **1.4** **Pattern store:** in-memory + optional file/SQLite (seen patterns, suppression list).
- **1.5** CLI: e.g. `ailert run --config config.yaml` that streams from configured sources, runs pattern engine, prints “new”/“known” and counts (no LLM yet).
- **1.6** **CI (Phase 1):** GitHub Actions: `go test ./...`; optional job that runs real Alertmanager in a container and a minimal test (e.g. POST one alert via our client).

**Exit criteria:** Single binary that reads from file or one API, outputs pattern hashes and new/known per line (or per batch); CI green with unit tests and optional Alertmanager smoke test.

### Phase 2 — Suppression and LLM

- **2.1** **Suppression store:** load/save; API: `Suppress(hash, reason)`, `IsSuppressed(hash)`.
- **2.2** **LLM module:** Given (pattern, sample, source), call configured LLM (OpenAI/OpenRouter/Anthropic) with fixed prompt; return “suppress” or “notify.”
- **2.3** Wire pipeline: pattern engine → suppression check → if not suppressed, LLM check → if still not suppressed, emit “alert” (e.g. to stdout; then to Alertmanager in Phase 2b).
- **2.4** One-click: CLI or HTTP endpoint `POST /suppress` with pattern hash or sample text → add to store; optionally create Alertmanager silence via API.
- **2.5** **Alertmanager sink:** Format alerts as Alertmanager payload; POST to `Alertmanager /api/v2/alerts`; optional create silence via `/api/v2/silences` for one-click suppress.
- **2.6** **CI (Phase 2):** Add simulated Prometheus `/metrics` endpoint (test fixture server); data generation scripts for synthetic logs; integration tests: scrape metrics, ingest logs, POST alerts/silences to real Alertmanager in CI; assert payloads and behaviour (see §10).

**Exit criteria:** Configurable LLM; one-click and LLM-based suppression working; alerts emitted only for “important” patterns; CI validates Alertmanager API and ingestion with generated data.

### Phase 3 — PicoClaw integration (SKILLs + tools)

- **3.1** Add AIlert as dependency or subpackage; config in PicoClaw’s `config.json` (e.g. `ailert.sources`, `ailert.suppression_file`).
- **3.2** Implement tools: `query_logs`, `query_metrics`, `list_new_patterns`, `list_active_alerts`, `suppress_pattern`, `ack_alert` (and optionally `run_playbook`).
- **3.3** Update `TOOLS.md` (and skill docs) so the agent knows when to use them.
- **3.4** HEARTBEAT or cron (alerts): every N minutes, run pattern check on configured sources; if new/important patterns, use `message` tool to notify user (or send to channel).
- **3.5** **Training loop:** HEARTBEAT (or separate schedule): `train_on_window` (e.g. last 6h) → `detect_changes` vs previous snapshot → `suggest_rules` → notify user or auto-apply per policy.
- **3.6** On-request training tools: `train_on_window`, `detect_changes`, `suggest_rules`, `apply_rule` when user asks e.g. "what changed?" or "suggest rules for the last 24h."

**Exit criteria:** User can talk to PicoClaw (“What’s new in the logs?”, “Suppress this pattern”) and get periodic alert summaries plus periodic (or on-demand) training and rule suggestions.

### Phase 4 — Alerting center polish

- **4.1** More sources: SQL, generic HTTP, more LogQL options.
- **4.2** Notifications: rely on Alertmanager contact points (Slack, PagerDuty, email); optional Grafana OnCall incoming webhook.
- **4.3** No custom UI: use Alertmanager and Grafana Alerting UIs for alerts, silences, and history; agent/chat for “suppress this pattern” (creates AM silence).
- **4.4** **Training & rule definition:** Persist baselines/snapshots for change detection; HEARTBEAT runs train → detect → suggest (and optionally apply) periodically; on-request "what changed?" / "suggest rules" via chat.
- **4.5** Observability: metrics (e.g. patterns/sec, suppressed count, LLM latency) for the pipeline itself; expose as Prometheus metrics if desired.
- **4.6** **CI & testing:** Extensive CI via GitHub Actions: real Alertmanager, data generation scripts, simulated Prometheus `/metrics` endpoint; validate APIs, pattern detection, ingestion (see §10).

---

## 10. CI & Testing (GitHub Actions)

We use **extensive CI testing** in **GitHub Actions** to validate APIs, pattern detection, ingestion, and end-to-end behaviour against **real Alertmanager** and **synthetic data**, with a **simulated Prometheus `/metrics`** endpoint.

### Goals

- **Validate our APIs** against the real Alertmanager (POST alerts, POST/GET silences, correct payload shape and status codes).
- **Validate logic:** pattern detection (new vs known, hashing, level detection), suppression (store + LLM path), ingestion from multiple source types.
- **Reproducible:** Data-driven tests with generated logs/metrics so behaviour is deterministic and regression-safe.

### Components

| Component | Purpose |
|-----------|---------|
| **Real Alertmanager in CI** | Run Alertmanager in a container (e.g. official image) in the Actions job; point AIlert at `http://localhost:9093` (or assigned port). Test that we can POST alerts, create/expire silences, and that Alertmanager accepts our payloads and returns expected responses. |
| **Simulated Prometheus `/metrics` endpoint** | A small HTTP server (or pre-recorded responses) that serves Prometheus text exposition format at `/metrics`. Used to test: (1) our Prometheus/metrics **source** (scrape, parse), (2) format mappings, (3) ingestion into the pattern pipeline when metrics are treated as log-like or used for thresholds. Can be implemented in Go in `test/fixtures` or as a static file + minimal server. |
| **Data generation scripts** | Scripts (Go or shell + templates) that generate: (1) **synthetic log streams** (known patterns, new patterns, mixed levels, malformed lines), (2) **time-windowed datasets** for train/detect_changes tests. Output: files or stdin streams that we feed into the pattern engine and ingestion pipeline. Assert on expected pattern hashes, new/known counts, and suppression outcomes. |

### What we validate

- **Alertmanager API client:** POST `/api/v2/alerts` (payload shape, 200); POST `/api/v2/silences` (create, 200); GET silences/alerts; error handling (4xx/5xx).
- **Pattern detection:** Given a fixed log corpus, assert pattern hashes and new/known classification; add tests for level detection, template extraction, and boundary cases (empty lines, very long lines, Unicode).
- **Ingestion:** End-to-end from source to pattern store: file tail, simulated Prometheus scrape, (optionally) LogQL mock. Assert records normalized, patterns extracted, and (where applicable) alerts emitted or suppressed.
- **Suppression:** With generated data that matches suppression rules (or LLM mock), assert no alert is sent to Alertmanager; with one-click suppress, assert silence created in Alertmanager and subsequent duplicate pattern does not alert.
- **Training / change detection:** Data generation scripts produce “before” and “after” snapshots; run `train_on_window` and `detect_changes`; assert diff (new/gone/changed) matches expected.

### GitHub Actions layout (outline)

- **Workflow:** On push/PR, run unit tests (`go test ./...`), then integration job(s).
- **Integration job:** Start Alertmanager container (e.g. `prom/alertmanager:latest` with minimal config); start simulated Prometheus `/metrics` server (or serve static fixture); run data generation script(s) to produce test logs; run AIlert binary or tests that call the library with config pointing at `localhost` Alertmanager and metrics URL; assert via Alertmanager API (GET alerts/silences) and/or stdout/file output that alerts and silences match expectations.
- **Secrets:** No Alertmanager auth required for CI; optional API keys for LLM tests (e.g. “LLM suppression” tests) can be skipped in CI or use a mock.

### Phasing

- **Phase 1:** Unit tests for pattern engine and store; optional: start Alertmanager in CI and test a minimal “POST one alert” Go test.
- **Phase 2:** Add simulated `/metrics` endpoint and data generation script(s); integration test: scrape metrics → normalize → optional pattern/ingestion path; Alertmanager tests for POST alerts and silences.
- **Phase 3+:** Full pipeline tests (generated logs → pattern → suppress/alert → Alertmanager); train/detect_changes tests with before/after datasets; LLM tests mocked or gated.

---

## 11. Tech Summary

| Component        | Choice / direction |
|------------------|--------------------|
| Language         | Go                 |
| Agent base       | PicoClaw (extend) or standalone inspired by it |
| Data sources     | Pluggable: File, LogQL, Prometheus, SQL, HTTP |
| Format mappings  | Per-source YAML/JSON → common Record schema |
| Pattern engine   | Realtime; logparser-style or Spell/Drain; “new” vs “known” |
| Pattern store    | In-memory + file or SQLite (seen + suppressed) |
| Suppression      | LLM + one-click; persisted |
| Actions          | SKILLs/tools: query, list, suppress, ack, playbook; **train_on_window**, **detect_changes**, **suggest_rules**, **apply_rule** |
| Agent training   | Train on evolving data (periodic + on request); detect change (new/gone/changed patterns); define or suggest rules (suppress/alert) |
| Notifications / UI | **Alertmanager** (receive alerts, silences, routing); **Grafana Alerting** (view alerts, contact points); no custom AIlert UI |
| CI & testing       | **GitHub Actions:** real Alertmanager in CI, data generation scripts, simulated Prometheus `/metrics`; validate APIs, pattern detection, ingestion, suppression, train/detect_changes |

---

## 12. References

- [Prometheus Alertmanager API v2](https://github.com/prometheus/alertmanager/blob/main/api/v2/openapi.yaml) — receive alerts (`POST /api/v2/alerts`), silences (`POST /api/v2/silences`).
- [Grafana Alerting](https://grafana.com/docs/grafana/latest/alerting/) — uses Alertmanager as backend; contact points, webhooks.
- [metrico/logparser](https://github.com/metrico/logparser) — pattern extraction, level detection, clustering (batch; use as design reference).
- [PicoClaw](https://github.com/sipeed/picoclaw) — Go agent, skills, TOOLS.md, HEARTBEAT, gateway.
- [PicoClaw custom skills](https://picoclaw.online/guide/custom-skills) — AGENTS.md, skills.
- [Spell (Go)](https://github.com/pfeak/spell) — streaming log template extraction.
- Drain3 / Drain — streaming template mining (for comparison or port).

This plan keeps the “important vs. noisy” product goal central, uses Go and PicoClaw (or its ideas), makes data sources and mappings extensible, and pushes logparser-style logic into a fast realtime pattern layer with LLM and one-click suppression on top.
