package vm

import (
	"os"
	"testing"
)

func TestListBackups_EmptyDir(t *testing.T) {
	backups, err := ListBackups()
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = backups
}

func TestFindLatestBackup_NotFound(t *testing.T) {
	_, err := FindLatestBackup("nonexistent-host", "/etc/nonexistent.conf")
	if err == nil {
		t.Fatal("expected error when no backup exists")
	}
}
