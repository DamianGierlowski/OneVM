package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Manifest struct {
	Servers []ServerConfig `json:"servers"`
	Files   []FileConfig   `json:"files"`
}

type ServerConfig struct {
	Host     string `json:"host"`
	User     string `json:"user"`
	Key      string `json:"key,omitempty"`
	Password string `json:"password,omitempty"`
}

type FileConfig struct {
	Local   string `json:"local"`
	Remote  string `json:"remote"`
	Restart string `json:"restart,omitempty"`
}

func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest %s: %w", path, err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	if err := ValidateManifest(&manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

func ValidateManifest(m *Manifest) error {
	if len(m.Servers) == 0 {
		return fmt.Errorf("manifest: no servers defined")
	}
	if len(m.Files) == 0 {
		return fmt.Errorf("manifest: no files defined")
	}

	for i, s := range m.Servers {
		if s.Host == "" {
			return fmt.Errorf("manifest: server[%d] missing host", i)
		}
		if s.User == "" {
			return fmt.Errorf("manifest: server[%d] missing user", i)
		}
		if s.Key == "" && s.Password == "" {
			return fmt.Errorf("manifest: server[%d] missing key or password", i)
		}
	}

	for i, f := range m.Files {
		if f.Local == "" {
			return fmt.Errorf("manifest: file[%d] missing local path", i)
		}
		if f.Remote == "" {
			return fmt.Errorf("manifest: file[%d] missing remote path", i)
		}
	}

	return nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
