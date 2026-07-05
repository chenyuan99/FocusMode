# AGENTS.md

## Project Overview

FocusMode is a small Go 1.21 command-line tool that organizes desktop shortcuts by moving them between the user's desktop and mode-specific folders. The main implementation is in `move.go`; tests live in `move_test.go`.

## Repository Layout

- `move.go`: CLI flags, YAML config loading, shortcut categorization, move/restore logic, and focus-session helpers.
- `move_test.go`: Unit tests for config parsing, categorization, file movement, restore behavior, and focus-session timing.
- `profile.yml`: Example/default mode configuration.
- `categories.yml`: Shortcut categorization keywords and display metadata.
- `.github/workflows/ci.yml`: CI test/build workflow.
- `.github/workflows/release.yml`: Tagged release packaging workflow.

## Build and Test Commands

Run these from the repository root:

```bash
go test ./...
go test -v ./...
go test -cover ./...
go build -o focusmode .
```

On Windows, the CI-style binary name is:

```bash
go build -v -o focusmode.exe .
```

Before submitting Go code changes, run:

```bash
gofmt -w *.go
go test ./...
```

## Coding Guidelines

- Follow standard Go style and keep the package as `main`.
- Prefer small, direct helper functions over broad abstractions; this codebase is intentionally compact.
- Preserve existing YAML field names and config compatibility unless the user explicitly asks for a breaking change.
- Keep CLI flag behavior stable and update `README.md` when adding or changing user-facing commands.
- Add or update tests in `move_test.go` for behavioral changes, especially file movement, restore behavior, config parsing, and categorization logic.

## Safety Notes

- Be careful with functions that move files on the user's real desktop. Prefer temp directories in tests and dry-run behavior when manually validating.
- Do not run commands that move or restore actual desktop shortcuts unless the user explicitly asks for that behavior.
- Avoid committing generated binaries, coverage files, release archives, or IDE metadata.
- The repo currently ignores `.idea/`; leave local IDE workspace changes alone unless the user specifically requests otherwise.

## Configuration Notes

- `profile.yml` controls modes such as `focusmode` and `gamemode`, their destination folders, explicit shortcut lists, and `move_all`.
- `categories.yml` controls keyword-based classification used by `-list-desktop` and `-auto-config`.
- If `categories.yml` is missing, the app falls back to default categories in code.

## Release Notes

Releases are triggered by tags matching `v*.*.*`. The release workflow builds platform binaries, packages `profile.yml`, `categories.yml`, and `README.md`, creates checksums, and publishes a GitHub release.
