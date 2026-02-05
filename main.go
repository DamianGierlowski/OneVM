package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "ping":
		cmdPing(os.Args[2:])
	case "deploy":
		cmdDeploy(os.Args[2:])
	case "rollback":
		cmdRollback(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: vm-config <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  ping      Test SSH connection to a server")
	fmt.Println("  deploy    Deploy config files from manifest")
	fmt.Println("  rollback  Restore a file from backup")
	fmt.Println()
	fmt.Println("Use 'vm-config <command> --help' for command details.")
}

// --- ping ---

func cmdPing(args []string) {
	fs := flag.NewFlagSet("ping", flag.ExitOnError)
	host := fs.String("host", "", "Server hostname or IP")
	user := fs.String("user", "", "SSH username")
	key := fs.String("key", "", "Path to SSH private key")
	password := fs.String("password", "", "SSH password")
	jsonOut := fs.Bool("json", false, "Output in JSON format")
	fs.Parse(args)

	if *host == "" || *user == "" {
		fmt.Fprintln(os.Stderr, "Error: --host and --user are required")
		fs.Usage()
		os.Exit(1)
	}
	if *key == "" && *password == "" {
		fmt.Fprintln(os.Stderr, "Error: --key or --password is required")
		fs.Usage()
		os.Exit(1)
	}

	client, err := NewSSHClient(*host, *user, SSHAuth{KeyPath: expandHome(*key), Password: *password})
	if err != nil {
		exitError("Connection failed", err, *jsonOut)
	}
	defer client.Close()

	hostname, err := client.Execute("hostname")
	if err != nil {
		exitError("Command failed", err, *jsonOut)
	}

	if *jsonOut {
		outputJSON(map[string]any{
			"success":  true,
			"hostname": hostname,
			"host":     *host,
			"user":     *user,
		})
	} else {
		fmt.Printf("Connected to %s@%s — hostname: %s\n", *user, *host, hostname)
	}
}

// --- deploy ---

type DeployResult struct {
	Server string `json:"server"`
	File   string `json:"file"`
	Status string `json:"status"`
	Backup string `json:"backup,omitempty"`
	Error  string `json:"error,omitempty"`
}

func cmdDeploy(args []string) {
	fs := flag.NewFlagSet("deploy", flag.ExitOnError)
	manifestPath := fs.String("manifest", "", "Path to manifest JSON file")
	dryRun := fs.Bool("dry-run", false, "Show changes without applying")
	jsonOut := fs.Bool("json", false, "Output in JSON format")
	fs.Parse(args)

	if *manifestPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --manifest is required")
		fs.Usage()
		os.Exit(1)
	}

	m, err := LoadManifest(*manifestPath)
	if err != nil {
		exitError("Loading manifest failed", err, *jsonOut)
	}

	results := executeDeploy(m, *dryRun)

	if *jsonOut {
		outputJSON(map[string]any{"results": results})
	} else {
		printDeployResults(results)
	}

	for _, r := range results {
		if r.Status == "error" {
			os.Exit(1)
		}
	}
}

func executeDeploy(m *Manifest, dryRun bool) []DeployResult {
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

		auth := SSHAuth{KeyPath: expandHome(server.Key), Password: server.Password}
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

	// Step 1: Backup existing file (mandatory if exists)
	backupPath, err := CreateBackup(transfer, file.Remote, server.Host)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("backup failed (aborting): %v", err)
		return result
	}
	result.Backup = backupPath

	// Step 2: Normalize local file (CRLF → LF)
	normalized, err := NormalizeFile(file.Local)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("normalization failed: %v", err)
		return result
	}

	// Step 3: Upload normalized content directly
	if err := transfer.UploadBytes(normalized, file.Remote); err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("upload failed: %v", err)
		return result
	}

	// Step 4: Restart service if configured
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

func printDeployResults(results []DeployResult) {
	for _, r := range results {
		switch r.Status {
		case "ok":
			fmt.Printf("[OK]      %s → %s\n", r.File, r.Server)
			if r.Backup != "" {
				fmt.Printf("          backup: %s\n", r.Backup)
			}
		case "dry-run":
			fmt.Printf("[DRY-RUN] %s → %s\n", r.File, r.Server)
		case "warning":
			fmt.Printf("[WARN]    %s → %s: %s\n", r.File, r.Server, r.Error)
		case "error":
			fmt.Printf("[ERROR]   %s → %s: %s\n", r.File, r.Server, r.Error)
		}
	}
}

// --- rollback ---

func cmdRollback(args []string) {
	fs := flag.NewFlagSet("rollback", flag.ExitOnError)
	file := fs.String("file", "", "Remote file path to restore")
	server := fs.String("server", "", "Server in user@host format")
	key := fs.String("key", "", "Path to SSH private key")
	password := fs.String("password", "", "SSH password")
	backup := fs.String("backup", "", "Specific backup file (default: latest)")
	jsonOut := fs.Bool("json", false, "Output in JSON format")
	fs.Parse(args)

	if *file == "" || *server == "" {
		fmt.Fprintln(os.Stderr, "Error: --file and --server are required")
		fs.Usage()
		os.Exit(1)
	}
	if *key == "" && *password == "" {
		fmt.Fprintln(os.Stderr, "Error: --key or --password is required")
		fs.Usage()
		os.Exit(1)
	}

	user, host, err := parseServerString(*server)
	if err != nil {
		exitError("Invalid server format", err, *jsonOut)
	}

	backupPath := *backup
	if backupPath == "" {
		backupPath, err = findLatestBackup(host, *file)
		if err != nil {
			exitError("Finding backup failed", err, *jsonOut)
		}
	}

	client, err := NewSSHClient(host, user, SSHAuth{KeyPath: expandHome(*key), Password: *password})
	if err != nil {
		exitError("Connection failed", err, *jsonOut)
	}
	defer client.Close()

	transfer, err := NewSFTPTransfer(client)
	if err != nil {
		exitError("SFTP failed", err, *jsonOut)
	}
	defer transfer.Close()

	if err := transfer.Upload(backupPath, *file); err != nil {
		exitError("Rollback upload failed", err, *jsonOut)
	}

	if *jsonOut {
		outputJSON(map[string]any{
			"success": true,
			"file":    *file,
			"server":  *server,
			"backup":  backupPath,
		})
	} else {
		fmt.Printf("Restored %s on %s from %s\n", *file, *server, backupPath)
	}
}

// --- helpers ---

func parseServerString(s string) (string, string, error) {
	parts := strings.SplitN(s, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected user@host format, got: %s", s)
	}
	return parts[0], parts[1], nil
}

func exitError(msg string, err error, jsonOut bool) {
	if jsonOut {
		outputJSON(map[string]any{
			"success": false,
			"error":   fmt.Sprintf("%s: %v", msg, err),
		})
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s: %v\n", msg, err)
	}
	os.Exit(1)
}

func outputJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}
