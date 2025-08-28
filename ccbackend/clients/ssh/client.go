package ssh

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"

	"golang.org/x/crypto/ssh"
)

// SSHClientInterface defines the interface for SSH operations
type SSHClientInterface interface {
	ExecuteCommand(host, command string) error
}

// SSHClient provides methods for SSH connections and command execution
type SSHClient struct {
	privateKeyBase64 string
}

// NewSSHClient creates a new SSH client instance
func NewSSHClient(privateKeyBase64 string) *SSHClient {
	return &SSHClient{
		privateKeyBase64: privateKeyBase64,
	}
}

// ExecuteCommand executes a command on the remote server via SSH
func (c *SSHClient) ExecuteCommand(host, command string) error {
	log.Printf("📋 Starting SSH command execution on host: %s", host)

	// Decode the base64 private key
	privateKeyBytes, err := base64.StdEncoding.DecodeString(c.privateKeyBase64)
	if err != nil {
		return fmt.Errorf("failed to decode SSH private key: %w", err)
	}

	// Parse the private key
	signer, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse SSH private key: %w", err)
	}

	// Create SSH client config
	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Note: In production, should verify host keys
	}

	// Connect to the SSH server
	addr := fmt.Sprintf("%s:22", host)
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server %s: %w", addr, err)
	}
	defer conn.Close()

	// Create a session
	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Capture stdout and stderr
	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	log.Printf("📋 Executing command: %s", command)
	if err := session.Start(command); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Read stdout and stderr
	go func() {
		if output, err := io.ReadAll(stdout); err == nil && len(output) > 0 {
			log.Printf("📋 SSH stdout: %s", string(output))
		}
	}()

	go func() {
		if output, err := io.ReadAll(stderr); err == nil && len(output) > 0 {
			log.Printf("❌ SSH stderr: %s", string(output))
		}
	}()

	// Wait for the command to finish
	if err := session.Wait(); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	log.Printf("📋 SSH command completed successfully")
	return nil
}