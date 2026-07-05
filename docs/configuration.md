# Configuration

FocusMode uses two YAML files by default:

- `profile.yml`: Defines modes and which shortcuts each mode moves.
- `categories.yml`: Defines keyword rules used when listing desktop files or generating a profile.

You can pass custom paths with `-config` and `-categories`.

## Profile Configuration

Example `profile.yml`:

```yaml
modes:
  focusmode:
    destination: "FocusMode_Shortcuts"
    shortcuts:
      - "Steam.lnk"
      - "Epic Games.lnk"
    move_all: false

  gamemode:
    destination: "GameMode_Shortcuts"
    shortcuts:
      - "Visual Studio Code.lnk"
      - "Docker Desktop.lnk"
    move_all: false

default_mode: "focusmode"
```

### Fields

- `modes`: Map of mode names to mode settings.
- `destination`: Folder name created under your home directory.
- `shortcuts`: Shortcut or file names to move from the desktop.
- `move_all`: When `true`, moves every desktop shortcut found by FocusMode for that mode.
- `default_mode`: Mode used when `-mode` is not provided.

Mode names are user-defined. The examples use `focusmode` and `gamemode`, but you can add other modes.

## Categories Configuration

Example `categories.yml`:

```yaml
categories:
  game:
    name: "Games"
    icon: "[game]"
    keywords:
      - "steam"
      - "epic"
      - "game"

  development:
    name: "Development Tools"
    icon: "[dev]"
    keywords:
      - "code"
      - "docker"
      - "git"

  work:
    name: "Work/Productivity"
    icon: "[work]"
    keywords:
      - "office"
      - "word"
      - "excel"

category_order:
  - game
  - development
  - work
  - other
```

### How Categorization Works

- FocusMode compares each desktop file name to category keywords.
- Matching is case-insensitive.
- `category_order` controls priority; the first matching category wins.
- Files that do not match any configured category are treated as `other`.
- If `categories.yml` is missing, FocusMode uses built-in default categories.

## Generate a Starter Profile

You can generate `profile.yml` from your desktop:

```bash
./focusmode -auto-config
```

Review the generated file before running FocusMode without `-dry-run`.

