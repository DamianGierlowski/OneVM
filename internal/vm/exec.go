package vm

import "fmt"

type ExecResult struct {
	Server string `json:"server"`
	Status string `json:"status"`
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

func ExecuteExec(cfg *ClientConfig, aliases []string, command string) []ExecResult {
	var results []ExecResult

	for _, alias := range aliases {
		result := ExecResult{Server: alias}

		server, err := cfg.ResolveHost(alias)
		if err != nil {
			result.Status = "error"
			result.Error = err.Error()
			results = append(results, result)
			continue
		}

		auth := SSHAuth{KeyPath: server.Key, Password: server.Password}
		client, err := NewSSHClient(server.Host, server.User, auth)
		if err != nil {
			result.Status = "error"
			result.Error = fmt.Sprintf("connection failed: %v", err)
			results = append(results, result)
			continue
		}

		output, err := client.Execute(command)
		client.Close()

		result.Output = output
		if err != nil {
			result.Status = "error"
			result.Error = fmt.Sprintf("command failed: %v", err)
		} else {
			result.Status = "ok"
		}

		results = append(results, result)
	}

	return results
}
