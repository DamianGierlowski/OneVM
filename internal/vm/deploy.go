package vm

import "fmt"

type DeployResult struct {
	Server string `json:"server"`
	File   string `json:"file"`
	Status string `json:"status"`
	Backup string `json:"backup,omitempty"`
	Error  string `json:"error,omitempty"`
}

func ExecuteDeploy(m *Manifest, dryRun bool) []DeployResult {
	var results []DeployResult

	for _, server := range m.Servers {
		if dryRun {
			for _, file := range m.Files {
				results = append(results, DeployResult{
					Server: server.Host,
					File:   file.Remote,
					Status: "dry-run",
				})
			}
			continue
		}

		auth := SSHAuth{KeyPath: ExpandHome(server.Key), Password: server.Password}
		client, err := NewSSHClient(server.Host, server.User, auth)
		if err != nil {
			for _, file := range m.Files {
				results = append(results, DeployResult{
					Server: server.Host,
					File:   file.Remote,
					Status: "error",
					Error:  fmt.Sprintf("connection failed: %v", err),
				})
			}
			continue
		}

		transfer, err := NewSFTPTransfer(client)
		if err != nil {
			client.Close()
			for _, file := range m.Files {
				results = append(results, DeployResult{
					Server: server.Host,
					File:   file.Remote,
					Status: "error",
					Error:  fmt.Sprintf("SFTP failed: %v", err),
				})
			}
			continue
		}

		for _, file := range m.Files {
			result := deploySingleFile(client, transfer, server, file)
			results = append(results, result)
		}

		transfer.Close()
		client.Close()
	}

	return results
}

func deploySingleFile(client *SSHClient, transfer *SFTPTransfer, server ServerConfig, file FileConfig) DeployResult {
	result := DeployResult{
		Server: server.Host,
		File:   file.Remote,
	}

	backupPath, err := CreateBackup(transfer, file.Remote, server.Host)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("backup failed (aborting): %v", err)
		return result
	}
	result.Backup = backupPath

	normalized, err := NormalizeFile(file.Local)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("normalization failed: %v", err)
		return result
	}

	if err := transfer.UploadBytes(normalized, file.Remote); err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("upload failed: %v", err)
		return result
	}

	if file.Restart != "" {
		output, err := client.Execute(file.Restart)
		if err != nil {
			result.Status = "warning"
			result.Error = fmt.Sprintf("restart failed: %v (output: %s)", err, output)
			return result
		}
	}

	result.Status = "ok"
	return result
}
