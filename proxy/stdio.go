package proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
)

// StdioTransport manages communication with an MCP server via stdio
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	stderr io.ReadCloser
}

// NewStdioTransport creates and starts a stdio MCP server process
func NewStdioTransport(ctx context.Context, command string, args []string) (*StdioTransport, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	// Read stderr in background
	go func() {
		io.Copy(io.Discard, stderr)
	}()

	return &StdioTransport{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewScanner(stdout),
		stderr: stderr,
	}, nil
}

// Send sends a JSON-RPC message to the server
func (t *StdioTransport) Send(msg json.RawMessage) error {
	_, err := t.stdin.Write(append(msg, '\n'))
	return err
}

// Receive reads the next JSON-RPC message from the server (blocking)
func (t *StdioTransport) Receive() (json.RawMessage, error) {
	if t.stdout.Scan() {
		return append(json.RawMessage{}, t.stdout.Bytes()...), nil
	}
	if err := t.stdout.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

// Close terminates the server process
func (t *StdioTransport) Close() error {
	t.stdin.Close()
	return t.cmd.Process.Kill()
}
