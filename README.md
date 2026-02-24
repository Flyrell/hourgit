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
hourgit init --project <project_name>
```

If the project doesn't exist yet, it will be created for you.

## Commands

### `hourgit init`

Initialize Hourgit in the current git repository by installing a post-checkout hook.

```bash
hourgit init [--project <project_name>] [--force] [--merge] [--yes]
```

| Flag | Description |
|------|-------------|
| `--project` | Assign repository to a project by name or ID (creates if needed) |
| `--force` | Overwrite existing post-checkout hook |
| `--merge` | Append to existing post-checkout hook |
| `--yes` | Skip confirmation prompt |

### `hourgit completion`

Manage shell completions. Supported shells: `bash`, `zsh`, `fish`, `powershell`.

#### `hourgit completion install`

Install shell completions into your shell config file. Auto-detects your shell if not specified.

```bash
hourgit completion install [SHELL] [--yes]
```

| Flag | Description |
|------|-------------|
| `--yes` | Skip confirmation prompt |


#### `hourgit completion generate`

Generate a shell completion script. If no shell is specified, Hourgit auto-detects it from the `$SHELL` environment variable.

```bash
hourgit completion generate [SHELL]
```

Shell completions are also offered automatically during `hourgit init`.

For manual setup, add the appropriate line to your shell config:

```bash
# zsh (~/.zshrc)
eval "$(hourgit completion generate zsh)"

# bash (~/.bashrc)
eval "$(hourgit completion generate bash)"

# fish (~/.config/fish/config.fish)
hourgit completion generate fish | source
```

### `hourgit log`

Manually log time for a project. Uses a hybrid mode: provide any combination of flags and you'll be prompted only for the missing pieces.

**Fully specified** — no prompts:

```bash
hourgit log --duration 3h "did some work"
hourgit log --from 9am --to 12pm "morning work"
hourgit log --duration 3h --date 2025-01-10 "forgot to log"
```

**Partial flags** — prompted for the rest:

```bash
hourgit log --duration 3h          # prompted for message
hourgit log --from 9am             # prompted for --to and message
hourgit log "meeting notes"        # prompted for time mode and inputs
hourgit log --date 2025-01-10      # prompted for time mode, inputs, and message
```

**Fully interactive** — guided prompts for everything:

```bash
hourgit log
```

| Flag | Description |
|------|-------------|
| `--project` | Project name or ID (auto-detected from repo if omitted) |
| `--duration` | Duration to log (e.g. `30m`, `3h`, `1d3h30m`) |
| `--from` | Start time (e.g. `9am`, `14:00`) |
| `--to` | End time (e.g. `5pm`, `17:00`) |
| `--date` | Date to log for (`YYYY-MM-DD`, default: today) |
| `--task` | Task label for this entry |

Notes:
- `--duration` and `--from`/`--to` are mutually exclusive
- A message is always required (prompted if not provided)

### `hourgit edit`

Edit an existing log entry by its hash. Supports two modes:

**Flag mode** — when any edit flag is provided, apply only those changes directly:

```bash
hourgit edit <hash> --duration 3h
hourgit edit <hash> --from 9am --to 12pm
hourgit edit <hash> --task "reviews"
hourgit edit <hash> --date 2025-02-20
hourgit edit <hash> -m "updated message"
```

**Interactive mode** — when no edit flags provided, prompts for each field with current values pre-filled:

```bash
hourgit edit <hash>
```

| Flag | Description |
|------|-------------|
| `--project` | Project name or ID (auto-detected from repo if omitted) |
| `--duration` | New duration (e.g. `30m`, `3h`, `3h30m`) |
| `--from` | New start time (e.g. `9am`, `14:00`) |
| `--to` | New end time (e.g. `5pm`, `17:00`) |
| `--date` | New date (`YYYY-MM-DD`) |
| `--task` | New task label (empty string clears it) |
| `-m`, `--message` | New message |

Notes:
- `--duration` and `--from`/`--to` are mutually exclusive
- `--from` only: keeps existing end time, recalculates duration
- `--to` only: keeps existing start time, recalculates duration
- Entry ID and creation timestamp are preserved
- If the entry is not found in the current repo's project, all projects are searched

### `hourgit checkout`

Record a branch checkout event. This command is called internally by the post-checkout git hook to track branch transitions.

```bash
hourgit checkout --prev <branch> --next <branch> [--project <project_name>]
```

| Flag | Description |
|------|-------------|
| `--prev` | Previous branch name (required) |
| `--next` | Next branch name (required) |
| `--project` | Project name or ID (auto-detected from repo if omitted) |

### `hourgit report`

Generate a monthly time report as an interactive table showing tasks (rows) × days (columns). Time is attributed to branches based on checkout events clipped to your configured schedule, with manual log entries shown alongside.

```bash
hourgit report [--month <1-12>] [--year <YYYY>] [--project <project_name>]
```

| Flag | Description |
|------|-------------|
| `--month` | Month number 1-12 (default: current month) |
| `--year` | Year (default: current year) |
| `--project` | Project name or ID (auto-detected from repo if omitted) |

The table shows:
- Each row is a task (branch name or manual log task/message)
- Each column is a day of the month
- Totals are shown in brackets next to task names
- Use `←`/`→` arrow keys to scroll horizontally
- Press `q`, `Esc`, or `Ctrl+C` to quit

In non-interactive environments (piped output), a static table is printed instead.

### `hourgit history`

Show a chronological feed of all recorded activity (log entries and checkout events) across projects, newest first.

```bash
hourgit history [--project <project_name>] [--limit <N>]
```

| Flag | Description |
|------|-------------|
| `--project` | Filter by project name or ID |
| `--limit` | Maximum number of entries to show (default: 50, use 0 for all) |

Each line shows the entry hash, timestamp, type (log or checkout), project name, and details:
- **Log entries:** duration + task label (if set) + message
- **Checkout entries:** previous branch → next branch

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
hourgit project add <project_name>
```

#### `hourgit project assign`

Assign the current repository to a project.

```bash
hourgit project assign <project_name> [--force] [--yes]
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
hourgit project remove <project_name> [--yes]
```

| Flag | Description |
|------|-------------|
| `--yes` | Skip confirmation prompt |

### `hourgit config`

Manage per-project schedule configuration. If `--project` is omitted, the project is auto-detected from the current repository.

#### `hourgit config get`

Show the schedule configuration for a project.

```bash
hourgit config get [--project <project_name> or <project_id>]
```

#### `hourgit config set`

Interactively edit a project's schedule using a guided schedule builder.

```bash
hourgit config set [--project <project_name> or <project_id>]
```

#### `hourgit config reset`

Reset a project's schedule to the defaults.

```bash
hourgit config reset [--project <project_name> or <project_id>] [--yes]
```

#### `hourgit config report`

Show expanded working hours for a given month (resolves schedule rules into concrete days and time ranges).

```bash
hourgit config report [--project <project_name> or <project_id>] [--month <1-12>] [--year <YYYY>]
```

| Flag | Description |
|------|-------------|
| `--project` | Project name or ID (auto-detected from repo if omitted) |
| `--month` | Month number 1-12 (default: current month) |
| `--year` | Year (default: current year) |

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

#### `hourgit defaults report`

Show expanded default working hours for a given month.

```bash
hourgit defaults report [--month <1-12>] [--year <YYYY>]
```

| Flag | Description |
|------|-------------|
| `--month` | Month number 1-12 (default: current month) |
| `--year` | Year (default: current year) |

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
| `~/.hourgit/<slug>/<hash>` | Per-project log entries (one JSON file per entry) |
| `~/.hourgit/<slug>/checkouts/<hash>` | Per-project checkout entries (one JSON file per checkout event) |

## Roadmap

The following features are planned but not yet implemented:

- **Automatic time logging** — automatic time calculation from checkout entries
- **Deleting entries** — remove entries by hash
- **Status** — show currently active branch/project and time logged today

## License

This project is licensed under the [Functional Source License, Version 1.1, MIT Future License (FSL-1.1-MIT)](LICENSE).
