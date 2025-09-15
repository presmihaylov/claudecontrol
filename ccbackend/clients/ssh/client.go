package ssh

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"golang.org/x/crypto/ssh"
)

// SSHClientInterface defines the interface for SSH operations
type SSHClientInterface interface {
	ExecuteCommand(host, command string) error
}

// SSHClient provides methods for SSH connections and command execution
type SSHClient struct {
	privateKeyBase64  string
	knownHostsContent string
}

// NewSSHClient creates a new SSH client instance
func NewSSHClient(privateKeyBase64, knownHostsContent string) *SSHClient {
	return &SSHClient{
		privateKeyBase64:  privateKeyBase64,
		knownHostsContent: knownHostsContent,
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

	// Create secure host key callback using known hosts content
	if c.knownHostsContent == "" {
		return fmt.Errorf("known hosts content is required for secure SSH connections")
	}

	hostKeyCallback, err := c.createHostKeyCallback(c.knownHostsContent)
	if err != nil {
		return fmt.Errorf("failed to create host key callback: %w", err)
	}

	// Create SSH client config
	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
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

// createHostKeyCallback creates a host key callback from known hosts content
func (c *SSHClient) createHostKeyCallback(knownHostsContent string) (ssh.HostKeyCallback, error) {
	// Parse known hosts content into a map of hostname -> public key
	knownHosts := make(map[string]ssh.PublicKey)

	scanner := bufio.NewScanner(strings.NewReader(knownHostsContent))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse the known_hosts line: hostname keytype key
		parts := strings.Fields(line)
		if len(parts) < 3 {
			return nil, fmt.Errorf("invalid known_hosts line %d: %s", lineNum, line)
		}

		hostname := parts[0]
		keyType := parts[1]
		keyData := parts[2]

		// Parse the public key
		keyLine := fmt.Sprintf("%s %s", keyType, keyData)
		pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(keyLine))
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key on line %d: %w", lineNum, err)
		}

		knownHosts[hostname] = pubKey

		// Also handle IP addresses vs hostnames by storing both
		if net.ParseIP(hostname) != nil {
			// If hostname is an IP, also store it as is
			knownHosts[hostname] = pubKey
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading known hosts content: %w", err)
	}

	// Return a callback function that verifies against known hosts
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// Check both the hostname and the IP address from remote
		hostsToCheck := []string{hostname}

		// Extract IP from remote address
		if tcpAddr, ok := remote.(*net.TCPAddr); ok {
			hostsToCheck = append(hostsToCheck, tcpAddr.IP.String())
		}

		for _, hostToCheck := range hostsToCheck {
			if expectedKey, exists := knownHosts[hostToCheck]; exists {
				if ssh.KeysEqual(key, expectedKey) {
					log.Printf("📋 Host key verified for %s", hostToCheck)
					return nil
				}
			}
		}

		return fmt.Errorf("host key verification failed for %s - key not found in known hosts", hostname)
	}, nil
}
