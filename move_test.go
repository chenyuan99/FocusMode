package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
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

