package vm

import (
	"encoding/json"
	"fmt"
	"os"
)

type ClientConfig struct {
	Hosts map[string]ServerConfig `json:"hosts"`
	Tasks map[string][]TaskStep   `json:"tasks"`
}

type TaskStep struct {
	Type   string `json:"type"`
	Local  string `json:"local,omitempty"`
	Remote string `json:"remote,omitempty"`
	Run    string `json:"run,omitempty"`
}

func LoadClientConfig(path string) (*ClientConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg ClientConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *ClientConfig) Validate() error {
	if len(c.Hosts) == 0 {
		return fmt.Errorf("config: no hosts defined")
	}
	if len(c.Tasks) == 0 {
		return fmt.Errorf("config: no tasks defined")
	}

	for name, host := range c.Hosts {
		if host.Host == "" {
			return fmt.Errorf("config: host %q missing host address", name)
		}
		if host.User == "" {
			return fmt.Errorf("config: host %q missing user", name)
		}
		if host.Key == "" && host.Password == "" {
			return fmt.Errorf("config: host %q missing key or password", name)
		}
	}

	for name, steps := range c.Tasks {
		if len(steps) == 0 {
			return fmt.Errorf("config: task %q has no steps", name)
		}
		for i, step := range steps {
			switch step.Type {
			case "file":
				if step.Local == "" {
					return fmt.Errorf("config: task %q step[%d] missing local path", name, i)
				}
				if step.Remote == "" {
					return fmt.Errorf("config: task %q step[%d] missing remote path", name, i)
				}
			case "exec":
				if step.Run == "" {
					return fmt.Errorf("config: task %q step[%d] missing run command", name, i)
				}
			default:
				return fmt.Errorf("config: task %q step[%d] unknown type %q", name, i, step.Type)
			}
		}
	}

	return nil
}

func (c *ClientConfig) ResolveHost(alias string) (ServerConfig, error) {
	host, ok := c.Hosts[alias]
	if !ok {
		return ServerConfig{}, fmt.Errorf("unknown host alias: %s", alias)
	}
	host.Key = ExpandHome(host.Key)
	return host, nil
}

func (c *ClientConfig) ResolveTask(name string) ([]TaskStep, error) {
	steps, ok := c.Tasks[name]
	if !ok {
		return nil, fmt.Errorf("unknown task: %s", name)
	}
	return steps, nil
}
