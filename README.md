# FocusMode

A simple Go tool to help you focus by organizing desktop shortcuts. Move shortcuts from your cluttered desktop to an organized folder.

## Features

- 📁 Move desktop shortcuts to an organized folder
- ⚙️ Configure shortcuts via YAML file
- 🎮 Multiple modes: FocusMode and GameMode (easily extensible)
- 🔄 Support for moving all shortcuts or specific ones
- 🖥️ Cross-platform (Windows, macOS, Linux)
- 🧪 Dry-run mode to preview changes

## Installation

1. Make sure you have Go installed (1.21 or later)
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Build the project:
   ```bash
   go build -o focusmode move.go
   ```

## Configuration

Edit `profile.yml` to configure which shortcuts to move for different modes:

```yaml
modes:
  focusmode:
    destination: "FocusMode_Shortcuts"  # Folder name in home directory
    shortcuts:
      - "MyShortcut.lnk"
      - "AnotherShortcut.lnk"
    move_all: false  # Set to true to move ALL shortcuts
  
  gamemode:
    destination: "GameMode_Shortcuts"
    shortcuts:
      - "Steam.lnk"
      - "Epic Games.lnk"
    move_all: false

default_mode: "focusmode"  # Default mode if not specified
```

## Usage

### Basic usage (uses default mode)
```bash
./focusmode
```

### Use a specific mode
```bash
./focusmode -mode focusmode
./focusmode -mode gamemode
```

### List available modes
```bash
./focusmode -list-modes
```

### With custom config file
```bash
./focusmode -config myconfig.yml
```

### Dry-run (preview without moving)
```bash
./focusmode -mode gamemode -dry-run
```

### Command-line options
- `-config`: Path to configuration file (default: `profile.yml`)
- `-mode`: Mode to use (focusmode, gamemode, etc.) - uses default if not specified
- `-dry-run`: Preview what would be moved without actually moving files
- `-list-modes`: List all available modes from configuration

## How it works

1. Reads configuration from `profile.yml`
2. Creates destination folder in your home directory (if it doesn't exist)
3. Moves specified shortcuts from desktop to the destination folder
4. Provides a summary of moved files

## Example

If you have shortcuts on your desktop:
- `Chrome.lnk`
- `VS Code.lnk`
- `Steam.lnk`
- `Epic Games.lnk`

And your `profile.yml` contains:
```yaml
modes:
  focusmode:
    destination: "FocusMode_Shortcuts"
    shortcuts:
      - "Chrome.lnk"
      - "VS Code.lnk"
    move_all: false
  
  gamemode:
    destination: "GameMode_Shortcuts"
    shortcuts:
      - "Steam.lnk"
      - "Epic Games.lnk"
    move_all: false

default_mode: "focusmode"
```

Running `./focusmode -mode focusmode` will:
- Create `~/FocusMode_Shortcuts` folder
- Move `Chrome.lnk` and `VS Code.lnk` to that folder
- Leave game shortcuts on the desktop

Running `./focusmode -mode gamemode` will:
- Create `~/GameMode_Shortcuts` folder
- Move `Steam.lnk` and `Epic Games.lnk` to that folder
- Leave work shortcuts on the desktop

## License

MIT


