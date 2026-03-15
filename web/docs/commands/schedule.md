# Schedule Configuration

Manage per-project schedule configuration. If `--project` is omitted, the project is auto-detected from the current repository.

## `hourgit project schedule get`

Show the schedule configuration for a project.

```bash
hourgit project schedule get [--project <name>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-p`, `--project` | auto-detect | Project name or ID |

## `hourgit project schedule set`

Interactively edit a project's schedule using a guided schedule builder.

```bash
hourgit project schedule set [--project <name>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-p`, `--project` | auto-detect | Project name or ID |

The interactive editor lets you define:
- **Recurring** schedules — repeats on a regular pattern (e.g., every weekday)
- **One-off** schedules — applies to a single specific date
- **Date range** schedules — applies to a contiguous range of dates

Each schedule entry defines one or more time ranges for the days it covers.

## `hourgit project schedule reset`

Reset a project's schedule to the defaults.

```bash
hourgit project schedule reset [--project <name>] [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-p`, `--project` | auto-detect | Project name or ID |
| `-y`, `--yes` | `false` | Skip confirmation prompt |

## `hourgit project schedule report`

Show expanded working hours for a given month (resolves schedule rules into concrete days and time ranges).

```bash
hourgit project schedule report [--project <name>] [--month <1-12>] [--year <YYYY>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-p`, `--project` | auto-detect | Project name or ID |
| `-m`, `--month` | current month | Month number 1-12 |
| `-y`, `--year` | current year | Year |
