package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Server.Port != 7681 {
		t.Errorf("expected default port 7681, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("expected default host 0.0.0.0, got %s", cfg.Server.Host)
	}
	if cfg.Terminal.Shell == "" {
		t.Error("expected non-empty default shell")
	}
	if cfg.Desktop.Theme != "classic" {
		t.Errorf("expected default theme 'classic', got %s", cfg.Desktop.Theme)
	}
}

func TestLoadConfig(t *testing.T) {
	tomlData := `
[server]
host = "127.0.0.1"
port = 8888

[terminal]
shell = "/bin/sh"
scrollback = 2000
font_size = 16

[desktop]
theme = "amber"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(tomlData), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("expected host '127.0.0.1', got %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 8888 {
		t.Errorf("expected port 8888, got %d", cfg.Server.Port)
	}
	if cfg.Terminal.Shell != "/bin/sh" {
		t.Errorf("expected shell '/bin/sh', got %s", cfg.Terminal.Shell)
	}
	if cfg.Terminal.Scrollback != 2000 {
		t.Errorf("expected scrollback 2000, got %d", cfg.Terminal.Scrollback)
	}
	if cfg.Terminal.FontSize != 16 {
		t.Errorf("expected font_size 16, got %d", cfg.Terminal.FontSize)
	}
	if cfg.Desktop.Theme != "amber" {
		t.Errorf("expected theme 'amber', got %s", cfg.Desktop.Theme)
	}
}
