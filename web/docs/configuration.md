# Configuration

Hourgit uses a schedule system to define working hours. The factory default is **Monday-Friday, 9 AM - 5 PM**.

## Schedule Types

The interactive schedule editor (`config set` / `defaults set`) supports three schedule types:

- **Recurring** — repeats on a regular pattern (e.g., every weekday, every Monday/Wednesday/Friday)
- **One-off** — applies to a single specific date (e.g., a holiday or overtime day)
- **Date range** — applies to a contiguous range of dates (e.g., a week with different hours)

Each schedule entry defines one or more time ranges for the days it covers. Multiple entries can be combined to build complex schedules.

## Per-Project Overrides

Every project starts with a copy of the defaults. You can then customize a project's schedule independently:

```bash
# View current schedule
hourgit config get --project 'My Project'

# Edit schedule interactively
hourgit config set --project 'My Project'

# Revert to defaults
hourgit config reset --project 'My Project'

# See expanded hours for a month
hourgit config report --project 'My Project' --month 3
```

## Editing Defaults

Changes to defaults only affect newly created projects. Existing projects keep their current schedule.

```bash
# View defaults
hourgit defaults get

# Edit defaults
hourgit defaults set

# Reset to factory (Mon-Fri, 9-5)
hourgit defaults reset
```
