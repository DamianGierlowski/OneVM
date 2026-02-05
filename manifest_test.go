package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifest(t *testing.T) {
	dir := t.TempDir()

	t.Run("valid manifest", func(t *testing.T) {
		path := filepath.Join(dir, "valid.json")
		os.WriteFile(path, []byte(`{
			"servers": [{"host": "10.0.0.1", "user": "admin", "key": "~/.ssh/id_rsa"}],
			"files": [{"local": "./nginx.conf", "remote": "/etc/nginx/nginx.conf"}]
		}`), 0644)

		m, err := LoadManifest(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(m.Servers) != 1 {
			t.Errorf("got %d servers, want 1", len(m.Servers))
		}
		if m.Servers[0].Host != "10.0.0.1" {
			t.Errorf("got host %q, want %q", m.Servers[0].Host, "10.0.0.1")
		}
		if len(m.Files) != 1 {
			t.Errorf("got %d files, want 1", len(m.Files))
		}
	})

	t.Run("manifest with restart command", func(t *testing.T) {
		path := filepath.Join(dir, "restart.json")
		os.WriteFile(path, []byte(`{
			"servers": [{"host": "10.0.0.1", "user": "admin", "key": "~/.ssh/id_rsa"}],
			"files": [{"local": "./app.conf", "remote": "/etc/app.conf", "restart": "systemctl reload app"}]
		}`), 0644)

		m, err := LoadManifest(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Files[0].Restart != "systemctl reload app" {
			t.Errorf("got restart %q, want %q", m.Files[0].Restart, "systemctl reload app")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := LoadManifest(filepath.Join(dir, "nope.json"))
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		path := filepath.Join(dir, "bad.json")
		os.WriteFile(path, []byte(`{not json}`), 0644)

		_, err := LoadManifest(path)
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})
}

func TestValidateManifest(t *testing.T) {
	tests := []struct {
		name    string
		m       Manifest
		wantErr bool
	}{
		{
			name: "valid",
			m: Manifest{
				Servers: []ServerConfig{{Host: "h", User: "u", Key: "k"}},
				Files:   []FileConfig{{Local: "l", Remote: "r"}},
			},
			wantErr: false,
		},
		{
			name: "no servers",
			m: Manifest{
				Servers: []ServerConfig{},
				Files:   []FileConfig{{Local: "l", Remote: "r"}},
			},
			wantErr: true,
		},
		{
			name: "no files",
			m: Manifest{
				Servers: []ServerConfig{{Host: "h", User: "u", Key: "k"}},
				Files:   []FileConfig{},
			},
			wantErr: true,
		},
		{
			name: "server missing host",
			m: Manifest{
				Servers: []ServerConfig{{Host: "", User: "u", Key: "k"}},
				Files:   []FileConfig{{Local: "l", Remote: "r"}},
			},
			wantErr: true,
		},
		{
			name: "server missing user",
			m: Manifest{
				Servers: []ServerConfig{{Host: "h", User: "", Key: "k"}},
				Files:   []FileConfig{{Local: "l", Remote: "r"}},
			},
			wantErr: true,
		},
		{
			name: "server with password instead of key",
			m: Manifest{
				Servers: []ServerConfig{{Host: "h", User: "u", Password: "p"}},
				Files:   []FileConfig{{Local: "l", Remote: "r"}},
			},
			wantErr: false,
		},
		{
			name: "server missing key and password",
			m: Manifest{
				Servers: []ServerConfig{{Host: "h", User: "u"}},
				Files:   []FileConfig{{Local: "l", Remote: "r"}},
			},
			wantErr: true,
		},
		{
			name: "file missing local",
			m: Manifest{
				Servers: []ServerConfig{{Host: "h", User: "u", Key: "k"}},
				Files:   []FileConfig{{Local: "", Remote: "r"}},
			},
			wantErr: true,
		},
		{
			name: "file missing remote",
			m: Manifest{
				Servers: []ServerConfig{{Host: "h", User: "u", Key: "k"}},
				Files:   []FileConfig{{Local: "l", Remote: ""}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateManifest(&tt.m)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateManifest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseServerString(t *testing.T) {
	tests := []struct {
		input    string
		wantUser string
		wantHost string
		wantErr  bool
	}{
		{"admin@10.0.0.1", "admin", "10.0.0.1", false},
		{"root@server.local", "root", "server.local", false},
		{"bad-format", "", "", true},
		{"@nouser", "", "", true},
		{"nohost@", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			user, host, err := parseServerString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if user != tt.wantUser {
				t.Errorf("user = %q, want %q", user, tt.wantUser)
			}
			if host != tt.wantHost {
				t.Errorf("host = %q, want %q", host, tt.wantHost)
			}
		})
	}
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input string
		want  string
	}{
		{"~/.ssh/id_rsa", filepath.Join(home, ".ssh/id_rsa")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := expandHome(tt.input)
			if got != tt.want {
				t.Errorf("expandHome(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
