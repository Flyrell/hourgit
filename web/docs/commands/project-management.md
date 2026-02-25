# Project Management

Group repositories into projects for organized time tracking.

## `hourgit project add`

Create a new project.

```bash
hourgit project add <name>
```

## `hourgit project assign`

Assign the current repository to a project.

```bash
hourgit project assign <name> [--force] [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | `false` | Reassign repository to a different project |
| `--yes` | `false` | Skip confirmation prompt |

## `hourgit project list`

List all projects and their repositories.

```bash
hourgit project list
```

## `hourgit project remove`

Remove a project and clean up its repository assignments.

```bash
hourgit project remove <name> [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--yes` | `false` | Skip confirmation prompt |
