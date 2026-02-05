package vm

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

type SSHClient struct {
	Client *ssh.Client
	Host   string
	User   string
}

type SSHAuth struct {
	KeyPath  string
	Password string
}

func NewSSHClient(host, user string, auth SSHAuth) (*SSHClient, error) {
	var methods []ssh.AuthMethod

	if auth.KeyPath != "" {
		keyData, err := os.ReadFile(auth.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("reading SSH key %s: %w", auth.KeyPath, err)
		}
		signer, err := ssh.ParsePrivateKey(keyData)
		if err != nil {
			return nil, fmt.Errorf("parsing SSH key: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}

	if auth.Password != "" {
		methods = append(methods, ssh.Password(auth.Password))
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no auth method provided (need key or password)")
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            methods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := host
	if !strings.Contains(host, ":") {
		addr = host + ":22"
	}

	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", addr, err)
	}

	return &SSHClient{
		Client: client,
		Host:   host,
		User:   user,
	}, nil
}

func (c *SSHClient) Execute(cmd string) (string, error) {
	session, err := c.Client.NewSession()
	if err != nil {
		return "", fmt.Errorf("creating session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(output), fmt.Errorf("executing %q: %w", cmd, err)
	}

	return strings.TrimSpace(string(output)), nil
}

func (c *SSHClient) Close() error {
	return c.Client.Close()
}
