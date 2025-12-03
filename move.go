package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// ModeConfig represents the configuration for a specific mode
type ModeConfig struct {
	Destination string   `yaml:"destination"`
	Shortcuts   []string `yaml:"shortcuts"`
	MoveAll     bool     `yaml:"move_all"`
}

// Config represents the YAML configuration structure
type Config struct {
	Modes       map[string]ModeConfig `yaml:"modes"`
	DefaultMode string                `yaml:"default_mode"`
}

// getDesktopPath returns the desktop path for the current operating system
func getDesktopPath() (string, error) {
	switch runtime.GOOS {
	case "windows":
		desktopPath := filepath.Join(os.Getenv("USERPROFILE"), "Desktop")
		return desktopPath, nil
	case "darwin":
		// On macOS, the desktop path is typically ~/Desktop.
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, "Desktop"), nil
	case "linux":
		// On Linux, it can vary, but a common location is ~/Desktop.
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, "Desktop"), nil
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// moveDesktopShortcut moves a shortcut from desktop to destination directory
func moveDesktopShortcut(shortcutName string, destinationDir string) error {
	desktopPath, err := getDesktopPath()
	if err != nil {
		return fmt.Errorf("error getting desktop path: %w", err)
	}

	oldPath := filepath.Join(desktopPath, shortcutName)
	newPath := filepath.Join(destinationDir, shortcutName)

	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return fmt.Errorf("shortcut '%s' not found on desktop", shortcutName)
	}

	err = os.Rename(oldPath, newPath)
	if err != nil {
		return fmt.Errorf("error moving shortcut: %w", err)
	}
	return nil
}

// loadConfig loads the configuration from profile.yml
func loadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing YAML: %w", err)
	}

	// Set default mode if not specified
	if config.DefaultMode == "" {
		config.DefaultMode = "focusmode"
	}

	return &config, nil
}

// getModeConfig returns the configuration for a specific mode
func (c *Config) getModeConfig(modeName string) (*ModeConfig, error) {
	modeConfig, exists := c.Modes[modeName]
	if !exists {
		return nil, fmt.Errorf("mode '%s' not found in configuration. Available modes: %v", modeName, c.getAvailableModes())
	}

	// Set default destination if not specified
	if modeConfig.Destination == "" {
		modeConfig.Destination = fmt.Sprintf("%s_Shortcuts", modeName)
	}

	return &modeConfig, nil
}

// getAvailableModes returns a list of available mode names
func (c *Config) getAvailableModes() []string {
	modes := make([]string, 0, len(c.Modes))
	for mode := range c.Modes {
		modes = append(modes, mode)
	}
	return modes
}

// getAllDesktopShortcuts returns all files on the desktop
func getAllDesktopShortcuts() ([]string, error) {
	desktopPath, err := getDesktopPath()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(desktopPath)
	if err != nil {
		return nil, fmt.Errorf("error reading desktop directory: %w", err)
	}

	var shortcuts []string
	for _, entry := range entries {
		if !entry.IsDir() {
			shortcuts = append(shortcuts, entry.Name())
		}
	}

	return shortcuts, nil
}

func main() {
	// Command-line flags
	configPath := flag.String("config", "profile.yml", "Path to configuration file")
	mode := flag.String("mode", "", "Mode to use (focusmode, gamemode, etc.)")
	dryRun := flag.Bool("dry-run", false, "Show what would be moved without actually moving")
	listModes := flag.Bool("list-modes", false, "List all available modes")
	flag.Parse()

	// Load configuration
	config, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// List modes if requested
	if *listModes {
		fmt.Println("Available modes:")
		for modeName := range config.Modes {
			if modeName == config.DefaultMode {
				fmt.Printf("  %s (default)\n", modeName)
			} else {
				fmt.Printf("  %s\n", modeName)
			}
		}
		return
	}

	// Determine which mode to use
	modeName := *mode
	if modeName == "" {
		modeName = config.DefaultMode
	}

	// Get mode-specific configuration
	modeConfig, err := config.getModeConfig(modeName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Use -list-modes to see available modes\n")
		os.Exit(1)
	}

	fmt.Printf("Using mode: %s\n", modeName)

	// Get destination folder
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	destinationFolder := filepath.Join(homeDir, modeConfig.Destination)

	// Create the destination folder if it doesn't exist
	if !*dryRun {
		if _, err := os.Stat(destinationFolder); os.IsNotExist(err) {
			err := os.MkdirAll(destinationFolder, 0755)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating destination folder: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Created destination folder: %s\n", destinationFolder)
		}
	}

	// Determine which shortcuts to move
	var shortcutsToMove []string

	if modeConfig.MoveAll {
		// Get all shortcuts from desktop
		allShortcuts, err := getAllDesktopShortcuts()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting desktop shortcuts: %v\n", err)
			os.Exit(1)
		}
		shortcutsToMove = allShortcuts
		fmt.Printf("Moving ALL shortcuts from desktop (%d found)\n", len(shortcutsToMove))
	} else {
		shortcutsToMove = modeConfig.Shortcuts
		fmt.Printf("Moving specified shortcuts (%d configured)\n", len(shortcutsToMove))
	}

	// Move shortcuts
	successCount := 0
	failCount := 0

	for _, shortcutName := range shortcutsToMove {
		if *dryRun {
			fmt.Printf("[DRY RUN] Would move: %s -> %s\n", shortcutName, destinationFolder)
			successCount++
		} else {
			err := moveDesktopShortcut(shortcutName, destinationFolder)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error moving '%s': %v\n", shortcutName, err)
				failCount++
			} else {
				fmt.Printf("✓ Moved: %s\n", shortcutName)
				successCount++
			}
		}
	}

	// Summary
	fmt.Println("\n--- Summary ---")
	fmt.Printf("Mode: %s\n", modeName)
	fmt.Printf("Successfully moved: %d\n", successCount)
	if failCount > 0 {
		fmt.Printf("Failed: %d\n", failCount)
	}
	if *dryRun {
		fmt.Println("(Dry run - no files were actually moved)")
	} else {
		fmt.Printf("All shortcuts moved to: %s\n", destinationFolder)
	}
}
