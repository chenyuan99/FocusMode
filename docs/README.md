# FocusMode User Docs

FocusMode is a command-line tool for keeping your desktop focused. It moves selected desktop shortcuts into mode-specific folders, then can restore them later when you want your desktop back.

## Start Here

- [Getting Started](getting-started.md): Install, build, and run FocusMode for the first time.
- [Configuration](configuration.md): Set up `profile.yml` and `categories.yml`.
- [Command Reference](commands.md): See all supported commands and flags.
- [Taskbar Shortcuts](taskbar-shortcuts.md): Create one-click Windows shortcuts for switching modes.
- [Troubleshooting](troubleshooting.md): Fix common setup and usage problems.

## Common Workflows

Preview what FocusMode would move:

```bash
./focusmode -mode focusmode -dry-run
```

List your desktop files with suggested categories:

```bash
./focusmode -list-desktop
```

Generate a starter profile from your desktop:

```bash
./focusmode -auto-config
```

Restore shortcuts for a mode:

```bash
./focusmode -restore -mode focusmode
```

Switch directly to a mode:

```bash
./focusmode -switch -mode gamemode
```
