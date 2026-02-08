package vm

import "fmt"

type StepResult struct {
	Step   string `json:"step"`
	Status string `json:"status"`
	Backup string `json:"backup,omitempty"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

type RunResult struct {
	Server string       `json:"server"`
	Task   string       `json:"task"`
	Steps  []StepResult `json:"steps"`
	Status string       `json:"status"`
}

func ExecuteRun(cfg *ClientConfig, taskName string, aliases []string, dryRun bool) []RunResult {
	var results []RunResult

	steps, err := cfg.ResolveTask(taskName)
	if err != nil {
		for _, alias := range aliases {
			results = append(results, RunResult{
				Server: alias,
				Task:   taskName,
				Status: "error",
				Steps: []StepResult{{
					Step:   "resolve",
					Status: "error",
					Error:  err.Error(),
				}},
			})
		}
		return results
	}

	for _, alias := range aliases {
		result := executeRunOnServer(cfg, alias, taskName, steps, dryRun)
		results = append(results, result)
	}

	return results
}

func executeRunOnServer(cfg *ClientConfig, alias, taskName string, steps []TaskStep, dryRun bool) RunResult {
	result := RunResult{
		Server: alias,
		Task:   taskName,
	}

	server, err := cfg.ResolveHost(alias)
	if err != nil {
		result.Status = "error"
		result.Steps = []StepResult{{
			Step:   "resolve",
			Status: "error",
			Error:  err.Error(),
		}}
		return result
	}

	if dryRun {
		for _, step := range steps {
			result.Steps = append(result.Steps, StepResult{
				Step:   stepLabel(step),
				Status: "dry-run",
			})
		}
		result.Status = "dry-run"
		return result
	}

	auth := SSHAuth{KeyPath: server.Key, Password: server.Password}
	client, err := NewSSHClient(server.Host, server.User, auth)
	if err != nil {
		result.Status = "error"
		result.Steps = []StepResult{{
			Step:   "connect",
			Status: "error",
			Error:  fmt.Sprintf("connection failed: %v", err),
		}}
		return result
	}
	defer client.Close()

	var transfer *SFTPTransfer
	needsSFTP := false
	for _, step := range steps {
		if step.Type == "file" {
			needsSFTP = true
			break
		}
	}

	if needsSFTP {
		transfer, err = NewSFTPTransfer(client)
		if err != nil {
			result.Status = "error"
			result.Steps = []StepResult{{
				Step:   "sftp",
				Status: "error",
				Error:  fmt.Sprintf("SFTP failed: %v", err),
			}}
			return result
		}
		defer transfer.Close()
	}

	allOK := true
	for _, step := range steps {
		var stepResult StepResult

		switch step.Type {
		case "file":
			stepResult = executeFileStep(transfer, step, server.Host)
		case "exec":
			stepResult = executeExecStep(client, step)
		}

		result.Steps = append(result.Steps, stepResult)

		if stepResult.Status == "error" {
			allOK = false
			break
		}
	}

	if allOK {
		result.Status = "ok"
	} else {
		result.Status = "error"
	}

	return result
}

func executeFileStep(transfer *SFTPTransfer, step TaskStep, host string) StepResult {
	sr := StepResult{Step: stepLabel(step)}

	backupPath, err := CreateBackup(transfer, step.Remote, host)
	if err != nil {
		sr.Status = "error"
		sr.Error = fmt.Sprintf("backup failed (aborting): %v", err)
		return sr
	}
	sr.Backup = backupPath

	normalized, err := NormalizeFile(step.Local)
	if err != nil {
		sr.Status = "error"
		sr.Error = fmt.Sprintf("normalization failed: %v", err)
		return sr
	}

	if err := transfer.UploadBytes(normalized, step.Remote); err != nil {
		sr.Status = "error"
		sr.Error = fmt.Sprintf("upload failed: %v", err)
		return sr
	}

	sr.Status = "ok"
	return sr
}

func executeExecStep(client *SSHClient, step TaskStep) StepResult {
	sr := StepResult{Step: stepLabel(step)}

	output, err := client.Execute(step.Run)
	sr.Output = output

	if err != nil {
		sr.Status = "error"
		sr.Error = fmt.Sprintf("command failed: %v", err)
		return sr
	}

	sr.Status = "ok"
	return sr
}

func stepLabel(step TaskStep) string {
	switch step.Type {
	case "file":
		return fmt.Sprintf("file:%s", step.Remote)
	case "exec":
		return fmt.Sprintf("exec:%s", step.Run)
	default:
		return step.Type
	}
}
