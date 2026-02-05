# OneVM

A lightweight CLI tool for deploying configuration files to remote Linux servers with automatic backups, line-ending normalization, and manifest-based orchestration.

## Problem

Updating config files on remote servers is error-prone and repetitive:

- **No safety net** — overwriting a config without backup leads to "I broke it and can't go back"
- **Cross-platform issues** — Windows line endings (CRLF) silently break Linux services
- **Manual repetition** — each change requires the same sequence: backup, upload, restart

OneVM automates the entire flow into a single command.

## Features

- **Manifest-driven deploy** — define servers and files in a single JSON, deploy everything at once
- **Mandatory backup** — every existing file is backed up locally before overwriting (no backup = operation aborted)
- **CRLF → LF normalization** — automatic line-ending conversion prevents cross-platform issues
- **Service restart** — optional post-deploy commands (e.g. `systemctl reload nginx`)
- **Dry-run mode** — preview what will change without touching anything
- **JSON output** — structured output for CI/CD pipelines and AI agent integration
- **Rollback** — restore any file from a local backup with a single command
- **Single binary** — zero dependencies, cross-compiled for macOS and Windows

## Installation

### From source

```bash
git clone https://github.com/your-username/OneVM.git
cd OneVM
go build -o vm-config ./cmd/vm-config/
```

Requires Go 1.23+.

## Quick Start

### 1. Test connection

```bash
# With SSH key
./vm-config ping --host 192.168.1.10 --user admin --key ~/.ssh/id_rsa

# With password
./vm-config ping --host 192.168.1.10 --user admin --password 'secret'
```

### 2. Create a manifest

```json
{
  "servers": [
    {
      "host": "192.168.1.10",
      "user": "admin",
      "password": "secret"
    }
  ],
  "files": [
    {
      "local": "./configs/nginx.conf",
      "remote": "/etc/nginx/nginx.conf",
      "restart": "systemctl reload nginx"
    }
  ]
}
```

### 3. Deploy

```bash
# Preview changes (no modifications)
./vm-config deploy --manifest servers.json --dry-run

# Deploy for real
./vm-config deploy --manifest servers.json
```

### 4. Rollback if needed

```bash
./vm-config rollback --file /etc/nginx/nginx.conf --server admin@192.168.1.10 --password 'secret'
```

## Usage

### Commands

| Command | Description |
|---------|-------------|
| `ping` | Test SSH connection to a server |
| `deploy` | Deploy config files from a manifest |
| `rollback` | Restore a file from a local backup |

### `ping`

```
vm-config ping --host <ip> --user <user> [--key <path>] [--password <pass>] [--json]
```

### `deploy`

```
vm-config deploy --manifest <path> [--dry-run] [--json]
```

### `rollback`

```
vm-config rollback --file <remote-path> --server <user@host> [--key <path>] [--password <pass>] [--backup <file>] [--json]
```

### Authentication

Supports both SSH key and password authentication. At least one is required:

```json
{"host": "10.0.0.1", "user": "admin", "key": "~/.ssh/id_rsa"}
{"host": "10.0.0.1", "user": "admin", "password": "secret"}
{"host": "10.0.0.1", "user": "admin", "key": "~/.ssh/id_rsa", "password": "fallback"}
```

### Manifest Format

```json
{
  "servers": [
    {
      "host": "string — hostname or IP",
      "user": "string — SSH username",
      "key": "string (optional) — path to SSH private key",
      "password": "string (optional) — SSH password"
    }
  ],
  "files": [
    {
      "local": "string — local file path",
      "remote": "string — remote destination path",
      "restart": "string (optional) — command to run after upload"
    }
  ]
}
```

### JSON Output

All commands support `--json` for structured output:

```bash
./vm-config deploy --manifest servers.json --json
```

```json
{
  "results": [
    {
      "server": "192.168.1.10",
      "file": "/etc/nginx/nginx.conf",
      "status": "ok",
      "backup": "./backups/192.168.1.10_etc_nginx_nginx.conf_20260205-153000"
    }
  ]
}
```

## Deploy Flow

```
For each server:
  Connect SSH → Open SFTP →
    For each file:
      1. Backup remote file to ./backups/ (mandatory)
      2. Read local file and normalize CRLF → LF
      3. Upload normalized content
      4. Run restart command (if configured)
  → Close connections
```

If backup fails, the file is **skipped** — no data is overwritten without a safety copy.

## Project Structure

```
OneVM/
├── cmd/
│   └── vm-config/
│       └── main.go              # CLI entry point
├── internal/
│   └── vm/
│       ├── ssh.go               # SSH client
│       ├── transfer.go          # SFTP upload/download
│       ├── normalize.go         # CRLF → LF conversion
│       ├── backup.go            # Backup management
│       ├── manifest.go          # Manifest parsing and validation
│       └── deploy.go            # Deploy orchestration
├── configs/                     # Config files to deploy
├── go.mod
└── go.sum
```

## Tech Stack

- **Go 1.23** — single binary, cross-compilation
- **golang.org/x/crypto/ssh** — SSH protocol (official Go extension)
- **github.com/pkg/sftp** — SFTP file transfer

## License

MIT
