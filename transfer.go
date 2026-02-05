package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
)

type SFTPTransfer struct {
	client *sftp.Client
}

func NewSFTPTransfer(sshClient *SSHClient) (*SFTPTransfer, error) {
	client, err := sftp.NewClient(sshClient.client)
	if err != nil {
		return nil, fmt.Errorf("creating SFTP client: %w", err)
	}

	return &SFTPTransfer{client: client}, nil
}

func (t *SFTPTransfer) Upload(localPath, remotePath string) error {
	local, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("opening local file %s: %w", localPath, err)
	}
	defer local.Close()

	remote, err := t.client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("creating remote file %s: %w", remotePath, err)
	}
	defer remote.Close()

	if _, err = io.Copy(remote, local); err != nil {
		return fmt.Errorf("uploading to %s: %w", remotePath, err)
	}

	return nil
}

func (t *SFTPTransfer) UploadBytes(data []byte, remotePath string) error {
	remote, err := t.client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("creating remote file %s: %w", remotePath, err)
	}
	defer remote.Close()

	if _, err = remote.Write(data); err != nil {
		return fmt.Errorf("writing to %s: %w", remotePath, err)
	}

	return nil
}

func (t *SFTPTransfer) Download(remotePath, localPath string) error {
	remote, err := t.client.Open(remotePath)
	if err != nil {
		return fmt.Errorf("opening remote file %s: %w", remotePath, err)
	}
	defer remote.Close()

	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	local, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("creating local file %s: %w", localPath, err)
	}
	defer local.Close()

	if _, err = io.Copy(local, remote); err != nil {
		return fmt.Errorf("downloading %s: %w", remotePath, err)
	}

	return nil
}

func (t *SFTPTransfer) FileExists(remotePath string) bool {
	_, err := t.client.Stat(remotePath)
	return err == nil
}

func (t *SFTPTransfer) Close() error {
	return t.client.Close()
}
