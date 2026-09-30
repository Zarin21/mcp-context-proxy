package pagination

import (
	"encoding/json"
	"fmt"
)

// FetchMoreRequest is the argument schema for the fetch_more tool
type FetchMoreRequest struct {
	HandleID string `json:"handle_id"`
	Offset   int    `json:"offset"` // byte offset (default 0)
	Limit    int    `json:"limit"`  // max bytes to return (default 4096)
}

// FetchMoreResponse is returned by the fetch_more tool
type FetchMoreResponse struct {
	Content   json.RawMessage `json:"content"`
	Offset    int             `json:"offset"`
	Limit     int             `json:"limit"`
	TotalSize int             `json:"total_size"`
	HasMore   bool            `json:"has_more"`
}

// Handler implements the fetch_more virtual tool
type Handler struct {
	store *Store
}

// NewHandler creates a fetch_more handler backed by the given store
func NewHandler(store *Store) *Handler {
	return &Handler{
		store: store,
	}
}

// ToolDefinition returns the MCP tool definition for fetch_more.
// Returns a map matching the MCP tool schema: name, description, inputSchema.
func (h *Handler) ToolDefinition() map[string]any {
	return map[string]any{
		"name":        "fetch_more",
		"description": "Fetch deferred/truncated content from a previous tool call by handle ID. Use this when a tool result indicates content was truncated and provides a handle_id.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"handle_id": map[string]any{
					"type":        "string",
					"description": "Handle ID of the deferred content",
				},
				"offset": map[string]any{
					"type":        "integer",
					"description": "Byte offset into the deferred content (default 0)",
				},
				"limit": map[string]any{
					"type":        "integer",
					"description": "Maximum bytes to return (default 4096)",
				},
			},
			"required": []string{"handle_id"},
		},
	}
}

// Handle processes a fetch_more tool call
func (h *Handler) Handle(args json.RawMessage) (json.RawMessage, error) {
	var req FetchMoreRequest
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("invalid arguments: %v", err)
	}

	if req.Limit <= 0 {
		req.Limit = 4096
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	content, totalSize, exists := h.store.Fetch(req.HandleID, req.Offset, req.Limit)
	if !exists {
		return nil, fmt.Errorf("handle not found or expired: %s", req.HandleID)
	}

	hasMore := req.Offset+len(content) < totalSize

	resp := FetchMoreResponse{
		Content:   content,
		Offset:    req.Offset,
		Limit:     req.Limit,
		TotalSize: totalSize,
		HasMore:   hasMore,
	}

	return json.Marshal(resp)
}
