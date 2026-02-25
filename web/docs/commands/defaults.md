# Default Schedule

Manage the default schedule applied to new projects. The factory default is **Monday-Friday, 9 AM - 5 PM**.

## `hourgit defaults get`

Show the default schedule for new projects.

```bash
hourgit defaults get
```

## `hourgit defaults set`

Interactively edit the default schedule for new projects.

```bash
hourgit defaults set
```

## `hourgit defaults reset`

Reset the default schedule to factory settings (Mon-Fri, 9 AM - 5 PM).

```bash
hourgit defaults reset [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--yes` | `false` | Skip confirmation prompt |

## `hourgit defaults report`

Show expanded default working hours for a given month.

```bash
hourgit defaults report [--month <1-12>] [--year <YYYY>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--month` | current month | Month number 1-12 |
| `--year` | current year | Year |
