# Hourgit (in development)

> Git-integrated time tracking for developers. No timers. No manual input. Just code.

## Overview

Hourgit tracks your working time automatically by hooking into git's checkout events. When you switch branches, Hourgit starts attributing time to the new branch. Your configured working hours act as the boundary — so overnight gaps, weekends, and days off are handled automatically without any extra input from you.

Each unit of logged time is called a **log entry**, identified by a short hash (similar to git commits). The data model is intentionally flat: a log entry is a time range + optional description + optional metadata (branch, project, task label). There is no hierarchy beyond project — grouping is derived at report time, not stored structurally.

Manual logging is supported for non-code work (research, analysis, meetings) via explicit commands.

## Installation

Installation instructions will be available once the first release is published.

## Getting Started

Initialize Hourgit in a git repository:

```bash
hourgit init
```

This installs a post-checkout hook that automatically tracks branch switches. You can optionally assign the repo to a project during initialization:

```bash
hourgit init --project my-project
```

If the project doesn't exist yet, it will be created for you.

## Commands

### `hourgit init`

Initialize Hourgit in the current git repository by installing a post-checkout hook.

```bash
hourgit init [--project NAME] [--force] [--merge] [--yes]
```

| Flag | Description |
|------|-------------|
| `--project` | Assign repository to a project by name or ID (creates if needed) |
| `--force` | Overwrite existing post-checkout hook |
| `--merge` | Append to existing post-checkout hook |
| `--yes` | Skip confirmation prompt |

### `hourgit version`

Print version information.

```bash
hourgit version
```

### `hourgit project`

Manage projects. Projects group repositories together and hold schedule configuration.

#### `hourgit project add`

Create a new project.

```bash
hourgit project add PROJECT
```

#### `hourgit project assign`

Assign the current repository to a project.

```bash
hourgit project assign PROJECT [--force] [--yes]
```

| Flag | Description |
|------|-------------|
| `--force` | Reassign repository to a different project |
| `--yes` | Skip confirmation prompt |

#### `hourgit project list`

List all projects and their repositories.

```bash
hourgit project list
```

#### `hourgit project remove`

Remove a project and clean up its repository assignments.

```bash
hourgit project remove PROJECT [--yes]
```

| Flag | Description |
|------|-------------|
| `--yes` | Skip confirmation prompt |

### `hourgit config`

Manage per-project schedule configuration. If `--project` is omitted, the project is auto-detected from the current repository.

#### `hourgit config get`

Show the schedule configuration for a project.

```bash
hourgit config get [--project NAME]
```

#### `hourgit config set`

Interactively edit a project's schedule using a guided schedule builder.

```bash
hourgit config set [--project NAME]
```

#### `hourgit config reset`

Reset a project's schedule to the defaults.

```bash
hourgit config reset [--project NAME] [--yes]
```

#### `hourgit config read`

Show expanded working hours for the current month (resolves schedule rules into concrete days and time ranges).

```bash
hourgit config read [--project NAME]
```

### `hourgit defaults`

Manage the default schedule applied to new projects.

#### `hourgit defaults get`

Show the default schedule for new projects.

```bash
hourgit defaults get
```

#### `hourgit defaults set`

Interactively edit the default schedule for new projects.

```bash
hourgit defaults set
```

#### `hourgit defaults reset`

Reset the default schedule to factory settings (Mon–Fri, 9 AM – 5 PM).

```bash
hourgit defaults reset [--yes]
```

#### `hourgit defaults read`

Show expanded default working hours for the current month.

```bash
hourgit defaults read
```

## Configuration

Hourgit uses a schedule system to define working hours. The factory default is **Monday–Friday, 9 AM – 5 PM**.

### Schedule types

The interactive schedule editor (`config set` / `defaults set`) supports three schedule types:

- **Recurring** — repeats on a regular pattern (e.g., every weekday, every Monday/Wednesday/Friday)
- **One-off** — applies to a single specific date (e.g., a holiday or overtime day)
- **Date range** — applies to a contiguous range of dates (e.g., a week with different hours)

Each schedule entry defines one or more time ranges for the days it covers. Multiple entries can be combined to build complex schedules.

### Per-project overrides

Every project starts with a copy of the defaults. You can then customize a project's schedule independently using `hourgit config set --project NAME`. To revert a project back to the current defaults, use `hourgit config reset --project NAME`.

## Data Storage

| Path | Purpose |
|------|---------|
| `~/.hourgit/config.json` | Global config — defaults, projects (id, name, slug, repos, schedules) |
| `REPO/.git/.hourgit` | Per-repo project assignment (project name + project ID) |
| `~/.hourgit/<slug>/` | Per-project log directory (for future log entries) |

## Roadmap

The following features are planned but not yet implemented:

- **Time logging** — automatic logging via the post-checkout hook, manual log entries with duration or time range
- **Log history** — view logged entries with hashes
- **Editing and deleting entries** — update time ranges, descriptions, or project assignments by hash
- **Reports** — group time by branch, project, day, or task
- **Status** — show currently active branch/project and time logged today

## License

This project is licensed under the [Functional Source License, Version 1.1, MIT Future License (FSL-1.1-MIT)](LICENSE).
