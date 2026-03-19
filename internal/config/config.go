package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const fileName = "config.json"
const dbFileName = "rules.db"

type File struct {
	DBPath     string `json:"db_path"`
	SyncEditor string `json:"sync_editor,omitempty"`
}

type Resolved struct {
	Dir        string
	ConfigPath string
	DBPath     string
	SyncEditor string
}

func Ensure() (Resolved, error) {
	dir, err := configDir()
	if err != nil {
		return Resolved{}, err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Resolved{}, err
	}

	configPath := filepath.Join(dir, fileName)
	defaultDBPath := filepath.Join(dir, dbFileName)

	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		cfg := File{DBPath: defaultDBPath}
		if err := write(configPath, cfg); err != nil {
			return Resolved{}, err
		}
		return Resolved{Dir: dir, ConfigPath: configPath, DBPath: cfg.DBPath, SyncEditor: cfg.SyncEditor}, nil
	} else if err != nil {
		return Resolved{}, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return Resolved{}, err
	}

	var cfg File
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Resolved{}, err
	}

	if cfg.DBPath == "" {
		cfg.DBPath = defaultDBPath
	}
	cfg.SyncEditor = strings.TrimSpace(cfg.SyncEditor)

	normalizedDBPath, err := normalizePath(dir, cfg.DBPath)
	if err != nil {
		return Resolved{}, err
	}

	if cfg.DBPath != normalizedDBPath {
		cfg.DBPath = normalizedDBPath
		if err := write(configPath, cfg); err != nil {
			return Resolved{}, err
		}
	}

	return Resolved{Dir: dir, ConfigPath: configPath, DBPath: normalizedDBPath, SyncEditor: cfg.SyncEditor}, nil
}

func SetSyncEditor(editor string) (Resolved, error) {
	resolved, err := Ensure()
	if err != nil {
		return Resolved{}, err
	}

	cfg := File{DBPath: resolved.DBPath, SyncEditor: strings.TrimSpace(editor)}
	if err := write(resolved.ConfigPath, cfg); err != nil {
		return Resolved{}, err
	}
	resolved.SyncEditor = cfg.SyncEditor
	return resolved, nil
}

func configDir() (string, error) {
	if dir := os.Getenv("MYNAMR_CONFIG_DIR"); dir != "" {
		return normalizePath("", dir)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", "mynamr"), nil
}

func normalizePath(baseDir, raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}

	if strings.HasPrefix(trimmed, "~/") || strings.HasPrefix(trimmed, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		trimmed = filepath.Join(home, trimmed[2:])
	}

	if baseDir != "" && !filepath.IsAbs(trimmed) {
		trimmed = filepath.Join(baseDir, trimmed)
	}

	return filepath.Abs(trimmed)
}

func write(path string, cfg File) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
