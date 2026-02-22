# Hourgit

> Git-integrated time tracking for developers. No timers. No manual input. Just code.

## Overview

Hourgit tracks your working time automatically by hooking into git's checkout events. When you switch branches, Hourgit starts attributing time to the new branch. Your configured working hours act as the boundary — so overnight gaps, weekends, and days off are handled automatically without any extra input from you.

Each unit of logged time is called a **log entry**, identified by a short hash (similar to git commits). The data model is intentionally flat: a log entry is a time range + optional description + optional metadata (branch, project, task label). There is no hierarchy beyond project — grouping is derived at report time, not stored structurally.

---

## Implementation Status

This project is currently in the **design and planning phase**. No code has been written yet. The commands and data model described in this document represent the intended design, not an existing implementation. The goal is to build towards this step by step.

> **Note:** If anything encountered during development contradicts, omits, or doesn't fit the intended design, the design should be flagged and updated before proceeding. This document should always reflect current intentions accurately.

---

## How It Works

1. You run `hourgit init` inside a git repository — this installs a post-checkout hook.
2. Every time you run `git checkout`, the hook fires and logs the branch switch.
3. Hourgit calculates time spent on each branch using the timestamps of checkout events, trimmed to your configured working hours.
4. If you work on a branch for 3 days without switching, Hourgit attributes 3 × (working hours) to that branch automatically.
5. Manual log entries let you track non-code work (research, analysis, meetings) outside of any branch context.

---

## Configuration

### View current defaults

```bash
hourgit defaults get
```

### Set default working hours

```bash
hourgit defaults set from 9AM to 5PM
```

### Override working hours for a specific day

```bash
hourgit defaults set from 9AM to 6PM on Monday
```

---

## Projects

### Create a project

```bash
hourgit add project project_name
```

### Initialize a repo and attach it to a project

```bash
hourgit init --project project_name
```

### Initialize a repo without a project

```bash
hourgit init
```

### Attach an already-initialized repo to a project

```bash
hourgit set project project_name
```

---

## Logging

Automatic logging happens via the post-checkout hook using an internal command:

```bash
hourgit log --type checkout branch_name
```

> This command is internal and called by the hook — you don't need to run it manually.

### Manually log time ending now

```bash
hourgit log --duration 4h "analyzed competitor pricing"
```

If it's 4PM and you log `--duration 4h`, the entry is attributed to 12PM–4PM (i.e. ending now).

### Manually log a specific time range

```bash
hourgit log --from 9AM --to 1PM "analyzed competitor pricing"
```

### Manual log attached to a project

```bash
hourgit log --duration 4h --project project_name "analyzed competitor pricing"
```

### Manual log for a past date

```bash
hourgit log --duration 4h --date yesterday "analyzed competitor pricing"
```

---

## Editing Log Entries

### View log history with hashes

```bash
hourgit log --history
```

Example output:

```
a3f9c21  2h  branch: feature/auth     "implemented login flow"
b12e884  4h  task: analysis           "analyzed competitor pricing"
c9d3a11  8h  branch: fix/login-bug    "fixed session expiry"
```

### Edit time range of a log entry

```bash
hourgit update <hash> --from 8AM --to 10AM
```

### Add or edit description of a log entry

```bash
hourgit update <hash> --describe "refactored auth middleware"
```

### Reassign a log entry to a project

```bash
hourgit update <hash> --project project_name
```

### Delete a log entry

```bash
hourgit delete <hash>
```

---

## Reports

### Default report (grouped by branch)

```bash
hourgit report
```

### Group by project

```bash
hourgit report --by project
```

### Group by day

```bash
hourgit report --by day
```

Example report output:

```
my_project
├── feature/auth          12h
├── fix/login-bug          3h
└── [manual] analysis      4h
```

---

## Status

Check the currently active branch/project and time logged today:

```bash
hourgit status
```

---

## Data Model

A log entry contains:

| Field       | Description                                      |
|-------------|--------------------------------------------------|
| `hash`      | Short unique identifier (git-style)              |
| `from`      | Start time                                       |
| `to`        | End time                                         |
| `type`      | `branch` (automatic) or `manual`                 |
| `branch`    | Branch name (if applicable)                      |
| `project`   | Project name (optional)                          |
| `description` | Free-text description of work done (optional) |

Automatic branch logs are split by day, giving each day its own hash and making individual edits straightforward.

---

## Why Hourgit?

Most time tracking tools ask you to change your behavior — start a timer, stop a timer, fill in a form. Hourgit works the other way: it observes what you're already doing (switching branches) and infers your time from that, bounded by the working hours you've defined. The result is time tracking that requires almost no effort, with enough manual override capability to stay accurate.