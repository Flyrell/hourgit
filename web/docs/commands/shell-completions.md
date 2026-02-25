# Shell Completions

Set up tab completions for your shell. Supported shells: `bash`, `zsh`, `fish`, `powershell`.

## `hourgit completion install`

Install shell completions into your shell config file. Auto-detects your shell if not specified.

```bash
hourgit completion install [SHELL] [--yes]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--yes` | `false` | Skip confirmation prompt |

Shell completions are also offered automatically during `hourgit init`.

## `hourgit completion generate`

Generate a shell completion script. If no shell is specified, Hourgit auto-detects it from the `$SHELL` environment variable.

```bash
hourgit completion generate [SHELL]
```

**Examples:**

```bash
# zsh (~/.zshrc)
eval "$(hourgit completion generate zsh)"

# bash (~/.bashrc)
eval "$(hourgit completion generate bash)"

# fish (~/.config/fish/config.fish)
hourgit completion generate fish | source
```
