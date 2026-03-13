# Hourgit

> Git-integrated time tracking for developers. No timers. No manual input. Just code.

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)
[![CI](https://github.com/Flyrell/hour-git/actions/workflows/ci.yml/badge.svg)](https://github.com/Flyrell/hour-git/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Flyrell/hour-git)](https://github.com/Flyrell/hour-git/releases)
[![Buy Me a Coffee](https://img.shields.io/badge/Buy%20Me%20a%20Coffee-ffdd00?logo=buy-me-a-coffee&logoColor=black)](https://buymeacoffee.com/dawidzbinski)

Hourgit tracks your working time automatically by hooking into git's checkout events. When you switch branches, Hourgit starts attributing time to the new branch. Your configured working hours act as the boundary — so overnight gaps, weekends, and days off are handled automatically without any extra input from you.

Each unit of logged time is called a **log entry**, identified by a short hash (similar to git commits). The data model is intentionally flat: a log entry is a time range + optional description + optional metadata (branch, project, task label). There is no hierarchy beyond project — grouping is derived at report time, not stored structurally.

Manual logging is supported for non-code work (research, analysis, meetings) via explicit commands.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands](#commands)
  - [Time Tracking](#time-tracking) — init, log, edit, remove, sync, report, history, status
  - [Project Management](#project-management) — project add/assign/list/remove
  - [Schedule Configuration](#schedule-configuration) — config get/set/reset/report
  - [Default Schedule](#default-schedule) — defaults get/set/reset/report
  - [Shell Completions](#shell-completions) — completion install/generate
  - [Other](#other) — version, watch
- [Precise Mode](#precise-mode)
- [Configuration](#configuration)
- [Data Storage](#data-storage)
- [Roadmap](#roadmap)
- [Sponsor](#sponsor)
- [License](#license)

## Installation

### Quick install (macOS and Linux)

```bash
curl -fsSL https://hourgit.com/install.sh | bash
```

This downloads the latest release, verifies the checksum, and installs to `~/.hourgit/bin/` with a symlink in `~/.local/bin/`. No `sudo` required.

### Manual install

Download the latest binary for your platform from the [Releases page](https://github.com/Flyrell/hour-git/releases/latest).

#### macOS

```bash
# Apple Silicon (M1/M2/M3/M4)
chmod +x hourgit-darwin-arm64-*
sudo mv hourgit-darwin-arm64-* /usr/local/bin/hourgit

# Intel
chmod +x hourgit-darwin-amd64-*
sudo mv hourgit-darwin-amd64-* /usr/local/bin/hourgit
```

#### Linux

```bash
# x86_64
chmod +x hourgit-linux-amd64-*
sudo mv hourgit-linux-amd64-* /usr/local/bin/hourgit

# ARM64
chmod +x hourgit-linux-arm64-*
sudo mv hourgit-linux-arm64-* /usr/local/bin/hourgit
```

#### Windows

Move `hourgit-windows-amd64-*.exe` to a directory in your `PATH` and rename it to `hourgit.exe`.

#### Verify

```bash
hourgit version
```

## Quick Start

1. **Install the hook** in your git repository:
   ```bash
   hourgit init
   ```

2. **Work normally** — time tracks automatically on every `git checkout`.

3. **Log non-git work** manually:
   ```bash
   hourgit log --duration 1h30m "standup"
   hourgit log --from 9am --to 10:30am "standup"
   ```

4. **View the interactive report:**
   ```bash
   hourgit report
   ```
   Navigate with arrow keys, press `e` to edit entries, `a` to add new ones. Checkout-derived time appears automatically (marked with `*`). Press `s` to **submit** the period — this persists all generated entries and marks the period as complete.

5. **Export a PDF** for sharing:
   ```bash
   hourgit report --export pdf
   ```

## Commands

### Time Tracking

Core commands for recording, viewing, and managing your time entries.

Commands: `init` · `log` · `edit` · `remove` · `sync` · `report` · `history` · `status`

#### `hourgit init`

Initialize Hourgit in the current git repository by installing a post-checkout hook.

```bash
hourgit init [--project <name>] [--mode <mode>] [--force] [--merge] [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-p`, `--project` | auto-detect | Assign repository to a project by name or ID (creates if needed) |
| `--mode` | `standard` | Tracking mode: `standard` or `precise` (enables filesystem watcher for idle detection) |
| `-f`, `--force` | `false` | Overwrite existing post-checkout hook |
| `-m`, `--merge` | `false` | Append to existing post-checkout hook |
| `-y`, `--yes` | `false` | Skip confirmation prompt |

#### `hourgit log`

Manually log time for a project. Uses a hybrid mode: provide any combination of flags and you'll be prompted only for the missing pieces.

```bash
hourgit log [MESSAGE] [--duration <dur>] [--from <time>] [--to <time>] [--date <date>] [--task <label>] [--project <name>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-p`, `--project` | auto-detect | Project name or ID |
| `-d`, `--duration` | — | Duration to log (e.g. `30m`, `3h`, `1d3h30m`) |
| `-F`, `--from` | — | Start time (e.g. `9am`, `14:00`) |
| `-T`, `--to` | — | End time (e.g. `5pm`, `17:00`) |
| `-D`, `--date` | today | Date to log for (`YYYY-MM-DD`) |
| `-t`, `--task` | — | Task label for this entry |

> `--duration` and `--from`/`--to` are mutually exclusive. A message is always required (prompted if not provided).

**Examples**

```bash
# Fully specified — no prompts
hourgit log --duration 3h "did some work"
hourgit log --from 9am --to 12pm "morning work"
hourgit log --duration 3h --date 2025-01-10 "forgot to log"

# Partial flags — prompted for the rest
hourgit log --duration 3h          # prompted for message
hourgit log --from 9am             # prompted for --to and message
hourgit log "meeting notes"        # prompted for time mode and inputs

# Fully interactive
hourgit log
```

#### `hourgit edit`

Edit an existing log entry by its hash. When edit flags are provided, only those changes are applied directly. Without flags, an interactive editor opens with current values pre-filled.

```bash
hourgit edit <hash> [--duration <dur>] [--from <time>] [--to <time>] [--date <date>] [--task <label>] [-m <msg>] [--project <name>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-p`, `--project` | auto-detect | Project name or ID |
| `-d`, `--duration` | — | New duration (e.g. `30m`, `3h`, `3h30m`) |
| `-F`, `--from` | — | New start time (e.g. `9am`, `14:00`) |
| `-T`, `--to` | — | New end time (e.g. `5pm`, `17:00`) |
| `-D`, `--date` | — | New date (`YYYY-MM-DD`) |
| `-t`, `--task` | — | New task label (empty string clears it) |
| `-m`, `--message` | — | New message |
| `-y`, `--yes` | `false` | Skip confirmation prompt |

> `--duration` and `--from`/`--to` are mutually exclusive. `--from` only: keeps existing end time, recalculates duration. `--to` only: keeps existing start time, recalculates duration. Entry ID and creation timestamp are preserved. If the entry is not found in the current repo's project, all projects are searched.

**Examples**

```bash
hourgit edit abc1234 --duration 3h
hourgit edit abc1234 --from 9am --to 12pm
hourgit edit abc1234 --task "reviews"
hourgit edit abc1234 -m "updated message"
hourgit edit abc1234              # interactive mode
```

#### `hourgit remove`

Remove a log or checkout entry by its hash.

```bash
hourgit remove <hash> [--project <name>] [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-p`, `--project` | auto-detect | Project name or ID |
| `-y`, `--yes` | `false` | Skip confirmation prompt |

> Works with both log and checkout entries (unlike `edit`, which only supports log entries). Shows entry details and asks for confirmation before deleting. If the entry is not found in the current repo's project, all projects are searched.

#### `hourgit sync`

Sync branch checkouts and commits from git reflog. Called automatically by the post-checkout hook, or run manually to backfill history. Commits are used to split checkout sessions into finer time blocks with commit messages.

```bash
hourgit sync [--project <name>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-p`, `--project` | auto-detect | Project name or ID |

#### `hourgit report`

Interactive time report with inline editing. Shows tasks (rows) × days (columns) with time attributed from branch checkouts, commits, and manual log entries. Checkout sessions are automatically split by commits, showing commit messages in a detail panel below the table.

```bash
hourgit report [--month <1-12>] [--week <1-53>] [--year <YYYY>] [--project <name>] [--export <format>] [--detail <level>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-m`, `--month` | current month | Month number 1-12 |
| `-w`, `--week` | — | ISO week number 1-53 |
| `-y`, `--year` | current year | Year (complementary to `--month` or `--week`) |
| `-p`, `--project` | auto-detect | Project name or ID |
| `-e`, `--export` | — | Export format (`pdf`); auto-generates filename based on period |
| `-d`, `--detail` | `summary` | Export detail level: `summary` (one row per task) or `full` (individual entries with commit messages) |

> `--month` and `--week` cannot be used together. `--year` alone is not valid — it must be paired with `--month` or `--week`. Neither flag defaults to the current month.

**Interactive table keybindings:**

| Key | Action |
|-----|--------|
| `←`/`→`/`↑`/`↓` or `h`/`l`/`k`/`j` | Navigate cells |
| `Tab` or `]` / `Shift+Tab` or `[` | Cycle through entries in selected cell |
| `e` | Edit selected cell entry |
| `a` | Add a new entry to selected cell |
| `r` or `Del` | Remove entry from selected cell |
| `s` | Submit period (persists all generated entries) |
| `q` or `Esc` | Quit |

In-memory generated entries (from checkout attribution) are marked with `*` in the table. Editing a generated entry persists it immediately. Submitting persists all remaining generated entries and creates a submit marker.

Previously submitted periods show a warning banner and can be re-edited and re-submitted. In non-interactive environments (piped output), a static table is printed instead.

**Examples**

```bash
hourgit report                                    # current month, interactive
hourgit report --week 8                           # ISO week 8
hourgit report --export pdf                       # export PDF (<project>-<YYYY>-month-<MM>.pdf)
hourgit report --export pdf --week 8              # export PDF (<project>-<YYYY>-week-<WW>.pdf)
hourgit report --export pdf --month 1 --year 2025
```

#### `hourgit history`

Show a chronological feed of all recorded activity (log entries and checkout events), newest first.

```bash
hourgit history [--project <name>] [--limit <N>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-p`, `--project` | all projects | Filter by project name or ID |
| `-l`, `--limit` | `50` | Maximum number of entries to show (use `0` for all) |

> Each line shows the entry hash, timestamp, type (log or checkout), project name, and details. Log entries display duration + task label (if set) + message. Checkout entries display previous branch → next branch.

#### `hourgit status`

Show current tracking status — project, branch, time logged today, and schedule state.

```bash
hourgit status [--project <name>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-p`, `--project` | auto-detect | Project name or ID |

**Output includes:**

- Current project and branch
- Time since last checkout
- Time logged today and remaining scheduled hours
- Today's schedule windows
- Tracking state (active/inactive based on current time vs schedule)
- Watcher state (when precise mode is enabled: active/stopped)

### Project Management

Group repositories into projects for organized time tracking.

Commands: `project add` · `project assign` · `project list` · `project remove`

#### `hourgit project add`

Create a new project.

```bash
hourgit project add <name> [--mode <mode>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--mode` | `standard` | Tracking mode: `standard` or `precise` (enables filesystem watcher for idle detection) |

#### `hourgit project assign`

Assign the current repository to a project.

```bash
hourgit project assign <name> [--force] [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-f`, `--force` | `false` | Reassign repository to a different project |
| `-y`, `--yes` | `false` | Skip confirmation prompt |

#### `hourgit project list`

List all projects and their repositories.

```bash
hourgit project list
```

No flags.

#### `hourgit project remove`

Remove a project and clean up its repository assignments.

```bash
hourgit project remove <name> [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-y`, `--yes` | `false` | Skip confirmation prompt |

### Schedule Configuration

Manage per-project schedule configuration. If `--project` is omitted, the project is auto-detected from the current repository.

Commands: `config get` · `config set` · `config reset` · `config report`

#### `hourgit config get`

Show the schedule configuration for a project.

```bash
hourgit config get [--project <name>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-p`, `--project` | auto-detect | Project name or ID |

#### `hourgit config set`

Interactively edit a project's schedule using a guided schedule builder.

```bash
hourgit config set [--project <name>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-p`, `--project` | auto-detect | Project name or ID |

#### `hourgit config reset`

Reset a project's schedule to the defaults.

```bash
hourgit config reset [--project <name>] [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-p`, `--project` | auto-detect | Project name or ID |
| `-y`, `--yes` | `false` | Skip confirmation prompt |

#### `hourgit config report`

Show expanded working hours for a given month (resolves schedule rules into concrete days and time ranges).

```bash
hourgit config report [--project <name>] [--month <1-12>] [--year <YYYY>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-p`, `--project` | auto-detect | Project name or ID |
| `-m`, `--month` | current month | Month number 1-12 |
| `-y`, `--year` | current year | Year |

### Default Schedule

Manage the default schedule applied to new projects.

Commands: `defaults get` · `defaults set` · `defaults reset` · `defaults report`

#### `hourgit defaults get`

Show the default schedule for new projects.

```bash
hourgit defaults get
```

No flags.

#### `hourgit defaults set`

Interactively edit the default schedule for new projects.

```bash
hourgit defaults set
```

No flags.

#### `hourgit defaults reset`

Reset the default schedule to factory settings (Mon-Fri, 9 AM - 5 PM).

```bash
hourgit defaults reset [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-y`, `--yes` | `false` | Skip confirmation prompt |

#### `hourgit defaults report`

Show expanded default working hours for a given month.

```bash
hourgit defaults report [--month <1-12>] [--year <YYYY>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-m`, `--month` | current month | Month number 1-12 |
| `-y`, `--year` | current year | Year |

### Shell Completions

Set up tab completions for your shell. Supported shells: `bash`, `zsh`, `fish`, `powershell`.

Commands: `completion install` · `completion generate`

#### `hourgit completion install`

Install shell completions into your shell config file. Auto-detects your shell if not specified.

```bash
hourgit completion install [SHELL] [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-y`, `--yes` | `false` | Skip confirmation prompt |

#### `hourgit completion generate`

Generate a shell completion script. If no shell is specified, Hourgit auto-detects it from the `$SHELL` environment variable.

```bash
hourgit completion generate [SHELL]
```

No flags.

Shell completions are also offered automatically during `hourgit init`.

**Examples**

```bash
# zsh (~/.zshrc)
eval "$(hourgit completion generate zsh)"

# bash (~/.bashrc)
eval "$(hourgit completion generate bash)"

# fish (~/.config/fish/config.fish)
hourgit completion generate fish | source
```

### Other

Commands: `version` · `update` · `watch`

#### `hourgit version`

Print version information.

```bash
hourgit version
```

No flags.

#### `hourgit update`

Check for and install updates. Always checks the latest version from GitHub, bypassing the cache TTL used by the automatic update check.

```bash
hourgit update
```

No flags.

#### `hourgit watch`

Run the filesystem watcher daemon in the foreground. The daemon monitors file changes in repositories with precise mode enabled and writes activity entries to detect idle gaps. Normally managed automatically as an OS service — use this command for debugging or manual operation.

```bash
hourgit watch
```

No flags.

## Precise Mode

By default, Hourgit attributes all time between branch checkouts (within your schedule) as work. **Precise mode** adds filesystem-level idle detection: a background daemon watches your repository for file changes and records when you stop and resume working.

### How it works

1. A background daemon watches file changes in your repository (excluding `.git/` and `.gitignore` patterns).
2. After a configurable idle threshold (default: 10 minutes) with no file changes, the daemon records an `activity_stop` entry.
3. When file changes resume, the daemon records an `activity_start` entry.
4. At report time, these idle gaps are trimmed from checkout sessions, giving you more accurate time attribution.

### Enabling precise mode

```bash
# During init
hourgit init --mode precise

# When adding a project
hourgit project add myproject --mode precise
```

When precise mode is enabled, Hourgit automatically installs a user-level OS service (launchd on macOS, systemd on Linux, Task Scheduler on Windows) to run the watcher daemon. No `sudo` required.

### Health checks

Hourgit checks whether the watcher daemon is running on every command. If it's stopped, you'll be prompted to restart it. The `status` command shows the current watcher state when precise mode is enabled.

## Configuration

Hourgit uses a schedule system to define working hours. The factory default is **Monday-Friday, 9 AM - 5 PM**.

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
| `~/.hourgit/<slug>/<hash>` | Per-project entries (one JSON file per entry — log, checkout, commit, submit, activity_stop, activity_start) |
| `~/.hourgit/watch.pid` | PID file for the filesystem watcher daemon (precise mode) |
| `~/.hourgit/watch.state` | Watcher state file — last activity timestamps per repo (precise mode) |

## Roadmap

Planned features:

- **Exporters** — export time data to platforms like Jira, Toggl, Clockify, and Harvest
- **Web app** — browser-based interface for viewing and managing time entries
- **GUI** — potentially add a graphical interface on macOS, Linux, and Windows

Have an idea? [Open an issue](https://github.com/Flyrell/hourgit/issues).

## Sponsor

Hourgit is free and open-source software. Sponsorship funds continued development and maintenance, cross-platform testing and releases, and helps keep the project free for everyone.

| Tier | Amount | Description |
|------|--------|-------------|
| Base | $5/mo | Support continued development |
| Pro | $25/mo | Fund cross-platform testing and releases |
| Enterprise | $100/mo | Help shape the roadmap and keep Hourgit free |

[Become a sponsor on GitHub](https://github.com/sponsors/Flyrell)

## License

This project is licensed under the [GNU General Public License v3.0](LICENSE).
