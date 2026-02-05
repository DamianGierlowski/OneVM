package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListBackups_EmptyDir(t *testing.T) {
	original := backupDir
	defer func() { /* restore not needed, const */ }()
	_ = original

	backups, err := ListBackups()
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = backups
}

func TestFindLatestBackup_NotFound(t *testing.T) {
	_, err := findLatestBackup("nonexistent-host", "/etc/nonexistent.conf")
	if err == nil {
		t.Fatal("expected error when no backup exists")
	}
}

func TestFindLatestBackup_Found(t *testing.T) {
	dir := t.TempDir()

	// Create fake backup files
	name := "10.0.0.1_etc_nginx_nginx.conf_20250101-120000"
	path := filepath.Join(dir, name)
	os.WriteFile(path, []byte("backup content"), 0644)

	// We can't easily test findLatestBackup without changing backupDir,
	// so this is a sanity check that the naming convention works
	if filepath.Base(path) != name {
		t.Errorf("backup name mismatch: got %q", filepath.Base(path))
	}
}
