package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
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

// SessionState represents the state of a focus session
type SessionState int

const (
	StateRunning SessionState = iota
	StatePaused
	StateCompleted
	StateInterrupted
)

// FocusSession represents a timed focus session
type FocusSession struct {
	Duration       time.Duration // Total session duration
	Mode           string        // Mode to apply (focusmode/gamemode)
	StartTime      time.Time     // When session started
	PausedAt       *time.Time    // When session was paused (nil if not paused)
	PausedTotal    time.Duration // Total time spent paused
	AutoRestore    bool          // Whether to auto-restore on completion
	Config         *Config       // Reference to loaded config
	State          SessionState  // Current state of the session
	MovedShortcuts []string      // List of shortcuts that were moved during session start
}

// elapsed returns the time elapsed since the session started, excluding paused time
func (fs *FocusSession) elapsed() time.Duration {
	if fs.State == StatePaused && fs.PausedAt != nil {
		// If currently paused, calculate elapsed up to pause point
		return fs.PausedAt.Sub(fs.StartTime) - fs.PausedTotal
	}
	// If running or completed, calculate elapsed up to now
	return time.Since(fs.StartTime) - fs.PausedTotal
}

// remaining returns the time remaining in the session
func (fs *FocusSession) remaining() time.Duration {
	elapsed := fs.elapsed()
	remaining := fs.Duration - elapsed
	if remaining < 0 {
		return 0
	}
	return remaining
}

// startFocusSession creates and initializes a new focus session with validation
func startFocusSession(config *Config, modeName string, duration int, autoRestore bool) (*FocusSession, error) {
	// Validate duration is positive
	if duration <= 0 {
		return nil, fmt.Errorf("duration must be positive, got: %d minutes", duration)
	}

	// Validate mode exists in configuration
	_, err := config.getModeConfig(modeName)
	if err != nil {
		availableModes := config.getAvailableModes()
		return nil, fmt.Errorf("invalid mode '%s'. Available modes: %v", modeName, availableModes)
	}

	// Initialize FocusSession struct with validated inputs
	session := &FocusSession{
		Duration:    time.Duration(duration) * time.Minute,
		Mode:        modeName,
		StartTime:   time.Now(),
		PausedAt:    nil,
		PausedTotal: 0,
		AutoRestore: autoRestore,
		Config:      config,
		State:       StateRunning,
	}

	return session, nil
}

// organizeShortcuts moves shortcuts according to the session's mode configuration
// Returns the list of successfully moved shortcuts and any error encountered
func (fs *FocusSession) organizeShortcuts() ([]string, error) {
	// Get mode configuration
	modeConfig, err := fs.Config.getModeConfig(fs.Mode)
	if err != nil {
		return nil, fmt.Errorf("error getting mode configuration: %w", err)
	}

	// Get destination folder
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("error getting home directory: %w", err)
	}

	destinationFolder := filepath.Join(homeDir, modeConfig.Destination)

	// Create the destination folder if it doesn't exist
	if _, err := os.Stat(destinationFolder); os.IsNotExist(err) {
		err := os.MkdirAll(destinationFolder, 0755)
		if err != nil {
			return nil, fmt.Errorf("error creating destination folder: %w", err)
		}
		fmt.Printf("Created destination folder: %s\n", destinationFolder)
	}

	// Determine which shortcuts to move
	var shortcutsToMove []string

	if modeConfig.MoveAll {
		// Get all shortcuts from desktop
		allShortcuts, err := getAllDesktopShortcuts()
		if err != nil {
			return nil, fmt.Errorf("error getting desktop shortcuts: %w", err)
		}
		shortcutsToMove = allShortcuts
		fmt.Printf("Moving ALL shortcuts from desktop (%d found)\n", len(shortcutsToMove))
	} else {
		shortcutsToMove = modeConfig.Shortcuts
		fmt.Printf("Moving specified shortcuts (%d configured)\n", len(shortcutsToMove))
	}

	// Move shortcuts and track successful moves
	var movedShortcuts []string
	successCount := 0
	failCount := 0

	for _, shortcutName := range shortcutsToMove {
		err := moveDesktopShortcut(shortcutName, destinationFolder)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error moving '%s': %v\n", shortcutName, err)
			failCount++
		} else {
			fmt.Printf("✓ Moved: %s\n", shortcutName)
			movedShortcuts = append(movedShortcuts, shortcutName)
			successCount++
		}
	}

	// Display summary
	fmt.Println("\n--- Organization Summary ---")
	fmt.Printf("Mode: %s\n", fs.Mode)
	fmt.Printf("Successfully moved: %d\n", successCount)
	if failCount > 0 {
		fmt.Printf("Failed: %d\n", failCount)
	}
	fmt.Printf("Shortcuts moved to: %s\n\n", destinationFolder)

	// Return the list of moved shortcuts even if some failed
	// This allows partial restoration if needed
	return movedShortcuts, nil
}

// formatDuration formats a duration into a human-readable string
// Returns format like "25m 30s", "1h 5m 30s", "45s", or "0s"
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	// Get total seconds
	totalSeconds := int(d.Seconds())

	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	// Build the formatted string
	var parts []string

	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}

	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}

	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	return strings.Join(parts, " ")
}

func getStatsPath(configPath string) string {
	if configPath == "" {
		configPath = "profile.yml"
	}
	return filepath.Join(filepath.Dir(configPath), "focusmode_stats.db")
}

func openStatsDB(statsPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(statsPath), 0755); err != nil {
		return nil, fmt.Errorf("error creating stats directory: %w", err)
	}

	db, err := sql.Open("sqlite", statsPath)
	if err != nil {
		return nil, fmt.Errorf("error opening stats database: %w", err)
	}

	if err := initStatsDB(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func initStatsDB(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS mode_totals (
			mode TEXT PRIMARY KEY,
			total_seconds INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS active_mode (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			mode TEXT NOT NULL,
			active_since TEXT NOT NULL
		)`,
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("error initializing stats database: %w", err)
		}
	}

	return nil
}

func stopActiveModeTx(tx *sql.Tx, now time.Time) error {
	var modeName string
	var activeSinceText string
	err := tx.QueryRow(`SELECT mode, active_since FROM active_mode WHERE id = 1`).Scan(&modeName, &activeSinceText)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("error reading active mode: %w", err)
	}

	activeSince, err := time.Parse(time.RFC3339Nano, activeSinceText)
	if err != nil {
		return fmt.Errorf("error parsing active mode timestamp: %w", err)
	}

	elapsed := now.Sub(activeSince)
	if elapsed > 0 {
		if _, err := tx.Exec(
			`INSERT INTO mode_totals (mode, total_seconds)
			 VALUES (?, ?)
			 ON CONFLICT(mode) DO UPDATE SET total_seconds = total_seconds + excluded.total_seconds`,
			modeName,
			int64(elapsed.Seconds()),
		); err != nil {
			return fmt.Errorf("error updating mode total: %w", err)
		}
	}

	if _, err := tx.Exec(`DELETE FROM active_mode WHERE id = 1`); err != nil {
		return fmt.Errorf("error clearing active mode: %w", err)
	}

	return nil
}

func startModeTracking(statsPath string, modeName string, dryRun bool) {
	if dryRun {
		fmt.Printf("[DRY RUN] Would start tracking mode time for: %s\n", modeName)
		return
	}

	db, err := openStatsDB(statsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		return
	}
	defer db.Close()

	if err := startModeTrackingAt(db, modeName, time.Now()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}
}

func startModeTrackingAt(db *sql.DB, modeName string, now time.Time) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error starting stats transaction: %w", err)
	}
	defer tx.Rollback()

	if err := stopActiveModeTx(tx, now); err != nil {
		return err
	}

	if _, err := tx.Exec(
		`INSERT INTO active_mode (id, mode, active_since)
		 VALUES (1, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET mode = excluded.mode, active_since = excluded.active_since`,
		modeName,
		now.Format(time.RFC3339Nano),
	); err != nil {
		return fmt.Errorf("error setting active mode: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error saving stats transaction: %w", err)
	}

	return nil
}

func stopModeTracking(statsPath string, dryRun bool) {
	if dryRun {
		fmt.Println("[DRY RUN] Would stop tracking active mode time")
		return
	}

	db, err := openStatsDB(statsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		return
	}
	defer db.Close()

	if err := stopModeTrackingAt(db, time.Now()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}
}

func stopModeTrackingAt(db *sql.DB, now time.Time) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error starting stats transaction: %w", err)
	}
	defer tx.Rollback()

	if err := stopActiveModeTx(tx, now); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error saving stats transaction: %w", err)
	}

	return nil
}

func readModeStats(db *sql.DB, now time.Time) (map[string]int64, string, error) {
	totals := make(map[string]int64)

	rows, err := db.Query(`SELECT mode, total_seconds FROM mode_totals`)
	if err != nil {
		return nil, "", fmt.Errorf("error reading mode totals: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var modeName string
		var seconds int64
		if err := rows.Scan(&modeName, &seconds); err != nil {
			return nil, "", fmt.Errorf("error scanning mode total: %w", err)
		}
		totals[modeName] = seconds
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("error reading mode totals: %w", err)
	}

	var activeMode string
	var activeSinceText string
	err = db.QueryRow(`SELECT mode, active_since FROM active_mode WHERE id = 1`).Scan(&activeMode, &activeSinceText)
	if err != nil {
		if err == sql.ErrNoRows {
			return totals, "", nil
		}
		return nil, "", fmt.Errorf("error reading active mode: %w", err)
	}

	activeSince, err := time.Parse(time.RFC3339Nano, activeSinceText)
	if err != nil {
		return nil, "", fmt.Errorf("error parsing active mode timestamp: %w", err)
	}

	elapsed := now.Sub(activeSince)
	if elapsed > 0 {
		totals[activeMode] += int64(elapsed.Seconds())
	}

	return totals, activeMode, nil
}

func displayModeStats(config *Config, statsPath string) {
	db, err := openStatsDB(statsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading stats: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	totals, activeMode, err := readModeStats(db, time.Now())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading stats: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Mode usage:")
	modes := config.getAvailableModes()
	sort.Strings(modes)
	for _, modeName := range modes {
		total := time.Duration(totals[modeName]) * time.Second
		activeMarker := ""
		if modeName == activeMode {
			activeMarker = " (active)"
		}
		fmt.Printf("  %s: %s%s\n", modeName, formatDuration(total), activeMarker)
	}
	fmt.Printf("\nStats file: %s\n", statsPath)
}

// displayProgress displays the current progress of a focus session
// Uses carriage return to update the same line
// Shows emoji indicators: ⏳ for running, ⏸ for paused
func displayProgress(elapsed, remaining time.Duration, paused bool) {
	var emoji string
	var status string

	if paused {
		emoji = "⏸"
		status = "Paused"
	} else {
		emoji = "⏳"
		status = "Focus Session"
	}

	elapsedStr := formatDuration(elapsed)
	remainingStr := formatDuration(remaining)

	// Use carriage return to overwrite the same line
	fmt.Printf("\r%s %s: %s elapsed | %s remaining", emoji, status, elapsedStr, remainingStr)
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
	return moveDesktopShortcutFromPath(shortcutName, destinationDir, "")
}

// moveDesktopShortcutFromPath moves a shortcut from a specific desktop path to destination directory
// If desktopPath is empty, it uses getDesktopPath()
func moveDesktopShortcutFromPath(shortcutName string, destinationDir string, desktopPath string) error {
	var err error
	if desktopPath == "" {
		desktopPath, err = getDesktopPath()
		if err != nil {
			return fmt.Errorf("error getting desktop path: %w", err)
		}
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

// restoreShortcutToDesktop moves a shortcut from destination directory back to desktop
func restoreShortcutToDesktop(shortcutName string, sourceDir string) error {
	desktopPath, err := getDesktopPath()
	if err != nil {
		return fmt.Errorf("error getting desktop path: %w", err)
	}

	sourcePath := filepath.Join(sourceDir, shortcutName)
	destPath := filepath.Join(desktopPath, shortcutName)

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("shortcut '%s' not found in source directory", shortcutName)
	}

	// Check if file already exists on desktop
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("shortcut '%s' already exists on desktop", shortcutName)
	}

	err = os.Rename(sourcePath, destPath)
	if err != nil {
		return fmt.Errorf("error restoring shortcut: %w", err)
	}
	return nil
}

// getShortcutsInFolder returns all files in a given folder
func getShortcutsInFolder(folderPath string) ([]string, error) {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, fmt.Errorf("error reading folder: %w", err)
	}

	var shortcuts []string
	for _, entry := range entries {
		if !entry.IsDir() {
			shortcuts = append(shortcuts, entry.Name())
		}
	}

	return shortcuts, nil
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
	return getAllDesktopShortcutsFromPath("")
}

// getAllDesktopShortcutsFromPath returns all files from a specific desktop path
// If desktopPath is empty, it uses getDesktopPath()
func getAllDesktopShortcutsFromPath(desktopPath string) ([]string, error) {
	var err error
	if desktopPath == "" {
		desktopPath, err = getDesktopPath()
		if err != nil {
			return nil, err
		}
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

// ShortcutCategory represents the category of a shortcut
type ShortcutCategory string

const (
	CategoryOther ShortcutCategory = "other"
)

// CategoryConfig represents the configuration for a category
type CategoryConfig struct {
	Name     string   `yaml:"name"`
	Icon     string   `yaml:"icon"`
	Keywords []string `yaml:"keywords"`
}

// CategoriesConfig represents the categories configuration structure
type CategoriesConfig struct {
	Categories    map[string]CategoryConfig `yaml:"categories"`
	CategoryOrder []string                  `yaml:"category_order"`
}

// loadCategoriesConfig loads the categories configuration from categories.yml
func loadCategoriesConfig(configPath string) (*CategoriesConfig, error) {
	// Default path if not specified
	if configPath == "" {
		configPath = "categories.yml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		// Return default categories if file doesn't exist
		return getDefaultCategoriesConfig(), nil
	}

	var config CategoriesConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing categories YAML: %w", err)
	}

	// Ensure category_order is set
	if len(config.CategoryOrder) == 0 {
		config.CategoryOrder = []string{"game", "development", "work", "other"}
	}

	return &config, nil
}

// getDefaultCategoriesConfig returns a default categories configuration
func getDefaultCategoriesConfig() *CategoriesConfig {
	return &CategoriesConfig{
		Categories: map[string]CategoryConfig{
			"game": {
				Name:     "Games",
				Icon:     "🎮",
				Keywords: []string{"game", "steam", "epic"},
			},
			"development": {
				Name:     "Development Tools",
				Icon:     "💻",
				Keywords: []string{"code", "docker", "git"},
			},
			"work": {
				Name:     "Work/Productivity",
				Icon:     "💼",
				Keywords: []string{"office", "word", "excel"},
			},
		},
		CategoryOrder: []string{"game", "development", "work", "other"},
	}
}

// categorizeShortcut attempts to categorize a shortcut based on its name using the config
func categorizeShortcut(name string, categoriesConfig *CategoriesConfig) ShortcutCategory {
	nameLower := strings.ToLower(name)

	// Check categories in order (first match wins)
	for _, categoryID := range categoriesConfig.CategoryOrder {
		if categoryID == "other" {
			continue // Skip "other" in the loop
		}

		category, exists := categoriesConfig.Categories[categoryID]
		if !exists {
			continue
		}

		// Check if any keyword matches
		for _, keyword := range category.Keywords {
			if strings.Contains(nameLower, strings.ToLower(keyword)) {
				return ShortcutCategory(categoryID)
			}
		}
	}

	return CategoryOther
}

// listDesktopFiles lists all files on the desktop with their types and categories
func listDesktopFiles() {
	categoriesConfig, err := loadCategoriesConfig("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Error loading categories config: %v\n", err)
		fmt.Fprintf(os.Stderr, "Using default categories.\n\n")
		categoriesConfig = getDefaultCategoriesConfig()
	}
	listDesktopFilesWithConfig(categoriesConfig)
}

// listDesktopFilesWithConfig lists all files on the desktop using the provided categories config
func listDesktopFilesWithConfig(categoriesConfig *CategoriesConfig) {

	desktopPath, err := getDesktopPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting desktop path: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Desktop path: %s\n\n", desktopPath)

	shortcuts, err := getAllDesktopShortcuts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading desktop: %v\n", err)
		os.Exit(1)
	}

	if len(shortcuts) == 0 {
		fmt.Println("No files found on desktop.")
		return
	}

	fmt.Printf("Found %d file(s) on desktop:\n\n", len(shortcuts))

	// Categorize shortcuts
	categorized := make(map[ShortcutCategory][]string)
	for _, shortcut := range shortcuts {
		category := categorizeShortcut(shortcut, categoriesConfig)
		categorized[category] = append(categorized[category], shortcut)
	}

	// Print categorized shortcuts in order
	for _, categoryID := range categoriesConfig.CategoryOrder {
		category := ShortcutCategory(categoryID)
		files, ok := categorized[category]
		if !ok || len(files) == 0 {
			continue
		}

		// Get category display info
		var label string
		var icon string
		if categoryID == "other" {
			label = "Other"
			icon = "📁"
		} else {
			if catConfig, exists := categoriesConfig.Categories[categoryID]; exists {
				label = catConfig.Name
				icon = catConfig.Icon
			} else {
				label = categoryID
				icon = "📁"
			}
		}

		fmt.Printf("%s %s (%d):\n", icon, label, len(files))
		for i, file := range files {
			// Show file type indicator
			ext := filepath.Ext(file)
			typeIndicator := ""
			if ext == ".lnk" {
				typeIndicator = " [Shortcut]"
			} else if ext == ".url" {
				typeIndicator = " [URL]"
			} else if ext != "" {
				typeIndicator = fmt.Sprintf(" [%s]", ext)
			}

			// Show suggested mode (which mode will move this shortcut)
			fileCategory := categorizeShortcut(file, categoriesConfig)
			suggestedMode := getModeForCategory(fileCategory)
			modeIndicator := ""
			if suggestedMode == "gamemode" {
				modeIndicator = " → 🎮 GameMode (moves work tools)"
			} else {
				modeIndicator = " → 💼 FocusMode (moves games/distractions)"
			}

			fmt.Printf("  %d. %s%s%s\n", i+1, file, typeIndicator, modeIndicator)
		}
		fmt.Println()
	}

	// Summary by category
	fmt.Println("--- Summary by Category ---")
	for _, categoryID := range categoriesConfig.CategoryOrder {
		category := ShortcutCategory(categoryID)
		if files, ok := categorized[category]; ok && len(files) > 0 {
			var label string
			var icon string
			if categoryID == "other" {
				label = "Other"
				icon = "📁"
			} else {
				if catConfig, exists := categoriesConfig.Categories[categoryID]; exists {
					label = catConfig.Name
					icon = catConfig.Icon
				} else {
					label = categoryID
					icon = "📁"
				}
			}
			fmt.Printf("%s %s: %d\n", icon, label, len(files))
		}
	}
	fmt.Printf("\nTotal: %d file(s)\n", len(shortcuts))
}

// getModeForCategory maps a category to a mode name
// This determines which mode should MOVE this category (to hide it)
func getModeForCategory(category ShortcutCategory) string {
	switch category {
	case ShortcutCategory("game"):
		// Games should be moved in focusmode (to remove distractions when working)
		return "focusmode"
	case ShortcutCategory("development"), ShortcutCategory("work"):
		// Work/development tools should be moved in gamemode (to remove work distractions when gaming)
		return "gamemode"
	default:
		// Other items can go to focusmode by default
		return "focusmode"
	}
}

// generateProfileFromDesktop generates a profile.yml based on desktop shortcuts and categories
func generateProfileFromDesktop(configPath string, categoriesPath string) {
	fmt.Println("Generating profile.yml from desktop shortcuts...")

	// Get desktop shortcuts
	shortcuts, err := getAllDesktopShortcuts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading desktop: %v\n", err)
		os.Exit(1)
	}

	if len(shortcuts) == 0 {
		fmt.Println("No shortcuts found on desktop.")
		return
	}

	// Load categories config to categorize shortcuts
	categoriesConfig, err := loadCategoriesConfig(categoriesPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Error loading categories config: %v\n", err)
		fmt.Fprintf(os.Stderr, "Using default categories.\n\n")
		categoriesConfig = getDefaultCategoriesConfig()
	}

	// Categorize shortcuts and group by which mode should move them
	focusmodeShortcuts := []string{} // Games and other distractions (moved in focusmode)
	gamemodeShortcuts := []string{}  // Work/development tools (moved in gamemode)

	for _, shortcut := range shortcuts {
		category := categorizeShortcut(shortcut, categoriesConfig)
		modeName := getModeForCategory(category)

		if modeName == "gamemode" {
			gamemodeShortcuts = append(gamemodeShortcuts, shortcut)
		} else {
			// focusmode gets games and other distractions
			focusmodeShortcuts = append(focusmodeShortcuts, shortcut)
		}
	}

	// Create config structure
	config := Config{
		Modes:       make(map[string]ModeConfig),
		DefaultMode: "focusmode",
	}

	// Set up focusmode (moves games and other distractions)
	if len(focusmodeShortcuts) > 0 {
		config.Modes["focusmode"] = ModeConfig{
			Destination: "Hidden_Shortcuts",
			Shortcuts:   focusmodeShortcuts,
			MoveAll:     false,
		}
	}

	// Set up gamemode (moves work/development tools)
	if len(gamemodeShortcuts) > 0 {
		config.Modes["gamemode"] = ModeConfig{
			Destination: "Hidden_Shortcuts",
			Shortcuts:   gamemodeShortcuts,
			MoveAll:     false,
		}
	}

	// If no modes were created, create at least focusmode with empty list
	if len(config.Modes) == 0 {
		config.Modes["focusmode"] = ModeConfig{
			Destination: "Hidden_Shortcuts",
			Shortcuts:   []string{},
			MoveAll:     false,
		}
	}

	// Generate YAML
	yamlData, err := yaml.Marshal(&config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating YAML: %v\n", err)
		os.Exit(1)
	}

	// Add header comment
	header := `# FocusMode Configuration
# Auto-generated from desktop shortcuts
# Review and adjust as needed

`
	fullYAML := header + string(yamlData)

	// Write to file
	err = os.WriteFile(configPath, []byte(fullYAML), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing config file: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Printf("✅ Generated %s\n\n", configPath)
	fmt.Println("Summary:")
	fmt.Printf("  FocusMode: %d shortcut(s) (games and other distractions - moved when focusing)\n", len(focusmodeShortcuts))
	fmt.Printf("  GameMode: %d shortcut(s) (work/development tools - moved when gaming)\n", len(gamemodeShortcuts))
	fmt.Printf("\nTotal shortcuts categorized: %d\n", len(shortcuts))
	fmt.Println("\nLogic:")
	fmt.Println("  - FocusMode: Moves games away → keeps work tools on desktop for quick access")
	fmt.Println("  - GameMode: Moves work tools away → keeps games on desktop for quick access")
	fmt.Printf("\nReview and edit %s if needed, then run:\n", configPath)
	fmt.Printf("  ./focusmode -mode focusmode -dry-run\n")
	fmt.Printf("  ./focusmode -mode gamemode -dry-run\n")
}

// restoreShortcutsForMode restores shortcuts from a specific mode's folder back to desktop
func restoreShortcutsForMode(config *Config, modeName string, dryRun bool) {
	// Get mode-specific configuration
	modeConfig, err := config.getModeConfig(modeName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Use -list-modes to see available modes\n")
		os.Exit(1)
	}

	fmt.Printf("Restoring shortcuts from mode: %s\n", modeName)

	// Get source folder
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	sourceFolder := filepath.Join(homeDir, modeConfig.Destination)

	// Check if source folder exists
	if _, err := os.Stat(sourceFolder); os.IsNotExist(err) {
		fmt.Printf("Source folder does not exist: %s\n", sourceFolder)
		fmt.Println("Nothing to restore.")
		return
	}

	// Get all shortcuts in the source folder
	shortcutsToRestore, err := getShortcutsInFolder(sourceFolder)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading source folder: %v\n", err)
		os.Exit(1)
	}

	if len(shortcutsToRestore) == 0 {
		fmt.Printf("No shortcuts found in %s\n", sourceFolder)
		return
	}

	fmt.Printf("Found %d shortcut(s) to restore from %s\n\n", len(shortcutsToRestore), sourceFolder)

	// Restore shortcuts
	successCount := 0
	failCount := 0

	for _, shortcutName := range shortcutsToRestore {
		if dryRun {
			fmt.Printf("[DRY RUN] Would restore: %s -> Desktop\n", shortcutName)
			successCount++
		} else {
			err := restoreShortcutToDesktop(shortcutName, sourceFolder)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error restoring '%s': %v\n", shortcutName, err)
				failCount++
			} else {
				fmt.Printf("✓ Restored: %s\n", shortcutName)
				successCount++
			}
		}
	}

	// Summary
	fmt.Println("\n--- Summary ---")
	fmt.Printf("Mode: %s\n", modeName)
	fmt.Printf("Successfully restored: %d\n", successCount)
	if failCount > 0 {
		fmt.Printf("Failed: %d\n", failCount)
	}
	if dryRun {
		fmt.Println("(Dry run - no files were actually restored)")
	} else {
		fmt.Printf("All shortcuts restored to desktop from: %s\n", sourceFolder)
	}
}

// restoreAllShortcuts restores shortcuts from all modes back to desktop
func restoreAllShortcuts(config *Config, dryRun bool) {
	fmt.Println("Restoring shortcuts from all modes...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	totalRestored := 0
	totalFailed := 0

	// Restore from each mode
	for modeName, modeConfig := range config.Modes {
		destination := modeConfig.Destination
		if destination == "" {
			destination = fmt.Sprintf("%s_Shortcuts", modeName)
		}

		sourceFolder := filepath.Join(homeDir, destination)

		// Check if folder exists
		if _, err := os.Stat(sourceFolder); os.IsNotExist(err) {
			fmt.Printf("Skipping %s (folder does not exist: %s)\n", modeName, sourceFolder)
			continue
		}

		// Get shortcuts in folder
		shortcuts, err := getShortcutsInFolder(sourceFolder)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading folder %s: %v\n", sourceFolder, err)
			continue
		}

		if len(shortcuts) == 0 {
			fmt.Printf("No shortcuts in %s\n", modeName)
			continue
		}

		fmt.Printf("Mode: %s (%d shortcut(s))\n", modeName, len(shortcuts))

		// Restore each shortcut
		for _, shortcutName := range shortcuts {
			if dryRun {
				fmt.Printf("  [DRY RUN] Would restore: %s\n", shortcutName)
				totalRestored++
			} else {
				err := restoreShortcutToDesktop(shortcutName, sourceFolder)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  Error restoring '%s': %v\n", shortcutName, err)
					totalFailed++
				} else {
					fmt.Printf("  ✓ Restored: %s\n", shortcutName)
					totalRestored++
				}
			}
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("--- Summary ---")
	fmt.Printf("Successfully restored: %d\n", totalRestored)
	if totalFailed > 0 {
		fmt.Printf("Failed: %d\n", totalFailed)
	}
	if dryRun {
		fmt.Println("(Dry run - no files were actually restored)")
	} else {
		fmt.Println("All shortcuts restored to desktop from all modes")
	}
}

func main() {
	// Command-line flags
	configPath := flag.String("config", "profile.yml", "Path to configuration file")
	categoriesPath := flag.String("categories", "categories.yml", "Path to categories configuration file")
	mode := flag.String("mode", "", "Mode to use (focusmode, gamemode, etc.)")
	dryRun := flag.Bool("dry-run", false, "Show what would be moved without actually moving")
	listModes := flag.Bool("list-modes", false, "List all available modes")
	listDesktop := flag.Bool("list-desktop", false, "List all files on desktop")
	autoConfig := flag.Bool("auto-config", false, "Auto-generate profile.yml based on desktop shortcuts and categories")
	restore := flag.Bool("restore", false, "Restore shortcuts from organized folder back to desktop")
	restoreAll := flag.Bool("restore-all", false, "Restore shortcuts from all modes back to desktop")
	switchMode := flag.Bool("switch", false, "Restore all shortcuts, then apply the selected mode")
	tray := flag.Bool("tray", false, "Run as a Windows system tray app")
	statsFlag := flag.Bool("stats", false, "Show tracked mode usage totals")
	flag.Parse()

	// Auto-generate profile if requested
	if *autoConfig {
		generateProfileFromDesktop(*configPath, *categoriesPath)
		return
	}

	// Restore shortcuts if requested
	if *restore || *restoreAll {
		// Load configuration
		config, err := loadConfig(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		if *restoreAll {
			restoreAllShortcuts(config, *dryRun)
		} else {
			// Determine which mode to restore
			modeName := *mode
			if modeName == "" {
				modeName = config.DefaultMode
			}
			restoreShortcutsForMode(config, modeName, *dryRun)
		}
		stopModeTracking(getStatsPath(*configPath), *dryRun)
		return
	}

	// List desktop files if requested (doesn't require config)
	if *listDesktop {
		// Load categories config for listing
		categoriesConfig, err := loadCategoriesConfig(*categoriesPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Error loading categories config: %v\n", err)
			fmt.Fprintf(os.Stderr, "Using default categories.\n\n")
			categoriesConfig = getDefaultCategoriesConfig()
		}
		listDesktopFilesWithConfig(categoriesConfig)
		return
	}

	// Load configuration
	config, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	statsPath := getStatsPath(*configPath)

	if *statsFlag {
		displayModeStats(config, statsPath)
		return
	}

	if *tray {
		if err := runTray(config, *configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting tray app: %v\n", err)
			os.Exit(1)
		}
		return
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

	if *switchMode {
		fmt.Printf("Switching to mode: %s\n\n", modeName)
		stopModeTracking(statsPath, *dryRun)
		restoreAllShortcuts(config, *dryRun)
		fmt.Println()
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

	startModeTracking(statsPath, modeName, *dryRun)
}
