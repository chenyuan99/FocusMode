# FocusMode

[![CI](https://github.com/chenyuan99/FocusMode/actions/workflows/ci.yml/badge.svg)](https://github.com/chenyuan99/FocusMode/actions/workflows/ci.yml)
[![Release](https://github.com/chenyuan99/FocusMode/actions/workflows/release.yml/badge.svg)](https://github.com/chenyuan99/FocusMode/actions/workflows/release.yml)

A simple Go tool to help you focus by organizing desktop shortcuts. Move shortcuts from your cluttered desktop to an organized folder.

## Features

- 📁 Move desktop shortcuts to an organized folder
- ⚙️ Configure shortcuts via YAML file
- 🎮 Multiple modes: FocusMode and GameMode (easily extensible)
- 🔄 Support for moving all shortcuts or specific ones
- 🖥️ Cross-platform (Windows, macOS, Linux)
- 🧪 Dry-run mode to preview changes

## Installation

### From Releases (Recommended)

Download the latest release from the [Releases page](https://github.com/chenyuan99/FocusMode/releases):
- **Windows**: `focusmode-windows-amd64.zip`

Extract the archive and run `focusmode-windows-amd64.exe`.

### From Source

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

### Profile Configuration (`profile.yml`)

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

### Categories Configuration (`categories.yml`)

The `categories.yml` file defines keywords used to automatically categorize shortcuts when using `-list-desktop`. This helps identify which shortcuts are games, development tools, work applications, etc.

You can customize the keywords and categories to match your needs. The file structure:

```yaml
categories:
  game:
    name: "Games"
    icon: "🎮"
    keywords:
      - "steam"
      - "epic"
      - "game"
      # ... more keywords

  development:
    name: "Development Tools"
    icon: "💻"
    keywords:
      - "code"
      - "docker"
      - "git"
      # ... more keywords

  work:
    name: "Work/Productivity"
    icon: "💼"
    keywords:
      - "office"
      - "word"
      - "excel"
      # ... more keywords

category_order:
  - game
  - development
  - work
  - other
```

**Note:** If `categories.yml` doesn't exist, the tool will use default categories. You can create your own to customize the categorization logic.

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

### List desktop files
```bash
./focusmode -list-desktop
```
This command shows all files on your desktop, grouped by category, with suggested modes for each shortcut. This is helpful when configuring which shortcuts to move.

### Auto-generate profile
```bash
./focusmode -auto-config
```
This command automatically generates `profile.yml` based on your desktop shortcuts:
- **Games** → `focusmode` (moved when focusing to remove distractions)
- **Work/Development tools** → `gamemode` (moved when gaming to remove work distractions)
- **Other** → `focusmode` (moved when focusing)

**Logic:**
- **FocusMode**: Moves games away → keeps work tools on desktop for quick access
- **GameMode**: Moves work tools away → keeps games on desktop for quick access

The generated profile can be reviewed and customized as needed.

### Restore shortcuts to desktop
```bash
# Restore shortcuts from a specific mode
./focusmode -restore -mode focusmode

# Restore shortcuts from all modes
./focusmode -restore-all

# Preview what would be restored (dry-run)
./focusmode -restore -mode gamemode -dry-run
```
This command moves shortcuts back from organized folders to your desktop. Useful when you want to restore your desktop to its original state.

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
- `-categories`: Path to categories configuration file (default: `categories.yml`)
- `-mode`: Mode to use (focusmode, gamemode, etc.) - uses default if not specified
- `-dry-run`: Preview what would be moved/restored without actually moving files
- `-list-modes`: List all available modes from configuration
- `-list-desktop`: List all files on your desktop with suggested modes (useful for configuring shortcuts)
- `-auto-config`: Auto-generate `profile.yml` based on desktop shortcuts and categories
- `-restore`: Restore shortcuts from a specific mode's folder back to desktop
- `-restore-all`: Restore shortcuts from all modes back to desktop

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

## Releases

Releases are automatically built and published when you push a tag matching `v*.*.*` (e.g., `v1.0.0`).

### Creating a Release

1. **Using Git tags** (recommended):
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **Using GitHub Actions**:
   - Go to Actions → Release workflow
   - Click "Run workflow"
   - Enter the version tag (e.g., `v1.0.0`)

The workflow will automatically:
- Build binaries for all platforms (Windows, macOS, Linux - AMD64 and ARM64)
- Create release archives (`.tar.gz` for Unix, `.zip` for Windows)
- Generate checksums for verification
- Create a GitHub release with all artifacts

### Verifying Releases

Download the `checksums.txt` file and verify:
```bash
sha256sum -c checksums.txt
```

## Development

### Running Tests
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Run a specific test
go test -v -run TestGetDesktopPath
```

The test suite includes:
- ✅ Configuration loading and parsing
- ✅ Mode configuration retrieval
- ✅ Desktop path detection (cross-platform)
- ✅ File moving operations
- ✅ Desktop shortcut listing
- ✅ Error handling and edge cases

### Building Locally
```bash
# Build for current platform
go build -o focusmode move.go

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -o focusmode-linux-amd64 move.go
```

## License

MIT


