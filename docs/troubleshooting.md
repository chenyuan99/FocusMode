# Troubleshooting

## Nothing Moved

Try a dry run first:

```bash
./focusmode -mode focusmode -dry-run
```

Check that:

- The shortcut names in `profile.yml` exactly match the file names on your desktop.
- The selected mode exists in `profile.yml`.
- `move_all` is set the way you expect.
- You are running FocusMode on the user account that owns the desktop files.

## Wrong Mode Ran

If you do not pass `-mode`, FocusMode uses `default_mode` from `profile.yml`.

List available modes:

```bash
./focusmode -list-modes
```

Then run the intended mode explicitly:

```bash
./focusmode -mode gamemode
```

## Files Were Moved and I Want Them Back

Restore the mode that moved them:

```bash
./focusmode -restore -mode focusmode
```

If you are not sure which mode moved them, restore all configured modes:

```bash
./focusmode -restore-all
```

## Categories Look Incorrect

Run:

```bash
./focusmode -list-desktop
```

Then edit `categories.yml`:

- Add keywords for files that were missed.
- Move more specific categories earlier in `category_order`.
- Keep broad keywords later so they do not catch unrelated files.

## Profile Generation Produced Surprising Results

`-auto-config` uses keyword-based categorization, so it is a starting point rather than a final setup.

After running:

```bash
./focusmode -auto-config
```

Open `profile.yml`, review each mode, and run:

```bash
./focusmode -dry-run
```

## YAML Errors

YAML is indentation-sensitive. Use spaces, not tabs, and keep list items under the correct key:

```yaml
modes:
  focusmode:
    destination: "FocusMode_Shortcuts"
    shortcuts:
      - "Steam.lnk"
    move_all: false
```

## Build Fails

Make sure Go 1.21 or later is installed:

```bash
go version
```

Then refresh dependencies and build again:

```bash
go mod tidy
go build -o focusmode .
```

## Stats Look Wrong

Mode time is saved when FocusMode switches modes or restores shortcuts. If the computer is shut down while a mode is active, the next `-stats`, `-switch`, or restore command will include the elapsed time since that mode started.

The stats database is stored next to `profile.yml`:

```text
focusmode_stats.db
```

