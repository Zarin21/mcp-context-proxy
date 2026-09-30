package proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/zarinsubah/mcp-context-proxy/compress"
	"github.com/zarinsubah/mcp-context-proxy/config"
	"github.com/zarinsubah/mcp-context-proxy/pagination"
)

// Server is the MCP proxy server
type Server struct {
	config     *config.ProxyConfig
	engine     *compress.Engine
	pagStore   *pagination.Store
	pagHandler *pagination.Handler
	clients    []*Client
	router     *Router
	stats      []compress.Stats // accumulated stats
}

// NewServer creates a new proxy server
func NewServer(cfg *config.ProxyConfig, engine *compress.Engine, store *pagination.Store, handler *pagination.Handler) *Server {
	return &Server{
		config:     cfg,
		engine:     engine,
		pagStore:   store,
		pagHandler: handler,
		clients:    make([]*Client, 0),
		router:     NewRouter(),
		stats:      make([]compress.Stats, 0),
	}
}

// AddClient adds an upstream MCP server client
func (s *Server) AddClient(client *Client) {
	s.clients = append(s.clients, client)
}

// RunStdio runs the proxy in stdio mode — reads from os.Stdin, writes to os.Stdout
func (s *Server) RunStdio(ctx context.Context) error {
	log.Println("[mcp-context-proxy] proxy running in stdio mode")
	scanner := bufio.NewScanner(os.Stdin)
	buf := make([]byte, 1024*1024) // 1MB buffer
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		msg := scanner.Bytes()
		if len(msg) == 0 {
			continue
		}

		respMsg, err := s.handleMessage(ctx, msg)
		if err != nil {
			log.Printf("[mcp-context-proxy] error handling message: %v\n", err)
			// Send JSON-RPC error response
			var req JSONRPCRequest
			if json.Unmarshal(msg, &req) == nil && req.ID != nil {
				errResp := JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error:   &JSONRPCError{Code: -32603, Message: err.Error()},
				}
				if b, e := json.Marshal(errResp); e == nil {
					fmt.Println(string(b))
				}
			}
			continue
		}

		if respMsg != nil {
			fmt.Println(string(respMsg))
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("error reading stdin: %w", err)
	}
	return nil
}

// handleMessage processes a single JSON-RPC message from the agent
func (s *Server) handleMessage(ctx context.Context, msg json.RawMessage) (json.RawMessage, error) {
	var req JSONRPCRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return nil, fmt.Errorf("invalid json-rpc request: %w", err)
	}

	switch req.Method {
	case "initialize":
		if len(s.clients) > 0 {
			return s.clients[0].Forward(ctx, msg)
		}
		// No upstream servers — return a basic capability response
		result, _ := json.Marshal(map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":   map[string]any{"tools": map[string]any{}},
			"serverInfo":     map[string]any{"name": "mcp-context-proxy", "version": "0.1.0"},
		})
		resp := JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
		return json.Marshal(resp)

	case "notifications/initialized", "initialized":
		// Forward notification, don't expect response
		if len(s.clients) > 0 {
			_ = s.clients[0].transport.Send(msg)
		}
		return nil, nil

	case "tools/list":
		return s.handleToolsListRequest(ctx, &req)

	case "tools/call":
		return s.handleToolsCallRequest(ctx, &req)

	default:
		if len(s.clients) > 0 {
			return s.clients[0].Forward(ctx, msg)
		}
		return nil, fmt.Errorf("no upstream servers configured")
	}
}

// handleToolsListRequest intercepts tools/list to inject fetch_more tool and merge upstream tools
func (s *Server) handleToolsListRequest(ctx context.Context, req *JSONRPCRequest) (json.RawMessage, error) {
	var allTools []json.RawMessage

	for _, client := range s.clients {
		tools, err := client.ListTools(ctx)
		if err == nil {
			allTools = append(allTools, tools...)
		}
	}

	// Add the fetch_more virtual tool
	fetchMoreDef := s.pagHandler.ToolDefinition()
	fetchMoreBytes, _ := json.Marshal(fetchMoreDef)
	allTools = append(allTools, fetchMoreBytes)

	resultBytes, _ := json.Marshal(map[string]any{"tools": allTools})

	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  resultBytes,
	}
	return json.Marshal(resp)
}

// handleToolsCallRequest intercepts tools/call to compress results
func (s *Server) handleToolsCallRequest(ctx context.Context, req *JSONRPCRequest) (json.RawMessage, error) {
	var params MCPToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, fmt.Errorf("invalid tools/call params: %w", err)
	}

	// Handle fetch_more locally
	if params.Name == "fetch_more" {
		resultData, err := s.pagHandler.Handle(params.Arguments)
		if err != nil {
			mcpResult := MCPToolResult{
				Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
				IsError: true,
			}
			resultBytes, _ := json.Marshal(mcpResult)
			resp := JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: resultBytes}
			return json.Marshal(resp)
		}
		mcpResult := MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: string(resultData)}},
		}
		resultBytes, _ := json.Marshal(mcpResult)
		resp := JSONRPCResponse{JSONRPC: "2.0", ID: req.ID, Result: resultBytes}
		return json.Marshal(resp)
	}

	// Forward to upstream server
	if len(s.clients) == 0 {
		return nil, fmt.Errorf("no upstream servers configured")
	}

	reqMsg, _ := json.Marshal(req)
	respMsg, err := s.clients[0].Forward(ctx, reqMsg)
	if err != nil {
		return nil, fmt.Errorf("upstream error: %w", err)
	}

	// Parse the response to compress the result
	var resp JSONRPCResponse
	if err := json.Unmarshal(respMsg, &resp); err != nil {
		return respMsg, nil
	}

	if resp.Error != nil || len(resp.Result) == 0 {
		return respMsg, nil
	}

	// Parse MCP tool result and compress each text content block
	var result MCPToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		// Not a standard MCP result — try compressing the raw result
		compressed, overflow, stats, cErr := s.engine.Compress(params.Name, resp.Result)
		if cErr == nil {
			s.stats = append(s.stats, stats)
			if overflow != nil {
				handleID := s.pagStore.Put(overflow)
				stats.HandleCreated = true
				// Append a note about the handle
				note := fmt.Sprintf("\n\n[Content truncated. Use fetch_more tool with handle_id=%q to retrieve remaining %d bytes]", handleID, len(overflow))
				compressed = append(compressed[:len(compressed)], []byte(note)...)
			}
			resp.Result = compressed
			return json.Marshal(resp)
		}
		return respMsg, nil
	}

	// Compress each text content block
	modified := false
	for i, content := range result.Content {
		if content.Type != "text" || len(content.Text) == 0 {
			continue
		}

		textBytes := []byte(content.Text)
		compressed, overflow, stats, cErr := s.engine.Compress(params.Name, textBytes)
		if cErr != nil {
			continue
		}

		s.stats = append(s.stats, stats)
		newText := string(compressed)

		if overflow != nil {
			handleID := s.pagStore.Put(overflow)
			stats.HandleCreated = true
			newText += fmt.Sprintf("\n\n[Content truncated. Use fetch_more tool with handle_id=%q to retrieve remaining %d bytes]", handleID, len(overflow))
		}

		if newText != content.Text {
			result.Content[i].Text = newText
			modified = true
		}
	}

	if modified {
		resultBytes, _ := json.Marshal(result)
		resp.Result = resultBytes
		return json.Marshal(resp)
	}

	return respMsg, nil
}

// GetStats returns accumulated compression stats
func (s *Server) GetStats() []compress.Stats {
	return s.stats
}
