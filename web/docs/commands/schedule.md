# Schedule Configuration

Manage per-project schedule configuration. If `--project` is omitted, the project is auto-detected from the current repository.

## `hourgit config get`

Show the schedule configuration for a project.

```bash
hourgit config get [--project <name>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | auto-detect | Project name or ID |

## `hourgit config set`

Interactively edit a project's schedule using a guided schedule builder.

```bash
hourgit config set [--project <name>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | auto-detect | Project name or ID |

The interactive editor lets you define:
- **Recurring** schedules — repeats on a regular pattern (e.g., every weekday)
- **One-off** schedules — applies to a single specific date
- **Date range** schedules — applies to a contiguous range of dates

Each schedule entry defines one or more time ranges for the days it covers.

## `hourgit config reset`

Reset a project's schedule to the defaults.

```bash
hourgit config reset [--project <name>] [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | auto-detect | Project name or ID |
| `--yes` | `false` | Skip confirmation prompt |

## `hourgit config report`

Show expanded working hours for a given month (resolves schedule rules into concrete days and time ranges).

```bash
hourgit config report [--project <name>] [--month <1-12>] [--year <YYYY>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--project` | auto-detect | Project name or ID |
| `--month` | current month | Month number 1-12 |
| `--year` | current year | Year |
