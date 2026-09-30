package proxy

import (
	"context"
	"encoding/json"
	"sync"
)

// Transport is the interface for MCP server communication
type Transport interface {
	Send(msg json.RawMessage) error
	Receive() (json.RawMessage, error)
	Close() error
}

// Client represents a connection to an upstream MCP server
type Client struct {
	name      string
	transport Transport
	tools     []json.RawMessage // cached tool definitions
	mu        sync.Mutex
	requests  map[string]chan json.RawMessage
}

// NewClient creates a client connected to the given transport
func NewClient(name string, transport Transport) *Client {
	c := &Client{
		name:      name,
		transport: transport,
		requests:  make(map[string]chan json.RawMessage),
	}
	go c.listen()
	return c
}

func (c *Client) listen() {
	for {
		msg, err := c.transport.Receive()
		if err != nil {
			return
		}
		
		var resp JSONRPCResponse
		if err := json.Unmarshal(msg, &resp); err == nil && resp.ID != nil {
			b, _ := json.Marshal(resp.ID)
			idStr := string(b)
			c.mu.Lock()
			if ch, ok := c.requests[idStr]; ok {
				ch <- msg
				delete(c.requests, idStr)
			}
			c.mu.Unlock()
		}
	}
}

// Initialize sends the MCP initialize handshake
func (c *Client) Initialize(ctx context.Context) error {
	reqBody := `{"jsonrpc":"2.0","id":"init-req","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"mcp-context-proxy","version":"0.1.0"}}}`
	
	_, err := c.Forward(ctx, json.RawMessage(reqBody))
	if err != nil {
		return err
	}
	notifBody := `{"jsonrpc":"2.0","method":"initialized"}`
	return c.transport.Send(json.RawMessage(notifBody))
}

// ListTools returns the tools offered by this server
func (c *Client) ListTools(ctx context.Context) ([]json.RawMessage, error) {
	reqBody := `{"jsonrpc":"2.0","id":"list-tools-req","method":"tools/list"}`
	respMsg, err := c.Forward(ctx, json.RawMessage(reqBody))
	if err != nil {
		return nil, err
	}

	var resp JSONRPCResponse
	if err := json.Unmarshal(respMsg, &resp); err != nil {
		return nil, err
	}

	var result struct {
		Tools []json.RawMessage `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}

	c.tools = result.Tools
	return c.tools, nil
}

// Forward sends a request and returns the response
func (c *Client) Forward(ctx context.Context, req json.RawMessage) (json.RawMessage, error) {
	var rpcReq JSONRPCRequest
	if err := json.Unmarshal(req, &rpcReq); err != nil {
		return nil, err
	}

	idBytes, _ := json.Marshal(rpcReq.ID)
	idStr := string(idBytes)

	ch := make(chan json.RawMessage, 1)
	c.mu.Lock()
	c.requests[idStr] = ch
	c.mu.Unlock()

	if err := c.transport.Send(req); err != nil {
		c.mu.Lock()
		delete(c.requests, idStr)
		c.mu.Unlock()
		return nil, err
	}

	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.requests, idStr)
		c.mu.Unlock()
		return nil, ctx.Err()
	case resp := <-ch:
		return resp, nil
	}
}

// Close shuts down the client
func (c *Client) Close() error {
	return c.transport.Close()
}
