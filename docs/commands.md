# Command Reference

## Basic Usage

Run the default mode from `profile.yml`:

```bash
./focusmode
```

Run a specific mode:

```bash
./focusmode -mode focusmode
```

Preview changes without moving files:

```bash
./focusmode -mode focusmode -dry-run
```

Switch directly to a mode:

```bash
./focusmode -switch -mode gamemode
```

Start the Windows tray app:

```bash
./focusmode -tray
```

View tracked mode time:

```bash
./focusmode -stats
```

## Flags

| Flag | Default | Description |
| --- | --- | --- |
| `-config` | `profile.yml` | Path to the profile configuration file. |
| `-categories` | `categories.yml` | Path to the categories configuration file. |
| `-mode` | value of `default_mode` | Mode to run or restore. |
| `-dry-run` | `false` | Show what would be moved or restored without changing files. |
| `-list-modes` | `false` | List available modes from the profile configuration. |
| `-list-desktop` | `false` | List desktop files grouped by category with suggested modes. |
| `-auto-config` | `false` | Generate `profile.yml` from desktop files and categories. |
| `-restore` | `false` | Restore shortcuts from one mode's destination folder to the desktop. |
| `-restore-all` | `false` | Restore shortcuts from all configured mode folders to the desktop. |
| `-switch` | `false` | Restore all shortcuts, then apply the selected mode. |
| `-tray` | `false` | Run as a Windows system tray app. |
| `-stats` | `false` | Show tracked mode usage totals. |

## Listing Modes

```bash
./focusmode -list-modes
```

Use this when you are not sure which modes are available in `profile.yml`.

## Listing Desktop Files

```bash
./focusmode -list-desktop
```

This scans your desktop, groups files by category, and suggests which mode should move each item.

## Auto-Generating Configuration

```bash
./focusmode -auto-config
```

This creates or overwrites `profile.yml` based on desktop files and category rules. Review the file before using it to move files.

## Restoring Files

Restore one mode:

```bash
./focusmode -restore -mode gamemode
```

Restore every configured mode:

```bash
./focusmode -restore-all
```

Preview a restore:

```bash
./focusmode -restore-all -dry-run
```

## Switching Modes

```bash
./focusmode -switch -mode focusmode
./focusmode -switch -mode gamemode
```

Use this for taskbar shortcuts. It returns currently hidden shortcuts from all configured mode folders, then moves the shortcuts for the selected mode.

Preview a switch:

```bash
./focusmode -switch -mode gamemode -dry-run
```

## System Tray

```bash
./focusmode -tray
```

On Windows, this starts a persistent notification-area icon. Click the icon to open a menu with one item per configured mode, plus `Restore all` and `Quit`.

## Mode Stats

```bash
./focusmode -stats
```

FocusMode tracks mode time in `focusmode_stats.db` next to `profile.yml`. A mode becomes active when you run it directly or switch to it. The active timer is saved into totals when you switch modes or restore shortcuts.

## Custom Configuration Paths

```bash
./focusmode -config my-profile.yml -categories my-categories.yml -mode focusmode
```
