package vm

import "fmt"

type PushResult struct {
	Server string `json:"server"`
	File   string `json:"file"`
	Status string `json:"status"`
	Backup string `json:"backup,omitempty"`
	Error  string `json:"error,omitempty"`
}

func ExecutePush(cfg *ClientConfig, alias, localPath, remotePath string, dryRun bool) PushResult {
	result := PushResult{
		Server: alias,
		File:   remotePath,
	}

	server, err := cfg.ResolveHost(alias)
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result
	}

	if dryRun {
		result.Status = "dry-run"
		return result
	}

	auth := SSHAuth{KeyPath: server.Key, Password: server.Password}
	client, err := NewSSHClient(server.Host, server.User, auth)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("connection failed: %v", err)
		return result
	}
	defer client.Close()

	transfer, err := NewSFTPTransfer(client)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("SFTP failed: %v", err)
		return result
	}
	defer transfer.Close()

	backupPath, err := CreateBackup(transfer, remotePath, server.Host)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("backup failed (aborting): %v", err)
		return result
	}
	result.Backup = backupPath

	normalized, err := NormalizeFile(localPath)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("normalization failed: %v", err)
		return result
	}

	if err := transfer.UploadBytes(normalized, remotePath); err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("upload failed: %v", err)
		return result
	}

	result.Status = "ok"
	return result
}
