# Data Storage

All Hourgit data is stored locally on your machine. There are no servers or cloud sync.

## File Locations

| Path | Purpose |
|------|---------|
| `~/.hourgit/config.json` | Global config — defaults, projects (id, name, slug, repos, schedules) |
| `REPO/.git/.hourgit` | Per-repo project assignment (project name + project ID) |
| `~/.hourgit/<slug>/<hash>` | Per-project entries (one JSON file per entry) |
| `~/.hourgit/watch.pid` | PID file for the filesystem watcher daemon (precise mode) |
| `~/.hourgit/watch.state` | Watcher state file — last activity timestamps per repo (precise mode) |

## Entry Types

Each entry is a JSON file identified by a 7-character hex hash (similar to git commit hashes). The `type` field distinguishes between:

- **`log`** — manually logged time entry (duration, start time, message, task label)
- **`checkout`** — branch checkout event recorded by the git hook (previous branch, next branch, timestamp, repo)
- **`commit`** — git commit event from reflog (commit ref, timestamp, message, branch, repo); used to split checkout sessions into finer time blocks
- **`submit`** — submission marker for a report period (date range, creation timestamp)
- **`activity_stop`** — idle detection: records when file activity stops (timestamp of last file change, repo path)
- **`activity_start`** — idle detection: records when file activity resumes (timestamp, repo path)

## Projects

Projects are defined in `~/.hourgit/config.json` and contain:

- **id** — unique identifier
- **name** — display name
- **slug** — filesystem-safe name (used as directory name under `~/.hourgit/`)
- **repos** — list of assigned repository paths
- **schedules** — per-project working hours configuration

## Per-Repo Assignment

When you run `hourgit init` or `hourgit project assign` in a git repository, a `.hourgit` file is created inside the repo's `.git/` directory. This file maps the repository to a project without modifying tracked files.
