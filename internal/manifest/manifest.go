package manifest

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type PackageEntry struct {
	Name    string `toml:"name"`
	Manager string `toml:"manager"`
}

type ServiceEntry struct {
	Name    string `toml:"name"`
	Manager string `toml:"manager"`
}

type Manifest struct {
	Packages []PackageEntry `toml:"packages"`
	Services []ServiceEntry `toml:"services"`
}

func StateFile(configDir string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256([]byte(configDir))
	prefix := fmt.Sprintf("%x", hash[:8])
	dir := filepath.Join(home, ".local", "state", "pkgm")
	return filepath.Join(dir, prefix+".toml"), nil
}

func Load(configDir string) (*Manifest, error) {
	path, err := StateFile(configDir)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Manifest{}, nil
	}

	var m Manifest
	if _, err := toml.DecodeFile(path, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &m, nil
}

func Save(configDir string, m *Manifest) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	stateDir := filepath.Join(home, ".local", "state", "pkgm")
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return err
	}

	path, err := StateFile(configDir)
	if err != nil {
		return err
	}

	// Build raw map to control output order.
	// Services must come before packages so they're not nested inside [[packages]].
	raw := make(map[string]any)

	if len(m.Services) > 0 {
		raw["services"] = m.Services
	}
	if len(m.Packages) > 0 {
		raw["packages"] = m.Packages
	}

	// Write atomically: write to temp file, then rename.
	tmp, err := os.CreateTemp(stateDir, "manifest-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := toml.NewEncoder(tmp).Encode(raw); err != nil {
		tmp.Close()
		return fmt.Errorf("encode: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}
