package config

import (
	"os"

	"github.com/pelletier/go-toml/v2"
)

// ServerConfig holds HTTP/WS server parameters.
type ServerConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

// TerminalConfig holds terminal settings.
type TerminalConfig struct {
	Shell      string `toml:"shell"`
	Scrollback int    `toml:"scrollback"`
	FontSize   int    `toml:"font_size"`
}

// DesktopConfig holds desktop visual settings.
type DesktopConfig struct {
	Theme string `toml:"theme"`
}

// Config represents the application configuration.
type Config struct {
	Server   ServerConfig   `toml:"server"`
	Terminal TerminalConfig `toml:"terminal"`
	Desktop  DesktopConfig  `toml:"desktop"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	defaultShell := os.Getenv("SHELL")
	if defaultShell == "" {
		if _, err := os.Stat("/bin/bash"); err == nil {
			defaultShell = "/bin/bash"
		} else {
			defaultShell = "/bin/sh"
		}
	}

	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 7681,
		},
		Terminal: TerminalConfig{
			Shell:      defaultShell,
			Scrollback: 5000,
			FontSize:   14,
		},
		Desktop: DesktopConfig{
			Theme: "classic",
		},
	}
}

// LoadConfig loads configuration from a TOML file or returns defaults if path is empty.
func LoadConfig(filePath string) (*Config, error) {
	cfg := DefaultConfig()
	if filePath == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
