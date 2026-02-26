# Time Tracking Commands

Core commands for recording, viewing, and managing your time entries.

## `hourgit init`

Initialize Hourgit in the current git repository by installing a post-checkout hook.

```bash
hourgit init [--project <name>] [--force] [--merge] [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | auto-detect | Assign repository to a project by name or ID (creates if needed) |
| `--force` | `false` | Overwrite existing post-checkout hook |
| `--merge` | `false` | Append to existing post-checkout hook |
| `--yes` | `false` | Skip confirmation prompt |

## `hourgit log`

Manually log time for a project. Uses a hybrid mode: provide any combination of flags and you'll be prompted only for the missing pieces.

```bash
hourgit log [MESSAGE] [--duration <dur>] [--from <time>] [--to <time>] [--date <date>] [--task <label>] [--project <name>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | auto-detect | Project name or ID |
| `--duration` | — | Duration to log (e.g. `30m`, `3h`, `1d3h30m`) |
| `--from` | — | Start time (e.g. `9am`, `14:00`) |
| `--to` | — | End time (e.g. `5pm`, `17:00`) |
| `--date` | today | Date to log for (`YYYY-MM-DD`) |
| `--task` | — | Task label for this entry |

> `--duration` and `--from`/`--to` are mutually exclusive. A message is always required (prompted if not provided).

**Examples:**

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

## `hourgit edit`

Edit an existing log entry by its hash. When edit flags are provided, only those changes are applied directly. Without flags, an interactive editor opens with current values pre-filled.

```bash
hourgit edit <hash> [--duration <dur>] [--from <time>] [--to <time>] [--date <date>] [--task <label>] [-m <msg>] [--project <name>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | auto-detect | Project name or ID |
| `--duration` | — | New duration (e.g. `30m`, `3h`, `3h30m`) |
| `--from` | — | New start time (e.g. `9am`, `14:00`) |
| `--to` | — | New end time (e.g. `5pm`, `17:00`) |
| `--date` | — | New date (`YYYY-MM-DD`) |
| `--task` | — | New task label (empty string clears it) |
| `-m`, `--message` | — | New message |

> `--duration` and `--from`/`--to` are mutually exclusive. Entry ID and creation timestamp are preserved.

**Examples:**

```bash
hourgit edit abc1234 --duration 3h
hourgit edit abc1234 --from 9am --to 12pm
hourgit edit abc1234 --task "reviews"
hourgit edit abc1234 -m "updated message"
hourgit edit abc1234              # interactive mode
```

## `hourgit remove`

Remove a log or checkout entry by its hash.

```bash
hourgit remove <hash> [--project <name>] [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | auto-detect | Project name or ID |
| `--yes` | `false` | Skip confirmation prompt |

> Works with both log and checkout entries. Shows entry details and asks for confirmation before deleting.

## `hourgit sync`

Sync branch checkouts from git reflog. Called automatically by the post-checkout hook, or run manually to backfill history.

```bash
hourgit sync [--project <name>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | auto-detect | Project name or ID |

## `hourgit report`

Interactive time report with inline editing. Shows tasks (rows) × days (columns) with time attributed from branch checkouts and manual log entries.

```bash
hourgit report [--month <1-12>] [--week <1-53>] [--year <YYYY>] [--project <name>] [--output <path>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--month` | current month | Month number 1-12 |
| `--week` | — | ISO week number 1-53 |
| `--year` | current year | Year |
| `--project` | auto-detect | Project name or ID |
| `--output` | — | Export as PDF timesheet |

> `--month` and `--week` cannot be used together.

**Interactive keybindings:**

| Key | Action |
|-----|--------|
| `←`/`→`/`↑`/`↓` or `h`/`l`/`k`/`j` | Navigate cells |
| `e` | Edit selected cell entry |
| `a` | Add a new entry to selected cell |
| `r` or `Del` | Remove entry from selected cell |
| `s` | Submit period |
| `q` or `Esc` | Quit |

**Examples:**

```bash
hourgit report                                    # current month, interactive
hourgit report --week 8                           # ISO week 8
hourgit report --output timesheet.pdf             # export PDF
hourgit report --output                           # auto-named PDF
hourgit report --output report.pdf --month 1 --year 2025
```

## `hourgit history`

Show a chronological feed of all recorded activity, newest first.

```bash
hourgit history [--project <name>] [--limit <N>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | all projects | Filter by project name or ID |
| `--limit` | `50` | Maximum number of entries to show (use `0` for all) |
