# PicoClaw integration (optional)

**AIlert does not require PicoClaw.** The CLI and all features (run, suppress, detect-changes, suggest-rules, apply-rule, Alertmanager) work on their own. This integration is **optional** for users who want [PicoClaw](https://github.com/sipeed/picoclaw) as the agent for LLM-related tasks: deciding suppress vs notify, summarizing changes, and applying rules from suggestions. The agent runs the `ailert` binary via the **exec** tool and uses the LLM to interpret output and act.

## How it works

- PicoClaw’s **exec** tool runs the `ailert` binary. No changes to PicoClaw code are required.
- The **alerting** skill (and optional TOOLS/AGENTS/HEARTBEAT snippets) tell the agent which commands to run and how to interpret results.
- **LLM tasks** handled by the agent:
  - **Suppress vs notify** — From `ailert suggest-rules` output, decide which patterns to suppress (expected/noisy) vs alert on (important).
  - **Summarize** — “What changed?”, “Summarize new patterns” from `ailert detect-changes` / `ailert suggest-rules`.
  - **Apply rules** — Run `ailert apply-rule suppress <hash>` or `ailert apply-rule alert <hash>` (and optionally `-create-silence`) after the LLM decides.

## Setup

1. **Build ailert** and make it available to PicoClaw:
   - Put `ailert` in your PicoClaw **workspace** (e.g. `~/.picoclaw/workspace/`) so the agent can run `./ailert ...` from the workspace, or
   - Install `ailert` in PATH so the agent can run `ailert ...` from any working directory.

2. **Config**: Place your AIlert config (e.g. `config.yaml`) in the workspace or set `AILERT_CONFIG` so the agent runs:
   - `ailert run -config /path/to/config.yaml -save-snapshot /path/to/snapshots`
   - `ailert suggest-rules -config /path/to/config.yaml -snapshot-dir /path/to/snapshots`
   - etc.

3. **Copy workspace files** from this directory into your PicoClaw workspace:
   - **Skill**: Copy `skills/alerting/` to `~/.picoclaw/workspace/skills/alerting/` (or your configured workspace).
   - **TOOLS.md**: Merge `TOOLS-ailert.md` into your workspace `TOOLS.md` (or create one) so the agent knows the ailert commands.
   - **HEARTBEAT.md**: Optionally use or merge `HEARTBEAT-ailert.md` for periodic “run → suggest → summarize/apply” tasks.
   - **AGENTS.md**: Optionally merge `AGENTS-ailert.md` if you want the default agent to be alerting-aware.

4. **Enable the skill**: In your workspace `AGENTS.md`, list the alerting skill so it’s loaded (see PicoClaw docs for how skills are selected).

## Files in this directory

| File | Purpose |
|------|--------|
| `README.md` | This file — setup and overview. |
| `skills/alerting/SKILL.md` | Skill text: how to run ailert via exec, when to run/suggest/suppress/alert, and how the LLM should decide suppress vs notify. |
| `TOOLS-ailert.md` | Snippet for TOOLS.md: short description of ailert commands for the LLM. |
| `HEARTBEAT-ailert.md` | Example periodic tasks (run, suggest-rules, summarize, optionally apply). |
| `AGENTS-ailert.md` | Optional agent snippet (alerting-aware persona and constraints). |

## Security

- The agent runs `ailert` via **exec**. If PicoClaw is configured with `restrict_to_workspace: true`, keep the ailert binary and config inside the workspace so paths stay inside the sandbox.
- Do not give the agent access to run arbitrary shell; it should only run the documented `ailert` subcommands (the skill describes these).
