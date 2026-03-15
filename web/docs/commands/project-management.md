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

Assign the current repository to a project. When no project is specified, auto-detects from the current repo's assignment.

```bash
hourgit project assign [PROJECT] [--project <name>] [--force] [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-p`, `--project` | auto-detect | Project name or ID (alternative to positional argument) |
| `-f`, `--force` | `false` | Reassign repository to a different project |
| `-y`, `--yes` | `false` | Skip confirmation prompt |

## `hourgit project edit`

Edit an existing project's name or tracking mode. When edit flags are provided, only those changes are applied directly. Without flags, an interactive editor prompts for both name and mode.

```bash
hourgit project edit [PROJECT] [--name <new_name>] [--mode <mode>] [--idle-threshold <minutes>] [--project <name>] [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-n`, `--name` | — | New project name |
| `-m`, `--mode` | — | New tracking mode: `standard` or `precise` |
| `-t`, `--idle-threshold` | — | Idle threshold in minutes (precise mode only) |
| `-p`, `--project` | auto-detect | Project name or ID (alternative to positional argument) |
| `-y`, `--yes` | `false` | Skip confirmation prompt |

## `hourgit project list`

List all projects and their repositories.

```bash
hourgit project list
```

## `hourgit project remove`

Remove a project and clean up its repository assignments. When no project is specified, auto-detects from the current repo's assignment.

```bash
hourgit project remove [PROJECT] [--project <name>] [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-p`, `--project` | auto-detect | Project name or ID (alternative to positional argument) |
| `-y`, `--yes` | `false` | Skip confirmation prompt |
