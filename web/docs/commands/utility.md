# Utility

General-purpose commands for checking your Hourgit version and keeping it up to date.

## `hourgit version`

Print the current Hourgit version.

```bash
hourgit version
```

## `hourgit update`

Check for a newer version of Hourgit and install it.

```bash
hourgit update
```

This command always fetches the latest version from GitHub, bypassing the cached update check. If a newer version is available, you'll be prompted to install it.

> **Note:** Dev builds (`version = "dev"`) skip the update check automatically.

### Auto-update vs manual update

Hourgit also checks for updates automatically when you run any interactive command (with an 8-hour cache). The `update` command is for when you want to check right now, regardless of when the last check happened.
