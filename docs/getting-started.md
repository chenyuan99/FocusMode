# Getting Started

## Requirements

- Go 1.21 or later if building from source.
- A terminal or command prompt.
- A desktop folder with shortcuts or files you want FocusMode to organize.

## Build From Source

From the repository root:

```bash
go mod tidy
go build -o focusmode move.go
```

On Windows, you may prefer:

```bash
go build -o focusmode.exe move.go
```

## First Run

Start with a dry run so you can see what would happen without moving files:

```bash
./focusmode -dry-run
```

On Windows:

```bash
.\focusmode.exe -dry-run
```

By default, FocusMode reads `profile.yml` and uses the configured `default_mode`.

## Choose a Mode

Run a specific mode:

```bash
./focusmode -mode focusmode
./focusmode -mode gamemode
```

FocusMode will create the mode's destination folder under your home directory if it does not already exist, then move the configured shortcuts from your desktop into that folder.

## See Available Modes

```bash
./focusmode -list-modes
```

## Restore Shortcuts

Restore one mode:

```bash
./focusmode -restore -mode focusmode
```

Restore all configured modes:

```bash
./focusmode -restore-all
```

Preview restore actions first:

```bash
./focusmode -restore -mode focusmode -dry-run
```

## Switch Modes

Use `-switch` when you want one command that changes from the current mode to another:

```bash
./focusmode -switch -mode focusmode
./focusmode -switch -mode gamemode
```

This restores hidden shortcuts from all configured modes, then applies the selected mode.
