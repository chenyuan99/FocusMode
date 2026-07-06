# Taskbar Shortcuts

FocusMode can switch modes with one command:

```bash
focusmode.exe -switch -mode focusmode
```

The `-switch` flag restores shortcuts from all configured mode folders, then applies the selected mode. This makes it a good target for Windows taskbar shortcuts.

If you prefer a persistent notification-area icon with a menu, use [System Tray](system-tray.md) instead.

## Create a Shortcut Per Mode

1. Build or download `focusmode.exe`.
2. Right-click `focusmode.exe` and choose **Create shortcut**.
3. Rename the shortcut to the mode name, such as `FocusMode` or `GameMode`.
4. Open the shortcut properties.
5. Set **Target** to one of these commands:

```text
"C:\path\to\focusmode.exe" -switch -mode focusmode
```

```text
"C:\path\to\focusmode.exe" -switch -mode gamemode
```

6. Set **Start in** to the folder that contains `profile.yml`, for example:

```text
C:\Users\you\FocusMode
```

7. Right-click the shortcut and pin it to the taskbar.

## Test First

Before pinning, run a dry run from the same folder:

```bash
focusmode.exe -switch -mode focusmode -dry-run
```

If the preview looks right, remove `-dry-run` from the shortcut target.

## Use a Custom Config

If your config file is somewhere else, include `-config`:

```text
"C:\path\to\focusmode.exe" -config "C:\path\to\profile.yml" -switch -mode focusmode
```

