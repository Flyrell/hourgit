# Project Management

Group repositories into projects for organized time tracking.

## `hourgit project add`

Create a new project.

```bash
hourgit project add <name> [--mode <mode>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-m`, `--mode` | `standard` | Tracking mode: `standard` or `precise` (enables filesystem watcher for idle detection) |

## `hourgit project assign`

Assign the current repository to a project.

```bash
hourgit project assign <name> [--force] [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-f`, `--force` | `false` | Reassign repository to a different project |
| `-y`, `--yes` | `false` | Skip confirmation prompt |

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
| `-y`, `--yes` | `false` | Skip confirmation prompt |
