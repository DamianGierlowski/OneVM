package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const backupDir = "./backups"

type BackupInfo struct {
	Path      string
	Timestamp time.Time
}

func CreateBackup(transfer *SFTPTransfer, remotePath, host string) (string, error) {
	if !transfer.FileExists(remotePath) {
		return "", nil
	}

	timestamp := time.Now().Format("20060102-150405")
	safeName := strings.ReplaceAll(remotePath, "/", "_")
	backupName := fmt.Sprintf("%s%s_%s", host, safeName, timestamp)
	backupPath := filepath.Join(backupDir, backupName)

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("creating backup directory: %w", err)
	}

	if err := transfer.Download(remotePath, backupPath); err != nil {
		return "", fmt.Errorf("downloading backup of %s: %w", remotePath, err)
	}

	return backupPath, nil
}

func ListBackups() ([]BackupInfo, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading backup directory: %w", err)
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, BackupInfo{
			Path:      filepath.Join(backupDir, entry.Name()),
			Timestamp: info.ModTime(),
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

func findLatestBackup(host, remotePath string) (string, error) {
	backups, err := ListBackups()
	if err != nil {
		return "", err
	}

	safeName := strings.ReplaceAll(remotePath, "/", "_")
	prefix := host + safeName

	for _, b := range backups {
		if strings.HasPrefix(filepath.Base(b.Path), prefix) {
			return b.Path, nil
		}
	}

	return "", fmt.Errorf("no backup found for %s on %s", remotePath, host)
}
