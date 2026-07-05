# System Tray

FocusMode can run as a persistent Windows system tray app:

```bash
focusmode.exe -tray
```

When tray mode is running, click the FocusMode icon in the notification area to open the menu.

## Tray Menu

The tray menu includes:

- One `Switch to ...` item for each mode in `profile.yml`.
- `Restore all`, which restores shortcuts from all configured mode folders.
- `Quit`, which exits the tray app.

Choosing a mode runs the same behavior as:

```bash
focusmode.exe -switch -mode focusmode
```

That means FocusMode restores currently hidden shortcuts first, then applies the selected mode.

## Start With Windows

To start FocusMode automatically:

1. Press `Win+R`.
2. Run `shell:startup`.
3. Add a shortcut to `focusmode.exe`.
4. Set the shortcut target to:

```text
"C:\path\to\focusmode.exe" -tray
```

5. Set **Start in** to the folder containing `profile.yml`.

## Custom Config

If your config lives somewhere else:

```text
"C:\path\to\focusmode.exe" -config "C:\path\to\profile.yml" -tray
```

## Platform Support

Tray mode is currently Windows-only. The regular command-line commands still work on Windows, macOS, and Linux.

