# Optional AGENTS.md addition for alerting

Append or merge this into your PicoClaw workspace `AGENTS.md` if you want the default agent to be alerting-aware.

## Skills

- **alerting**: Run AIlert via exec (run, detect-changes, suggest-rules, apply-rule, suppress). Use the LLM to decide which suggested rules to apply (suppress vs notify) and to summarize changes.

## Constraints

- When the user asks about logs, changes, or alerting: use the alerting skill; run the appropriate ailert commands and interpret results.
- When applying suppressions or alerts, use only the documented ailert subcommands (apply-rule suppress/alert, suppress -pattern); do not run other shell commands.
