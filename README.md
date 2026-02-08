# OneVM

A lightweight CLI tool for managing remote Linux servers — deploy files, execute commands, and run named tasks with automatic backups and manifest-based orchestration.

## Problem

Managing multiple servers across clients without CI/CD means repetitive, error-prone manual work:

- **SSH into each server, one by one** — the same commands repeated across dev, rc, prod
- **No safety net** — overwriting a config without backup leads to "I broke it and can't go back"
- **Cross-platform issues** — Windows line endings (CRLF) silently break Linux services
- **Scattered credentials** — remembering IPs, users, and passwords for 15+ servers
- **No reusable operations** — the same deploy sequence typed out by hand every time

OneVM gives you named servers, named tasks, and a single command to run them.

## Features

- **Client config file** — one JSON per client with named servers and named tasks
- **`run` command** — execute a named task on any server: `onevm run restart-nginx prod`
- **`push` command** — upload a file ad-hoc: `onevm push ./fix.conf prod:/etc/app.conf`
- **`exec` command** — run a command ad-hoc: `onevm exec prod -- 'systemctl status app'`
- **Mandatory backup** — every file is backed up before overwriting (no backup = abort)
- **CRLF → LF normalization** — automatic line-ending conversion prevents cross-platform issues
- **Dry-run mode** — preview what will change without touching anything
- **JSON output** — structured output for scripting and automation
- **Rollback** — restore any file from a local backup with a single command
- **Backward compatible** — v1 manifest `deploy` still works unchanged
- **Single binary** — zero dependencies, cross-compiled for macOS and Windows

## Installation

```bash
git clone https://github.com/DamianGierlowski/OneVM.git
cd OneVM
go build -o onevm ./cmd/onevm/
```

Requires Go 1.23+.

## Quick Start

### 1. Create a client config

Create `onevm.json` (or any name with `--config`):

```json
{
  "hosts": {
    "dev": {
      "host": "192.168.1.30",
      "user": "dev",
      "key": "~/.ssh/id_rsa"
    },
    "prod": {
      "host": "192.168.1.10",
      "user": "admin",
      "password": "secret"
    }
  },
  "tasks": {
    "restart-nginx": [
      { "type": "exec", "run": "nginx -t" },
      { "type": "exec", "run": "systemctl reload nginx" }
    ],
    "update-backend": [
      { "type": "exec", "run": "cd /var/app && git pull origin main" },
      { "type": "exec", "run": "cd /var/app && npm install --production" },
      { "type": "exec", "run": "systemctl restart app" }
    ],
    "deploy-config": [
      { "type": "file", "local": "./nginx.conf", "remote": "/etc/nginx/nginx.conf" },
      { "type": "exec", "run": "nginx -t" },
      { "type": "exec", "run": "systemctl reload nginx" }
    ]
  }
}
```

### 2. Test connection

```bash
./onevm ping prod
./onevm ping dev prod
```

### 3. Run a task

```bash
# Preview first
./onevm run --dry-run restart-nginx prod

# Execute
./onevm run restart-nginx prod

# On multiple servers
./onevm run update-backend dev prod
```

### 4. Ad-hoc operations

```bash
# Push a file
./onevm push ./hotfix.conf prod:/etc/app/app.conf

# Execute a command
./onevm exec prod -- 'systemctl status nginx'
./onevm exec dev prod -- 'hostname'
```

### 5. Rollback if needed

```bash
./onevm rollback --file /etc/nginx/nginx.conf --server prod
```

## Usage

### Commands

| Command | Description | Example |
|---------|-------------|---------|
| `run` | Execute a named task on servers | `onevm run restart-nginx prod` |
| `push` | Upload a file to a server (ad-hoc) | `onevm push ./f.conf prod:/etc/f.conf` |
| `exec` | Execute a command on servers (ad-hoc) | `onevm exec prod -- 'hostname'` |
| `ping` | Test SSH connection | `onevm ping prod` |
| `deploy` | Deploy from v1 manifest | `onevm deploy --manifest servers.json` |
| `rollback` | Restore a file from backup | `onevm rollback --file /etc/f.conf --server prod` |

### `run`

Execute a named task from the config file on one or more servers.

```
onevm run [flags] <task-name> <server...>
```

Flags must come **before** positional arguments (Go `flag` standard behavior).

| Flag | Description | Default |
|------|-------------|---------|
| `--config` | Path to client config file | `./onevm.json` |
| `--dry-run` | Preview without executing | `false` |
| `--json` | JSON output | `false` |

```bash
./onevm run restart-nginx prod
./onevm run --dry-run update-backend dev prod
./onevm run --config clients/acme.json deploy-config prod
```

Output:

```
[prod] restart-nginx
  ✓ exec:nginx -t
  ✓ exec:systemctl reload nginx
```

### `push`

Upload a single file with mandatory backup and CRLF normalization.

```
onevm push [flags] <local-path> <alias>:<remote-path>
```

| Flag | Description | Default |
|------|-------------|---------|
| `--config` | Path to client config file | `./onevm.json` |
| `--dry-run` | Preview without executing | `false` |
| `--json` | JSON output | `false` |

```bash
./onevm push ./nginx.conf prod:/etc/nginx/nginx.conf
./onevm push --config clients/acme.json ./fix.conf rc:/etc/app.conf
```

### `exec`

Run an ad-hoc command on one or more servers. Use `--` to separate aliases from the command.

```
onevm exec [flags] <alias...> -- <command>
```

| Flag | Description | Default |
|------|-------------|---------|
| `--config` | Path to client config file | `./onevm.json` |
| `--json` | JSON output | `false` |

```bash
./onevm exec prod -- 'systemctl status nginx'
./onevm exec dev prod -- 'hostname'
./onevm exec prod -- 'cd /var/app && git pull && systemctl restart app'
```

Output:

```
[prod] OK
nginx.service - active (running)

[dev] OK
dev-server-01
```

### `ping`

Test SSH connection. Supports both aliases and explicit flags.

```bash
# With aliases (v2)
./onevm ping prod
./onevm ping dev rc prod

# With explicit flags (v1)
./onevm ping --host 192.168.1.10 --user admin --password 'secret'
```

### `deploy` (v1 compat)

Deploy from a v1 manifest file. Works exactly as before — no config file needed.

```bash
./onevm deploy --manifest servers.json --dry-run
./onevm deploy --manifest servers.json
```

### `rollback`

Restore a file from local backup. Supports both aliases and explicit flags.

```bash
# With alias (v2)
./onevm rollback --file /etc/nginx/nginx.conf --server prod

# With explicit flags (v1)
./onevm rollback --file /etc/nginx/nginx.conf --server admin@192.168.1.10 --password 'secret'
```

## Client Config Format

One JSON file per client. Contains **hosts** (where) and **tasks** (what).

```json
{
  "hosts": {
    "<alias>": {
      "host": "string — hostname or IP",
      "user": "string — SSH username",
      "key": "string (optional) — path to SSH private key",
      "password": "string (optional) — SSH password"
    }
  },
  "tasks": {
    "<task-name>": [
      { "type": "file", "local": "./src", "remote": "/dest" },
      { "type": "exec", "run": "command to execute" }
    ]
  }
}
```

### Task steps

| Type | Fields | Description |
|------|--------|-------------|
| `file` | `local`, `remote` | Upload file with backup + CRLF normalization |
| `exec` | `run` | Execute command via SSH |

Steps run **in order**. If any step fails, remaining steps on that server are skipped (fail-fast). Other servers continue independently.

### Per-environment differences

If paths or commands differ between environments, create separate tasks. No templates, no magic:

```json
{
  "tasks": {
    "update-backend": [
      { "type": "exec", "run": "cd /var/app && git pull origin main" }
    ],
    "update-backend-dev": [
      { "type": "exec", "run": "cd /home/dev/app && git pull origin develop" }
    ]
  }
}
```

### Multi-client setup

Keep configs per client in a `clients/` directory:

```
clients/
├── acme-corp.json
├── megashop.json
└── startupxyz.json
```

```bash
./onevm run update-backend prod --config clients/acme-corp.json
./onevm run full-release prod-web --config clients/megashop.json
./onevm exec prod -- 'hostname' --config clients/startupxyz.json
```

## JSON Output

All commands support `--json` for structured output:

```bash
./onevm run --json deploy-config prod
```

```json
{
  "results": [
    {
      "server": "prod",
      "task": "deploy-config",
      "steps": [
        {
          "step": "file:/etc/nginx/nginx.conf",
          "status": "ok",
          "backup": "backups/192.168.1.10_etc_nginx_nginx.conf_20260205-153000"
        },
        { "step": "exec:nginx -t", "status": "ok" },
        { "step": "exec:systemctl reload nginx", "status": "ok" }
      ],
      "status": "ok"
    }
  ]
}
```

## Backup & Rollback

Every file upload (via `run`, `push`, or `deploy`) creates a mandatory backup:

```
./backups/{host}_{sanitized_path}_{timestamp}
./backups/192.168.1.10_etc_nginx_nginx.conf_20260205-153000
```

If backup fails, the upload is **aborted** — no data is overwritten without a safety copy.

Rollback restores from the latest backup automatically:

```bash
./onevm rollback --file /etc/nginx/nginx.conf --server prod
```

## Project Structure

```
OneVM/
├── cmd/onevm/
│   └── main.go            # CLI entry point
├── internal/vm/
│   ├── config.go           # Client config (hosts + tasks)
│   ├── run.go              # Run task orchestration
│   ├── exec.go             # Ad-hoc command execution
│   ├── push.go             # Ad-hoc file upload
│   ├── ssh.go              # SSH client
│   ├── transfer.go         # SFTP upload/download
│   ├── backup.go           # Backup management
│   ├── normalize.go        # CRLF → LF conversion
│   ├── manifest.go         # v1 manifest parsing
│   └── deploy.go           # v1 deploy orchestration
├── clients/                # Client config files
├── configs/                # Config files to deploy
└── backups/                # Automatic backups
```

## Tech Stack

- **Go 1.23** — single binary, cross-compilation
- **golang.org/x/crypto/ssh** — SSH protocol
- **github.com/pkg/sftp** — SFTP file transfer

## License

MIT
