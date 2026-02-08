package vm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadClientConfig(t *testing.T) {
	dir := t.TempDir()

	t.Run("valid config", func(t *testing.T) {
		path := filepath.Join(dir, "valid.json")
		os.WriteFile(path, []byte(`{
			"hosts": {
				"prod": {"host": "10.0.0.1", "user": "admin", "key": "~/.ssh/id_rsa"},
				"dev": {"host": "10.0.0.2", "user": "dev", "password": "secret"}
			},
			"tasks": {
				"restart-nginx": [
					{"type": "exec", "run": "nginx -t"},
					{"type": "exec", "run": "systemctl reload nginx"}
				]
			}
		}`), 0644)

		cfg, err := LoadClientConfig(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.Hosts) != 2 {
			t.Errorf("got %d hosts, want 2", len(cfg.Hosts))
		}
		if len(cfg.Tasks) != 1 {
			t.Errorf("got %d tasks, want 1", len(cfg.Tasks))
		}
		if len(cfg.Tasks["restart-nginx"]) != 2 {
			t.Errorf("got %d steps, want 2", len(cfg.Tasks["restart-nginx"]))
		}
	})

	t.Run("config with file task", func(t *testing.T) {
		path := filepath.Join(dir, "file-task.json")
		os.WriteFile(path, []byte(`{
			"hosts": {
				"prod": {"host": "10.0.0.1", "user": "admin", "password": "pass"}
			},
			"tasks": {
				"deploy-config": [
					{"type": "file", "local": "./nginx.conf", "remote": "/etc/nginx/nginx.conf"},
					{"type": "exec", "run": "nginx -t"},
					{"type": "exec", "run": "systemctl reload nginx"}
				]
			}
		}`), 0644)

		cfg, err := LoadClientConfig(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		steps := cfg.Tasks["deploy-config"]
		if steps[0].Type != "file" {
			t.Errorf("got type %q, want %q", steps[0].Type, "file")
		}
		if steps[0].Local != "./nginx.conf" {
			t.Errorf("got local %q, want %q", steps[0].Local, "./nginx.conf")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := LoadClientConfig(filepath.Join(dir, "nope.json"))
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		path := filepath.Join(dir, "bad.json")
		os.WriteFile(path, []byte(`{not json}`), 0644)

		_, err := LoadClientConfig(path)
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})
}

func TestValidateClientConfig(t *testing.T) {
	validHost := map[string]ServerConfig{
		"prod": {Host: "h", User: "u", Key: "k"},
	}
	validTask := map[string][]TaskStep{
		"test": {{Type: "exec", Run: "echo hi"}},
	}

	tests := []struct {
		name    string
		cfg     ClientConfig
		wantErr bool
	}{
		{
			name:    "valid",
			cfg:     ClientConfig{Hosts: validHost, Tasks: validTask},
			wantErr: false,
		},
		{
			name:    "no hosts",
			cfg:     ClientConfig{Hosts: map[string]ServerConfig{}, Tasks: validTask},
			wantErr: true,
		},
		{
			name:    "no tasks",
			cfg:     ClientConfig{Hosts: validHost, Tasks: map[string][]TaskStep{}},
			wantErr: true,
		},
		{
			name: "host missing address",
			cfg: ClientConfig{
				Hosts: map[string]ServerConfig{"prod": {Host: "", User: "u", Key: "k"}},
				Tasks: validTask,
			},
			wantErr: true,
		},
		{
			name: "host missing user",
			cfg: ClientConfig{
				Hosts: map[string]ServerConfig{"prod": {Host: "h", User: "", Key: "k"}},
				Tasks: validTask,
			},
			wantErr: true,
		},
		{
			name: "host missing key and password",
			cfg: ClientConfig{
				Hosts: map[string]ServerConfig{"prod": {Host: "h", User: "u"}},
				Tasks: validTask,
			},
			wantErr: true,
		},
		{
			name: "task with empty steps",
			cfg: ClientConfig{
				Hosts: validHost,
				Tasks: map[string][]TaskStep{"empty": {}},
			},
			wantErr: true,
		},
		{
			name: "file step missing local",
			cfg: ClientConfig{
				Hosts: validHost,
				Tasks: map[string][]TaskStep{
					"bad": {{Type: "file", Local: "", Remote: "/r"}},
				},
			},
			wantErr: true,
		},
		{
			name: "file step missing remote",
			cfg: ClientConfig{
				Hosts: validHost,
				Tasks: map[string][]TaskStep{
					"bad": {{Type: "file", Local: "/l", Remote: ""}},
				},
			},
			wantErr: true,
		},
		{
			name: "exec step missing run",
			cfg: ClientConfig{
				Hosts: validHost,
				Tasks: map[string][]TaskStep{
					"bad": {{Type: "exec", Run: ""}},
				},
			},
			wantErr: true,
		},
		{
			name: "unknown step type",
			cfg: ClientConfig{
				Hosts: validHost,
				Tasks: map[string][]TaskStep{
					"bad": {{Type: "unknown"}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResolveHost(t *testing.T) {
	cfg := &ClientConfig{
		Hosts: map[string]ServerConfig{
			"prod": {Host: "10.0.0.1", User: "admin", Key: "~/.ssh/id_rsa"},
			"dev":  {Host: "10.0.0.2", User: "dev", Password: "secret"},
		},
		Tasks: map[string][]TaskStep{
			"test": {{Type: "exec", Run: "echo"}},
		},
	}

	t.Run("existing alias", func(t *testing.T) {
		host, err := cfg.ResolveHost("prod")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if host.Host != "10.0.0.1" {
			t.Errorf("got host %q, want %q", host.Host, "10.0.0.1")
		}
		if host.User != "admin" {
			t.Errorf("got user %q, want %q", host.User, "admin")
		}
	})

	t.Run("expands home in key", func(t *testing.T) {
		host, err := cfg.ResolveHost("prod")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if host.Key == "~/.ssh/id_rsa" {
			t.Error("expected key path to be expanded, got raw ~/ path")
		}
	})

	t.Run("password host no expansion needed", func(t *testing.T) {
		host, err := cfg.ResolveHost("dev")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if host.Password != "secret" {
			t.Errorf("got password %q, want %q", host.Password, "secret")
		}
	})

	t.Run("unknown alias", func(t *testing.T) {
		_, err := cfg.ResolveHost("staging")
		if err == nil {
			t.Fatal("expected error for unknown alias")
		}
	})
}

func TestResolveTask(t *testing.T) {
	cfg := &ClientConfig{
		Hosts: map[string]ServerConfig{
			"prod": {Host: "h", User: "u", Key: "k"},
		},
		Tasks: map[string][]TaskStep{
			"restart-nginx": {
				{Type: "exec", Run: "nginx -t"},
				{Type: "exec", Run: "systemctl reload nginx"},
			},
		},
	}

	t.Run("existing task", func(t *testing.T) {
		steps, err := cfg.ResolveTask("restart-nginx")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(steps) != 2 {
			t.Errorf("got %d steps, want 2", len(steps))
		}
		if steps[0].Run != "nginx -t" {
			t.Errorf("got run %q, want %q", steps[0].Run, "nginx -t")
		}
	})

	t.Run("unknown task", func(t *testing.T) {
		_, err := cfg.ResolveTask("deploy-backend")
		if err == nil {
			t.Fatal("expected error for unknown task")
		}
	})
}
