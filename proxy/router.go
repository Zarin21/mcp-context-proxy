package proxy

import "encoding/json"

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError is a JSON-RPC error object
type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// MCPToolResult represents the result of an MCP tools/call
type MCPToolResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
	Meta    any          `json:"_meta,omitempty"` // For _context_proxy metadata
}

// MCPContent is a content block in an MCP result
type MCPContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// MCPToolCallParams represents the params of a tools/call request
type MCPToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// Router handles routing and interception of JSON-RPC messages
type Router struct {
	// pending tracks request IDs to their tool call info for result interception
	pending map[string]*pendingCall
}

type pendingCall struct {
	ToolName string
	Request  *JSONRPCRequest
}

// NewRouter creates a new router
func NewRouter() *Router {
	return &Router{
		pending: make(map[string]*pendingCall),
	}
}

// TrackRequest tracks an outgoing tools/call request by its ID
func (r *Router) TrackRequest(req *JSONRPCRequest) {
	if req.ID == nil {
		return
	}
	idStr, err := idToString(req.ID)
	if err != nil {
		return
	}

	var params MCPToolCallParams
	if err := json.Unmarshal(req.Params, &params); err == nil {
		r.pending[idStr] = &pendingCall{
			ToolName: params.Name,
			Request:  req,
		}
	}
}

// MatchResponse flags a response for compression if its ID matches a tracked request
func (r *Router) MatchResponse(resp *JSONRPCResponse) (toolName string, matched bool) {
	if resp.ID == nil {
		return "", false
	}
	idStr, err := idToString(resp.ID)
	if err != nil {
		return "", false
	}

	if call, ok := r.pending[idStr]; ok {
		return call.ToolName, true
	}
	return "", false
}

// Untrack removes a tracked request by its ID
func (r *Router) Untrack(id string) {
	delete(r.pending, id)
}

func idToString(id any) (string, error) {
	b, err := json.Marshal(id)
	return string(b), err
}
