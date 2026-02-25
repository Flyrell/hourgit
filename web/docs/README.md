# Hourgit

> Git-integrated time tracking for developers. No timers. No manual input. Just code.

Hourgit tracks your working time automatically by hooking into git's checkout events. When you switch branches, Hourgit starts attributing time to the new branch. Your configured working hours act as the boundary — so overnight gaps, weekends, and days off are handled automatically.

Each unit of logged time is called a **log entry**, identified by a short hash (similar to git commits). The data model is intentionally flat: a log entry is a time range + optional description + optional metadata (branch, project, task label). Grouping is derived at report time, not stored structurally.

Manual logging is supported for non-code work (research, analysis, meetings) via explicit commands.

## Key Features

- **Automatic tracking** — time is attributed to branches via git post-checkout hooks
- **Working hours aware** — configurable schedules handle overnight gaps and weekends
- **Manual logging** — log meetings, reviews, and other non-code work
- **Interactive reports** — navigate a tasks × days table, edit inline, submit when ready
- **PDF export** — generate timesheets for sharing
- **Project management** — group repositories into projects with independent schedules
- **Zero dependencies** — single binary, no runtime requirements beyond git
