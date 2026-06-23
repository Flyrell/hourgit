# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Watcher daemon e2e tests running real `hourgit watch` inside Docker (basic start/stop, multiple sessions, graceful shutdown, crash recovery, multi-repo, gitignore filtering)
- `HOURGIT_IDLE_THRESHOLD` env var override for daemon idle threshold (enables short thresholds in e2e tests)

### Changed

- **BREAKING**: Idle threshold config changed from minutes to seconds (`idle_threshold_minutes` → `idle_threshold_seconds`, default: 600s)

## [0.8.2] - 2026-03-16

### Changed

- Bump GitHub Actions versions for Node.js 20 deprecation

## [0.8.1] - 2026-03-15

### Changed

- Use "(no task)" label for task-less log entries in reports
- Move log, edit, remove under `log` group command
- Move schedule commands under `project`/`defaults` groups
- Remove duplicate resolveProjectFromRepo in favor of ResolveProjectContext

## [0.8.0] - 2026-03-15

### Added

- Precise mode with filesystem watcher daemon for idle gap detection
- Project edit command (`project edit`) for renaming and mode changes
- `--idle-threshold` flag on project edit
- Commit tracking from git reflog
- PDF export with commit messages (`--detail full`)
- From/to time range display in report detail panel
- From/to time fields in edit, log, and report overlays
- Allow all three `--from`/`--to`/`--duration` flags in edit when consistent

### Changed

- Rename `--merge` to `--append` on init, add `-m` shorthand for `--mode`
- Centralize day budget computation
- Replace proportional scaling with targeted segment carving for log deduction

### Fixed

- Crash recovery hash collision and service manager bugs
- Status `--project` shows wrong branch when run from different repo
- Detail panel time range display

## [0.7.0] - 2026-03-02

### Added

- Flag shortcuts for common commands
- macOS binary code signing

## [0.6.0] - 2026-02-27

### Added

- `status` command showing current tracking state

### Fixed

- Hanging update check

## [0.5.0] - 2026-02-27

### Added

- Project removal (`project remove`)
- PDF report export (`report --export pdf`)

## [0.4.0] - 2026-02-26

### Added

- Checkout sync from git reflog (`sync` command)
- Update check and `update` command

### Fixed

- Commit deduplication logic

## [0.3.0] - 2026-02-25

### Added

- Website with documentation
- Installer script (`curl | bash`)

## [0.2.0] - 2026-02-25

### Added

- Interactive report with branch deduplication

## [0.1.0] - 2026-02-25

### Added

- Initial release: `init`, `log add`, `report`, `history` commands
- Project management and schedule configuration
- Shell completion generation

[Unreleased]: https://github.com/Flyrell/hourgit/compare/v0.8.2...HEAD
[0.8.2]: https://github.com/Flyrell/hourgit/compare/v0.8.1...v0.8.2
[0.8.1]: https://github.com/Flyrell/hourgit/compare/v0.8.0...v0.8.1
[0.8.0]: https://github.com/Flyrell/hourgit/compare/v0.7.3...v0.8.0
[0.7.0]: https://github.com/Flyrell/hourgit/compare/v0.6.1...v0.7.0
[0.6.0]: https://github.com/Flyrell/hourgit/compare/v0.5.3...v0.6.0
[0.5.0]: https://github.com/Flyrell/hourgit/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/Flyrell/hourgit/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/Flyrell/hourgit/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/Flyrell/hourgit/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Flyrell/hourgit/releases/tag/v0.1.0
