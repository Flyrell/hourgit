# Data Storage

All Hourgit data is stored locally on your machine. There are no servers or cloud sync.

## File Locations

| Path | Purpose |
|------|---------|
| `~/.hourgit/config.json` | Global config — defaults, projects (id, name, slug, repos, schedules) |
| `REPO/.git/.hourgit` | Per-repo project assignment (project name + project ID) |
| `~/.hourgit/<slug>/<hash>` | Per-project entries (one JSON file per entry) |

## Entry Types

Each entry is a JSON file identified by a 7-character hex hash (similar to git commit hashes). The `type` field distinguishes between:

- **`log`** — manually logged time entry (duration, start time, message, task label)
- **`checkout`** — branch checkout event recorded by the git hook (previous branch, next branch, timestamp)
- **`submit`** — submission marker for a report period (date range, creation timestamp)

## Projects

Projects are defined in `~/.hourgit/config.json` and contain:

- **id** — unique identifier
- **name** — display name
- **slug** — filesystem-safe name (used as directory name under `~/.hourgit/`)
- **repos** — list of assigned repository paths
- **schedules** — per-project working hours configuration

## Per-Repo Assignment

When you run `hourgit init` or `hourgit project assign` in a git repository, a `.hourgit` file is created inside the repo's `.git/` directory. This file maps the repository to a project without modifying tracked files.
