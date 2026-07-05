//go:build !windows

package main

import "fmt"

func runTray(config *Config, configPath string) error {
	return fmt.Errorf("tray mode is only supported on Windows")
}
