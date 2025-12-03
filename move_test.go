package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestGetDesktopPath(t *testing.T) {
	desktopPath, err := getDesktopPath()
	if err != nil {
		t.Fatalf("getDesktopPath() returned error: %v", err)
	}

	if desktopPath == "" {
		t.Error("getDesktopPath() returned empty path")
	}

	// Verify the path structure based on OS
	switch runtime.GOOS {
	case "windows":
		if !filepath.IsAbs(desktopPath) {
			t.Errorf("Expected absolute path on Windows, got: %s", desktopPath)
		}
		// On Windows, should contain Desktop (allow for localized names)
		baseName := filepath.Base(desktopPath)
		if baseName != "Desktop" && baseName != "桌面" {
			t.Logf("Desktop path: %s (may be localized)", desktopPath)
		}
	case "darwin", "linux":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("Failed to get home directory: %v", err)
		}
		expectedPath := filepath.Join(homeDir, "Desktop")
		if desktopPath != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, desktopPath)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yml")

	// Test valid config
	validConfig := `modes:
  focusmode:
    destination: "TestFolder"
    shortcuts:
      - "test1.lnk"
      - "test2.lnk"
    move_all: false
  gamemode:
    destination: "GameFolder"
    shortcuts:
      - "game1.lnk"
    move_all: false
default_mode: "focusmode"
`

	err := os.WriteFile(configPath, []byte(validConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("loadConfig() returned error: %v", err)
	}

	if config == nil {
		t.Fatal("loadConfig() returned nil config")
	}

	if config.DefaultMode != "focusmode" {
		t.Errorf("Expected default_mode 'focusmode', got '%s'", config.DefaultMode)
	}

	if len(config.Modes) != 2 {
		t.Errorf("Expected 2 modes, got %d", len(config.Modes))
	}

	// Test default mode when not specified
	configWithoutDefault := `modes:
  focusmode:
    destination: "TestFolder"
    shortcuts: []
    move_all: false
`

	configPath2 := filepath.Join(tempDir, "test_config2.yml")
	err = os.WriteFile(configPath2, []byte(configWithoutDefault), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config2, err := loadConfig(configPath2)
	if err != nil {
		t.Fatalf("loadConfig() returned error: %v", err)
	}

	if config2.DefaultMode != "focusmode" {
		t.Errorf("Expected default mode 'focusmode' when not specified, got '%s'", config2.DefaultMode)
	}

	// Test invalid config file
	_, err = loadConfig("nonexistent.yml")
	if err == nil {
		t.Error("Expected error for nonexistent config file")
	}

	// Test invalid YAML
	invalidConfig := `modes:
  focusmode:
    destination: "TestFolder"
    invalid: [unclosed
`

	configPath3 := filepath.Join(tempDir, "test_config3.yml")
	err = os.WriteFile(configPath3, []byte(invalidConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err = loadConfig(configPath3)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestConfigGetModeConfig(t *testing.T) {
	config := &Config{
		Modes: map[string]ModeConfig{
			"focusmode": {
				Destination: "FocusFolder",
				Shortcuts:   []string{"test.lnk"},
				MoveAll:     false,
			},
			"gamemode": {
				Destination: "",
				Shortcuts:   []string{"game.lnk"},
				MoveAll:     false,
			},
		},
		DefaultMode: "focusmode",
	}

	// Test existing mode
	modeConfig, err := config.getModeConfig("focusmode")
	if err != nil {
		t.Fatalf("getModeConfig() returned error: %v", err)
	}

	if modeConfig.Destination != "FocusFolder" {
		t.Errorf("Expected destination 'FocusFolder', got '%s'", modeConfig.Destination)
	}

	if len(modeConfig.Shortcuts) != 1 {
		t.Errorf("Expected 1 shortcut, got %d", len(modeConfig.Shortcuts))
	}

	// Test mode with empty destination (should get default)
	modeConfig2, err := config.getModeConfig("gamemode")
	if err != nil {
		t.Fatalf("getModeConfig() returned error: %v", err)
	}

	expectedDestination := "gamemode_Shortcuts"
	if modeConfig2.Destination != expectedDestination {
		t.Errorf("Expected default destination '%s', got '%s'", expectedDestination, modeConfig2.Destination)
	}

	// Test nonexistent mode
	_, err = config.getModeConfig("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent mode")
	}
}

func TestConfigGetAvailableModes(t *testing.T) {
	config := &Config{
		Modes: map[string]ModeConfig{
			"focusmode": {},
			"gamemode":  {},
			"workmode":  {},
		},
		DefaultMode: "focusmode",
	}

	modes := config.getAvailableModes()
	if len(modes) != 3 {
		t.Errorf("Expected 3 modes, got %d", len(modes))
	}

	// Check that all expected modes are present
	expectedModes := map[string]bool{
		"focusmode": true,
		"gamemode":  true,
		"workmode":  true,
	}

	for _, mode := range modes {
		if !expectedModes[mode] {
			t.Errorf("Unexpected mode: %s", mode)
		}
		delete(expectedModes, mode)
	}

	if len(expectedModes) > 0 {
		t.Errorf("Missing modes: %v", expectedModes)
	}
}

func TestMoveDesktopShortcut(t *testing.T) {
	// Create temporary directories to simulate desktop and destination
	tempDir := t.TempDir()
	desktopDir := filepath.Join(tempDir, "Desktop")
	destDir := filepath.Join(tempDir, "Destination")

	err := os.MkdirAll(desktopDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create desktop directory: %v", err)
	}

	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}

	// Create a test file on the "desktop"
	testFile := filepath.Join(desktopDir, "test.lnk")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test moving the file using the testable function
	err = moveDesktopShortcutFromPath("test.lnk", destDir, desktopDir)
	if err != nil {
		t.Fatalf("moveDesktopShortcutFromPath() returned error: %v", err)
	}

	// Verify file was moved
	expectedPath := filepath.Join(destDir, "test.lnk")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("File was not moved to destination")
	}

	if _, err := os.Stat(testFile); err == nil {
		t.Error("File still exists in source location")
	}

	// Test moving nonexistent file
	err = moveDesktopShortcutFromPath("nonexistent.lnk", destDir, desktopDir)
	if err == nil {
		t.Error("Expected error when moving nonexistent file")
	}
}

func TestGetAllDesktopShortcuts(t *testing.T) {
	// Create temporary desktop directory
	tempDir := t.TempDir()
	desktopDir := filepath.Join(tempDir, "Desktop")

	err := os.MkdirAll(desktopDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create desktop directory: %v", err)
	}

	// Create test files
	testFiles := []string{"file1.lnk", "file2.lnk", "file3.txt"}
	for _, filename := range testFiles {
		filePath := filepath.Join(desktopDir, filename)
		err := os.WriteFile(filePath, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create a subdirectory (should be ignored)
	subDir := filepath.Join(desktopDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Test using the testable function
	shortcuts, err := getAllDesktopShortcutsFromPath(desktopDir)
	if err != nil {
		t.Fatalf("getAllDesktopShortcutsFromPath() returned error: %v", err)
	}

	if len(shortcuts) != len(testFiles) {
		t.Errorf("Expected %d shortcuts, got %d", len(testFiles), len(shortcuts))
	}

	// Verify all test files are in the list
	shortcutMap := make(map[string]bool)
	for _, shortcut := range shortcuts {
		shortcutMap[shortcut] = true
	}

	for _, testFile := range testFiles {
		if !shortcutMap[testFile] {
			t.Errorf("Expected shortcut %s not found in results", testFile)
		}
	}
}

func TestModeConfigDefaults(t *testing.T) {
	// Test that empty destination gets default value
	config := &Config{
		Modes: map[string]ModeConfig{
			"testmode": {
				Destination: "",
				Shortcuts:   []string{},
				MoveAll:     false,
			},
		},
	}

	modeConfig, err := config.getModeConfig("testmode")
	if err != nil {
		t.Fatalf("getModeConfig() returned error: %v", err)
	}

	expected := "testmode_Shortcuts"
	if modeConfig.Destination != expected {
		t.Errorf("Expected default destination '%s', got '%s'", expected, modeConfig.Destination)
	}
}

func TestGetModeForCategory(t *testing.T) {
	tests := []struct {
		category ShortcutCategory
		expected string
		desc     string
	}{
		{ShortcutCategory("game"), "focusmode", "Games should go to focusmode (moved when focusing)"},
		{ShortcutCategory("development"), "gamemode", "Development tools should go to gamemode (moved when gaming)"},
		{ShortcutCategory("work"), "gamemode", "Work tools should go to gamemode (moved when gaming)"},
		{ShortcutCategory("other"), "focusmode", "Other items should go to focusmode"},
		{ShortcutCategory("unknown"), "focusmode", "Unknown categories should default to focusmode"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := getModeForCategory(tt.category)
			if result != tt.expected {
				t.Errorf("getModeForCategory(%s) = %s, want %s", tt.category, result, tt.expected)
			}
		})
	}
}

func TestCategorizeShortcut(t *testing.T) {
	// Create a test categories config
	categoriesConfig := &CategoriesConfig{
		Categories: map[string]CategoryConfig{
			"game": {
				Name:     "Games",
				Icon:     "🎮",
				Keywords: []string{"steam", "game", "epic"},
			},
			"development": {
				Name:     "Development Tools",
				Icon:     "💻",
				Keywords: []string{"code", "docker", "cursor"},
			},
			"work": {
				Name:     "Work/Productivity",
				Icon:     "💼",
				Keywords: []string{"office", "word", "excel"},
			},
		},
		CategoryOrder: []string{"game", "development", "work", "other"},
	}

	tests := []struct {
		name     string
		expected ShortcutCategory
		desc     string
	}{
		{"Steam.lnk", ShortcutCategory("game"), "Game shortcut"},
		{"My Game.url", ShortcutCategory("game"), "Game URL"},
		{"Visual Studio Code.lnk", ShortcutCategory("development"), "Development tool"},
		{"Docker Desktop.lnk", ShortcutCategory("development"), "Development tool"},
		{"Cursor.lnk", ShortcutCategory("development"), "Development tool"},
		{"Microsoft Word.lnk", ShortcutCategory("work"), "Work tool"},
		{"Excel.lnk", ShortcutCategory("work"), "Work tool"},
		{"RandomFile.txt", ShortcutCategory("other"), "Uncategorized file"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := categorizeShortcut(tt.name, categoriesConfig)
			if result != tt.expected {
				t.Errorf("categorizeShortcut(%s) = %s, want %s", tt.name, result, tt.expected)
			}
		})
	}
}

func TestLoadCategoriesConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "categories.yml")

	// Test valid categories config
	validConfig := `categories:
  game:
    name: "Games"
    icon: "🎮"
    keywords:
      - "steam"
      - "game"
  development:
    name: "Development Tools"
    icon: "💻"
    keywords:
      - "code"
      - "docker"
category_order:
  - game
  - development
  - other
`

	err := os.WriteFile(configPath, []byte(validConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := loadCategoriesConfig(configPath)
	if err != nil {
		t.Fatalf("loadCategoriesConfig() returned error: %v", err)
	}

	if config == nil {
		t.Fatal("loadCategoriesConfig() returned nil config")
	}

	if len(config.Categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(config.Categories))
	}

	if gameCat, ok := config.Categories["game"]; ok {
		if gameCat.Name != "Games" {
			t.Errorf("Expected game category name 'Games', got '%s'", gameCat.Name)
		}
		if len(gameCat.Keywords) != 2 {
			t.Errorf("Expected 2 keywords for game, got %d", len(gameCat.Keywords))
		}
	} else {
		t.Error("Game category not found")
	}

	// Test default config when file doesn't exist
	defaultConfig, err := loadCategoriesConfig("nonexistent.yml")
	if err != nil {
		t.Fatalf("loadCategoriesConfig() should return default config, got error: %v", err)
	}
	if defaultConfig == nil {
		t.Error("Expected default config when file doesn't exist")
	}
}

func TestRestoreShortcutToDesktop(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	desktopDir := filepath.Join(tempDir, "Desktop")
	sourceDir := filepath.Join(tempDir, "Source")

	err := os.MkdirAll(desktopDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create desktop directory: %v", err)
	}

	err = os.MkdirAll(sourceDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create a test file in source directory
	testFile := filepath.Join(sourceDir, "test.lnk")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Set USERPROFILE environment variable for Windows to point to tempDir
	originalUserProfile := os.Getenv("USERPROFILE")
	if runtime.GOOS == "windows" {
		os.Setenv("USERPROFILE", tempDir)
		defer os.Setenv("USERPROFILE", originalUserProfile)
	}

	// Test restoring the file
	err = restoreShortcutToDesktop("test.lnk", sourceDir)
	if err != nil {
		t.Fatalf("restoreShortcutToDesktop() returned error: %v", err)
	}

	// Verify file was moved to desktop
	expectedPath := filepath.Join(desktopDir, "test.lnk")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("File was not restored to desktop")
	}

	if _, err := os.Stat(testFile); err == nil {
		t.Error("File still exists in source location")
	}

	// Test restoring nonexistent file
	err = restoreShortcutToDesktop("nonexistent.lnk", sourceDir)
	if err == nil {
		t.Error("Expected error when restoring nonexistent file")
	}

	// Test restoring when file already exists on desktop
	err = os.WriteFile(filepath.Join(sourceDir, "existing.lnk"), []byte("source"), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	err = os.WriteFile(filepath.Join(desktopDir, "existing.lnk"), []byte("desktop"), 0644)
	if err != nil {
		t.Fatalf("Failed to create desktop file: %v", err)
	}

	err = restoreShortcutToDesktop("existing.lnk", sourceDir)
	if err == nil {
		t.Error("Expected error when file already exists on desktop")
	}
}

func TestGetShortcutsInFolder(t *testing.T) {
	tempDir := t.TempDir()
	testFolder := filepath.Join(tempDir, "TestFolder")

	err := os.MkdirAll(testFolder, 0755)
	if err != nil {
		t.Fatalf("Failed to create test folder: %v", err)
	}

	// Create test files
	testFiles := []string{"file1.lnk", "file2.lnk", "file3.txt"}
	for _, filename := range testFiles {
		filePath := filepath.Join(testFolder, filename)
		err := os.WriteFile(filePath, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create a subdirectory (should be ignored)
	subDir := filepath.Join(testFolder, "subdir")
	err = os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	shortcuts, err := getShortcutsInFolder(testFolder)
	if err != nil {
		t.Fatalf("getShortcutsInFolder() returned error: %v", err)
	}

	if len(shortcuts) != len(testFiles) {
		t.Errorf("Expected %d shortcuts, got %d", len(testFiles), len(shortcuts))
	}

	// Verify all test files are in the list
	shortcutMap := make(map[string]bool)
	for _, shortcut := range shortcuts {
		shortcutMap[shortcut] = true
	}

	for _, testFile := range testFiles {
		if !shortcutMap[testFile] {
			t.Errorf("Expected shortcut %s not found in results", testFile)
		}
	}
}

func TestGenerateProfileFromDesktop(t *testing.T) {
	// Create temporary desktop directory
	tempDir := t.TempDir()
	desktopDir := filepath.Join(tempDir, "Desktop")
	configPath := filepath.Join(tempDir, "profile.yml")
	categoriesPath := filepath.Join(tempDir, "categories.yml")

	err := os.MkdirAll(desktopDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create desktop directory: %v", err)
	}

	// Create test shortcuts
	testShortcuts := []string{
		"Steam.lnk",              // game
		"Visual Studio Code.lnk", // development
		"Microsoft Word.lnk",     // work
		"RandomFile.txt",         // other
	}
	for _, filename := range testShortcuts {
		filePath := filepath.Join(desktopDir, filename)
		err := os.WriteFile(filePath, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create categories config
	categoriesConfig := `categories:
  game:
    name: "Games"
    icon: "🎮"
    keywords:
      - "steam"
      - "game"
  development:
    name: "Development Tools"
    icon: "💻"
    keywords:
      - "code"
      - "visual studio"
  work:
    name: "Work/Productivity"
    icon: "💼"
    keywords:
      - "word"
      - "microsoft"
category_order:
  - game
  - development
  - work
  - other
`
	err = os.WriteFile(categoriesPath, []byte(categoriesConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write categories config: %v", err)
	}

	// Set USERPROFILE environment variable for Windows to point to tempDir
	originalUserProfile := os.Getenv("USERPROFILE")
	if runtime.GOOS == "windows" {
		os.Setenv("USERPROFILE", tempDir)
		defer os.Setenv("USERPROFILE", originalUserProfile)
	}

	// Generate profile
	generateProfileFromDesktop(configPath, categoriesPath)

	// Load and verify generated config
	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load generated config: %v", err)
	}

	// Verify focusmode has games and other
	focusmodeConfig, ok := config.Modes["focusmode"]
	if !ok {
		t.Fatal("focusmode not found in generated config")
	}

	// Focusmode should have Steam.lnk (game) and RandomFile.txt (other)
	hasSteam := false
	hasRandomFile := false
	for _, shortcut := range focusmodeConfig.Shortcuts {
		if shortcut == "Steam.lnk" {
			hasSteam = true
		}
		if shortcut == "RandomFile.txt" {
			hasRandomFile = true
		}
	}
	if !hasSteam {
		t.Error("Steam.lnk (game) should be in focusmode")
	}
	if !hasRandomFile {
		t.Error("RandomFile.txt (other) should be in focusmode")
	}

	// Verify gamemode has work and development tools
	gamemodeConfig, ok := config.Modes["gamemode"]
	if !ok {
		t.Fatal("gamemode not found in generated config")
	}

	// Gamemode should have Visual Studio Code.lnk (development) and Microsoft Word.lnk (work)
	hasVSCode := false
	hasWord := false
	for _, shortcut := range gamemodeConfig.Shortcuts {
		if shortcut == "Visual Studio Code.lnk" {
			hasVSCode = true
		}
		if shortcut == "Microsoft Word.lnk" {
			hasWord = true
		}
	}
	if !hasVSCode {
		t.Error("Visual Studio Code.lnk (development) should be in gamemode")
	}
	if !hasWord {
		t.Error("Microsoft Word.lnk (work) should be in gamemode")
	}

	// Verify destination is Hidden_Shortcuts
	if focusmodeConfig.Destination != "Hidden_Shortcuts" {
		t.Errorf("Expected focusmode destination 'Hidden_Shortcuts', got '%s'", focusmodeConfig.Destination)
	}
	if gamemodeConfig.Destination != "Hidden_Shortcuts" {
		t.Errorf("Expected gamemode destination 'Hidden_Shortcuts', got '%s'", gamemodeConfig.Destination)
	}
}

// TestFocusSessionElapsed tests the elapsed() method of FocusSession
func TestFocusSessionElapsed(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() *FocusSession
		expectedMin time.Duration
		expectedMax time.Duration
	}{
		{
			name: "Running session with no pause",
			setupFunc: func() *FocusSession {
				fs := &FocusSession{
					Duration:    30 * time.Minute,
					Mode:        "focusmode",
					StartTime:   time.Now().Add(-5 * time.Second),
					PausedAt:    nil,
					PausedTotal: 0,
					State:       StateRunning,
				}
				return fs
			},
			expectedMin: 4 * time.Second,
			expectedMax: 6 * time.Second,
		},
		{
			name: "Paused session",
			setupFunc: func() *FocusSession {
				pauseTime := time.Now().Add(-2 * time.Second)
				fs := &FocusSession{
					Duration:    30 * time.Minute,
					Mode:        "focusmode",
					StartTime:   time.Now().Add(-10 * time.Second),
					PausedAt:    &pauseTime,
					PausedTotal: 3 * time.Second,
					State:       StatePaused,
				}
				return fs
			},
			expectedMin: 4 * time.Second,
			expectedMax: 6 * time.Second,
		},
		{
			name: "Session with accumulated pause time",
			setupFunc: func() *FocusSession {
				fs := &FocusSession{
					Duration:    30 * time.Minute,
					Mode:        "focusmode",
					StartTime:   time.Now().Add(-15 * time.Second),
					PausedAt:    nil,
					PausedTotal: 5 * time.Second,
					State:       StateRunning,
				}
				return fs
			},
			expectedMin: 9 * time.Second,
			expectedMax: 11 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := tt.setupFunc()
			elapsed := fs.elapsed()

			if elapsed < tt.expectedMin || elapsed > tt.expectedMax {
				t.Errorf("elapsed() = %v, want between %v and %v", elapsed, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

// TestFocusSessionRemaining tests the remaining() method of FocusSession
func TestFocusSessionRemaining(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() *FocusSession
		expectedMin time.Duration
		expectedMax time.Duration
	}{
		{
			name: "Session with time remaining",
			setupFunc: func() *FocusSession {
				fs := &FocusSession{
					Duration:    30 * time.Minute,
					Mode:        "focusmode",
					StartTime:   time.Now().Add(-5 * time.Minute),
					PausedAt:    nil,
					PausedTotal: 0,
					State:       StateRunning,
				}
				return fs
			},
			expectedMin: 24*time.Minute + 50*time.Second,
			expectedMax: 25*time.Minute + 10*time.Second,
		},
		{
			name: "Session nearly complete",
			setupFunc: func() *FocusSession {
				fs := &FocusSession{
					Duration:    10 * time.Second,
					Mode:        "focusmode",
					StartTime:   time.Now().Add(-9 * time.Second),
					PausedAt:    nil,
					PausedTotal: 0,
					State:       StateRunning,
				}
				return fs
			},
			expectedMin: 0,
			expectedMax: 2 * time.Second,
		},
		{
			name: "Session overtime returns zero",
			setupFunc: func() *FocusSession {
				fs := &FocusSession{
					Duration:    5 * time.Second,
					Mode:        "focusmode",
					StartTime:   time.Now().Add(-10 * time.Second),
					PausedAt:    nil,
					PausedTotal: 0,
					State:       StateRunning,
				}
				return fs
			},
			expectedMin: 0,
			expectedMax: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := tt.setupFunc()
			remaining := fs.remaining()

			if remaining < tt.expectedMin || remaining > tt.expectedMax {
				t.Errorf("remaining() = %v, want between %v and %v", remaining, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

// TestSessionStateConstants tests that SessionState constants are defined correctly
func TestSessionStateConstants(t *testing.T) {
	// Verify that the constants are distinct
	states := map[SessionState]string{
		StateRunning:     "Running",
		StatePaused:      "Paused",
		StateCompleted:   "Completed",
		StateInterrupted: "Interrupted",
	}

	if len(states) != 4 {
		t.Errorf("Expected 4 distinct SessionState constants, got %d", len(states))
	}

	// Verify they have expected values (iota should start at 0)
	if StateRunning != 0 {
		t.Errorf("StateRunning should be 0, got %d", StateRunning)
	}
	if StatePaused != 1 {
		t.Errorf("StatePaused should be 1, got %d", StatePaused)
	}
	if StateCompleted != 2 {
		t.Errorf("StateCompleted should be 2, got %d", StateCompleted)
	}
	if StateInterrupted != 3 {
		t.Errorf("StateInterrupted should be 3, got %d", StateInterrupted)
	}
}

// TestOrganizeShortcuts tests the organizeShortcuts method of FocusSession
func TestOrganizeShortcuts(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	desktopDir := filepath.Join(tempDir, "Desktop")
	configPath := filepath.Join(tempDir, "profile.yml")

	err := os.MkdirAll(desktopDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create desktop directory: %v", err)
	}

	// Create test shortcuts on desktop
	testShortcuts := []string{"test1.lnk", "test2.lnk", "test3.lnk"}
	for _, filename := range testShortcuts {
		filePath := filepath.Join(desktopDir, filename)
		err := os.WriteFile(filePath, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create config file
	configContent := `modes:
  focusmode:
    destination: "TestDestination"
    shortcuts:
      - "test1.lnk"
      - "test2.lnk"
      - "nonexistent.lnk"
    move_all: false
default_mode: "focusmode"
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config
	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Set USERPROFILE environment variable for Windows to point to tempDir
	originalUserProfile := os.Getenv("USERPROFILE")
	if runtime.GOOS == "windows" {
		os.Setenv("USERPROFILE", tempDir)
		defer os.Setenv("USERPROFILE", originalUserProfile)
	}

	// Create FocusSession
	fs := &FocusSession{
		Duration:    30 * time.Minute,
		Mode:        "focusmode",
		StartTime:   time.Now(),
		AutoRestore: true,
		Config:      config,
		State:       StateRunning,
	}

	// Call organizeShortcuts
	movedShortcuts, err := fs.organizeShortcuts()
	if err != nil {
		t.Fatalf("organizeShortcuts() returned error: %v", err)
	}

	// Verify that 2 shortcuts were moved (test1.lnk and test2.lnk)
	// nonexistent.lnk should fail
	if len(movedShortcuts) != 2 {
		t.Errorf("Expected 2 shortcuts to be moved, got %d", len(movedShortcuts))
	}

	// Verify the moved shortcuts are tracked
	expectedMoved := map[string]bool{
		"test1.lnk": true,
		"test2.lnk": true,
	}

	for _, shortcut := range movedShortcuts {
		if !expectedMoved[shortcut] {
			t.Errorf("Unexpected shortcut in moved list: %s", shortcut)
		}
		delete(expectedMoved, shortcut)
	}

	if len(expectedMoved) > 0 {
		t.Errorf("Expected shortcuts not found in moved list: %v", expectedMoved)
	}

	// Verify shortcuts were actually moved
	destFolder := filepath.Join(tempDir, "TestDestination")
	for _, shortcut := range movedShortcuts {
		destPath := filepath.Join(destFolder, shortcut)
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			t.Errorf("Shortcut %s was not moved to destination", shortcut)
		}

		// Verify it's no longer on desktop
		desktopPath := filepath.Join(desktopDir, shortcut)
		if _, err := os.Stat(desktopPath); err == nil {
			t.Errorf("Shortcut %s still exists on desktop", shortcut)
		}
	}

	// Verify test3.lnk is still on desktop (not in config)
	test3Path := filepath.Join(desktopDir, "test3.lnk")
	if _, err := os.Stat(test3Path); os.IsNotExist(err) {
		t.Error("test3.lnk should still be on desktop")
	}
}

// TestOrganizeShortcutsMoveAll tests organizeShortcuts with move_all enabled
func TestOrganizeShortcutsMoveAll(t *testing.T) {
	// Create temporary directories
	tempDir := t.TempDir()
	desktopDir := filepath.Join(tempDir, "Desktop")
	configPath := filepath.Join(tempDir, "profile.yml")

	err := os.MkdirAll(desktopDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create desktop directory: %v", err)
	}

	// Create test shortcuts on desktop
	testShortcuts := []string{"test1.lnk", "test2.lnk", "test3.lnk"}
	for _, filename := range testShortcuts {
		filePath := filepath.Join(desktopDir, filename)
		err := os.WriteFile(filePath, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create config file with move_all enabled
	configContent := `modes:
  focusmode:
    destination: "TestDestination"
    shortcuts: []
    move_all: true
default_mode: "focusmode"
`
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load config
	config, err := loadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Set USERPROFILE environment variable for Windows to point to tempDir
	originalUserProfile := os.Getenv("USERPROFILE")
	if runtime.GOOS == "windows" {
		os.Setenv("USERPROFILE", tempDir)
		defer os.Setenv("USERPROFILE", originalUserProfile)
	}

	// Create FocusSession
	fs := &FocusSession{
		Duration:    30 * time.Minute,
		Mode:        "focusmode",
		StartTime:   time.Now(),
		AutoRestore: true,
		Config:      config,
		State:       StateRunning,
	}

	// Call organizeShortcuts
	movedShortcuts, err := fs.organizeShortcuts()
	if err != nil {
		t.Fatalf("organizeShortcuts() returned error: %v", err)
	}

	// Verify that all 3 shortcuts were moved
	if len(movedShortcuts) != 3 {
		t.Errorf("Expected 3 shortcuts to be moved, got %d", len(movedShortcuts))
	}

	// Verify all shortcuts are in the moved list
	expectedMoved := map[string]bool{
		"test1.lnk": true,
		"test2.lnk": true,
		"test3.lnk": true,
	}

	for _, shortcut := range movedShortcuts {
		if !expectedMoved[shortcut] {
			t.Errorf("Unexpected shortcut in moved list: %s", shortcut)
		}
		delete(expectedMoved, shortcut)
	}

	if len(expectedMoved) > 0 {
		t.Errorf("Expected shortcuts not found in moved list: %v", expectedMoved)
	}

	// Verify desktop is empty
	remainingShortcuts, err := getAllDesktopShortcutsFromPath(desktopDir)
	if err != nil {
		t.Fatalf("Failed to get desktop shortcuts: %v", err)
	}

	if len(remainingShortcuts) != 0 {
		t.Errorf("Expected desktop to be empty, found %d shortcuts: %v", len(remainingShortcuts), remainingShortcuts)
	}
}

// TestStartFocusSession tests the startFocusSession function
func TestStartFocusSession(t *testing.T) {
	// Create a test config
	config := &Config{
		Modes: map[string]ModeConfig{
			"focusmode": {
				Destination: "FocusFolder",
				Shortcuts:   []string{"test.lnk"},
				MoveAll:     false,
			},
			"gamemode": {
				Destination: "GameFolder",
				Shortcuts:   []string{"game.lnk"},
				MoveAll:     false,
			},
		},
		DefaultMode: "focusmode",
	}

	tests := []struct {
		name        string
		modeName    string
		duration    int
		autoRestore bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid session with focusmode",
			modeName:    "focusmode",
			duration:    25,
			autoRestore: true,
			expectError: false,
		},
		{
			name:        "Valid session with gamemode",
			modeName:    "gamemode",
			duration:    30,
			autoRestore: false,
			expectError: false,
		},
		{
			name:        "Invalid mode name",
			modeName:    "invalidmode",
			duration:    25,
			autoRestore: true,
			expectError: true,
			errorMsg:    "invalid mode",
		},
		{
			name:        "Zero duration",
			modeName:    "focusmode",
			duration:    0,
			autoRestore: true,
			expectError: true,
			errorMsg:    "duration must be positive",
		},
		{
			name:        "Negative duration",
			modeName:    "focusmode",
			duration:    -10,
			autoRestore: true,
			expectError: true,
			errorMsg:    "duration must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := startFocusSession(config, tt.modeName, tt.duration, tt.autoRestore)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
				if session != nil {
					t.Error("Expected nil session on error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if session == nil {
					t.Fatal("Expected non-nil session")
				}

				// Verify session fields
				if session.Mode != tt.modeName {
					t.Errorf("Expected mode '%s', got '%s'", tt.modeName, session.Mode)
				}

				expectedDuration := time.Duration(tt.duration) * time.Minute
				if session.Duration != expectedDuration {
					t.Errorf("Expected duration %v, got %v", expectedDuration, session.Duration)
				}

				if session.AutoRestore != tt.autoRestore {
					t.Errorf("Expected autoRestore %v, got %v", tt.autoRestore, session.AutoRestore)
				}

				if session.Config != config {
					t.Error("Expected session config to reference the provided config")
				}

				if session.State != StateRunning {
					t.Errorf("Expected initial state StateRunning, got %v", session.State)
				}

				if session.PausedAt != nil {
					t.Error("Expected PausedAt to be nil initially")
				}

				if session.PausedTotal != 0 {
					t.Error("Expected PausedTotal to be 0 initially")
				}

				// Verify StartTime is recent (within last second)
				timeSinceStart := time.Since(session.StartTime)
				if timeSinceStart < 0 || timeSinceStart > time.Second {
					t.Errorf("Expected StartTime to be recent, got %v ago", timeSinceStart)
				}
			}
		})
	}
}

// TestStartFocusSessionModeValidation tests that mode validation happens before session creation
func TestStartFocusSessionModeValidation(t *testing.T) {
	config := &Config{
		Modes: map[string]ModeConfig{
			"focusmode": {
				Destination: "FocusFolder",
				Shortcuts:   []string{"test.lnk"},
				MoveAll:     false,
			},
		},
		DefaultMode: "focusmode",
	}

	// Test with invalid mode
	session, err := startFocusSession(config, "nonexistent", 25, true)
	if err == nil {
		t.Error("Expected error for nonexistent mode")
	}
	if session != nil {
		t.Error("Expected nil session for invalid mode")
	}

	// Verify error message includes available modes
	if !strings.Contains(err.Error(), "focusmode") {
		t.Errorf("Expected error to list available modes, got: %v", err)
	}
}

// TestStartFocusSessionDurationValidation tests duration validation
func TestStartFocusSessionDurationValidation(t *testing.T) {
	config := &Config{
		Modes: map[string]ModeConfig{
			"focusmode": {
				Destination: "FocusFolder",
				Shortcuts:   []string{"test.lnk"},
				MoveAll:     false,
			},
		},
		DefaultMode: "focusmode",
	}

	invalidDurations := []int{0, -1, -10, -100}

	for _, duration := range invalidDurations {
		t.Run(fmt.Sprintf("Duration_%d", duration), func(t *testing.T) {
			session, err := startFocusSession(config, "focusmode", duration, true)
			if err == nil {
				t.Errorf("Expected error for duration %d", duration)
			}
			if session != nil {
				t.Error("Expected nil session for invalid duration")
			}
			if !strings.Contains(err.Error(), "duration must be positive") {
				t.Errorf("Expected error about positive duration, got: %v", err)
			}
		})
	}
}

// TestDisplayProgress tests the displayProgress function
func TestDisplayProgress(t *testing.T) {
	tests := []struct {
		name       string
		elapsed    time.Duration
		remaining  time.Duration
		paused     bool
		wantEmoji  string
		wantStatus string
	}{
		{
			name:       "Running session",
			elapsed:    5 * time.Minute,
			remaining:  20 * time.Minute,
			paused:     false,
			wantEmoji:  "⏳",
			wantStatus: "Focus Session",
		},
		{
			name:       "Paused session",
			elapsed:    10 * time.Minute,
			remaining:  15 * time.Minute,
			paused:     true,
			wantEmoji:  "⏸",
			wantStatus: "Paused",
		},
		{
			name:       "Nearly complete",
			elapsed:    24*time.Minute + 50*time.Second,
			remaining:  10 * time.Second,
			paused:     false,
			wantEmoji:  "⏳",
			wantStatus: "Focus Session",
		},
		{
			name:       "Just started",
			elapsed:    5 * time.Second,
			remaining:  25 * time.Minute,
			paused:     false,
			wantEmoji:  "⏳",
			wantStatus: "Focus Session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't easily capture stdout, but we can at least verify the function doesn't panic
			// and that formatDuration works correctly for the inputs
			displayProgress(tt.elapsed, tt.remaining, tt.paused)

			// Verify formatDuration produces expected output
			elapsedStr := formatDuration(tt.elapsed)
			remainingStr := formatDuration(tt.remaining)

			if elapsedStr == "" {
				t.Error("formatDuration returned empty string for elapsed time")
			}
			if remainingStr == "" {
				t.Error("formatDuration returned empty string for remaining time")
			}
		})
	}
}

// TestFormatDuration tests the formatDuration function
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "Zero duration",
			duration: 0,
			want:     "0s",
		},
		{
			name:     "Seconds only",
			duration: 45 * time.Second,
			want:     "45s",
		},
		{
			name:     "One minute",
			duration: 60 * time.Second,
			want:     "1m",
		},
		{
			name:     "Minutes and seconds",
			duration: 5*time.Minute + 30*time.Second,
			want:     "5m 30s",
		},
		{
			name:     "One hour",
			duration: 60 * time.Minute,
			want:     "1h",
		},
		{
			name:     "Hours and minutes",
			duration: 1*time.Hour + 25*time.Minute,
			want:     "1h 25m",
		},
		{
			name:     "Hours, minutes, and seconds",
			duration: 1*time.Hour + 5*time.Minute + 30*time.Second,
			want:     "1h 5m 30s",
		},
		{
			name:     "Multiple hours",
			duration: 2*time.Hour + 15*time.Minute + 45*time.Second,
			want:     "2h 15m 45s",
		},
		{
			name:     "Just under one minute",
			duration: 59 * time.Second,
			want:     "59s",
		},
		{
			name:     "Exactly 25 minutes",
			duration: 25 * time.Minute,
			want:     "25m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}
