package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// HTTPTransport manages communication with an HTTP MCP server
type HTTPTransport struct {
	baseURL    string
	client     *http.Client
	responseCh chan json.RawMessage
}

// NewHTTPTransport creates an HTTP transport to the given MCP server URL
func NewHTTPTransport(ctx context.Context, baseURL string) (*HTTPTransport, error) {
	return &HTTPTransport{
		baseURL:    baseURL,
		client:     &http.Client{},
		responseCh: make(chan json.RawMessage, 10), // Buffered
	}, nil
}

// Send sends a JSON-RPC request via HTTP POST
func (t *HTTPTransport) Send(msg json.RawMessage) error {
	req, err := http.NewRequest("POST", t.baseURL, bytes.NewReader(msg))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	t.responseCh <- json.RawMessage(body)
	return nil
}

// Receive returns the next response (blocking)
func (t *HTTPTransport) Receive() (json.RawMessage, error) {
	resp, ok := <-t.responseCh
	if !ok {
		return nil, fmt.Errorf("transport closed")
	}
	return resp, nil
}

// Close cleans up the transport
func (t *HTTPTransport) Close() error {
	close(t.responseCh)
	return nil
}
