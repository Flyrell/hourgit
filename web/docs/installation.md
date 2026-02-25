# Installation

## Quick Install (macOS & Linux)

```bash
curl -fsSL https://hourgit.com/install.sh | bash
```

This downloads the latest release, verifies the SHA256 checksum, and installs to `~/.hourgit/bin/` with a symlink in `~/.local/bin/`. No `sudo` required.

## Manual Install

Download the latest binary for your platform from the [Releases page](https://github.com/Flyrell/hourgit/releases/latest).

### macOS

```bash
# Apple Silicon (M1/M2/M3/M4)
chmod +x hourgit-darwin-arm64-*
sudo mv hourgit-darwin-arm64-* /usr/local/bin/hourgit

# Intel
chmod +x hourgit-darwin-amd64-*
sudo mv hourgit-darwin-amd64-* /usr/local/bin/hourgit
```

### Linux

```bash
# x86_64
chmod +x hourgit-linux-amd64-*
sudo mv hourgit-linux-amd64-* /usr/local/bin/hourgit

# ARM64
chmod +x hourgit-linux-arm64-*
sudo mv hourgit-linux-arm64-* /usr/local/bin/hourgit
```

### Windows

Move `hourgit-windows-amd64-*.exe` to a directory in your `PATH` and rename it to `hourgit.exe`.

### Verify

```bash
hourgit version
```
