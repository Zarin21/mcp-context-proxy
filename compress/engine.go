package compress

import (
	"encoding/json"
	"strings"
)

// Stats holds compression metrics for a single tool call
type Stats struct {
	ToolName         string  `json:"tool_name"`
	RawTokens        int     `json:"raw_tokens"`
	CompressedTokens int     `json:"compressed_tokens"`
	Ratio            float64 `json:"ratio"` // compression ratio (compressed/raw)
	Truncated        bool    `json:"truncated"`
	HandleCreated    bool    `json:"handle_created"`
}

// Engine orchestrates compression of MCP tool results
type Engine struct {
	rules       map[string]Rule // tool name -> rule
	defaultRule Rule
}

// NewEngine creates a new compression engine with the given per-tool rules
func NewEngine(rules map[string]Rule) *Engine {
	if rules == nil {
		rules = make(map[string]Rule)
	}
	return &Engine{
		rules:       rules,
		defaultRule: DefaultRule(),
	}
}

// Compress compresses a tool result. Returns:
//   - compressed: the compressed result bytes
//   - overflow: data that was truncated (nil if nothing truncated)
//   - stats: compression metrics
//   - err: any error during compression
func (e *Engine) Compress(toolName string, result json.RawMessage) (compressed json.RawMessage, overflow json.RawMessage, stats Stats, err error) {
	rule, ok := e.rules[toolName]
	if !ok {
		rule = e.defaultRule
	}

	rawTokens := EstimateTokensBytes(result)
	stats = Stats{
		ToolName:  toolName,
		RawTokens: rawTokens,
	}

	// If already under budget, return as-is
	if rawTokens <= rule.MaxTokens {
		stats.CompressedTokens = rawTokens
		stats.Ratio = 1.0
		return result, nil, stats, nil
	}

	currentData := result

	// Phase 1: Try to apply log deduplication on text-like content
	currentData = e.applyLogDedup(currentData)

	// Phase 2: Apply JSON pruning
	pruner := &JSONPruner{Rule: rule}
	if pruned, pErr := pruner.Prune(currentData); pErr == nil {
		currentData = pruned
	}

	// Check if we're now under budget
	newTokens := EstimateTokensBytes(currentData)
	if newTokens <= rule.MaxTokens {
		stats.CompressedTokens = newTokens
		if stats.RawTokens > 0 {
			stats.Ratio = float64(newTokens) / float64(stats.RawTokens)
		}
		return currentData, nil, stats, nil
	}

	// Phase 3: Hard truncate to fit budget
	maxBytes := rule.MaxTokens * 4
	if maxBytes < len(currentData) {
		truncData := currentData[:maxBytes]
		ovfData := currentData[maxBytes:]

		stats.CompressedTokens = EstimateTokensBytes(truncData)
		if stats.RawTokens > 0 {
			stats.Ratio = float64(stats.CompressedTokens) / float64(stats.RawTokens)
		}
		stats.Truncated = true

		return truncData, ovfData, stats, nil
	}

	stats.CompressedTokens = EstimateTokensBytes(currentData)
	if stats.RawTokens > 0 {
		stats.Ratio = float64(stats.CompressedTokens) / float64(stats.RawTokens)
	}
	return currentData, nil, stats, nil
}

// applyLogDedup applies log deduplication to text content within JSON.
// Handles both bare JSON strings and objects/arrays containing string values.
func (e *Engine) applyLogDedup(data json.RawMessage) json.RawMessage {
	deduper := &LogDeduplicator{}

	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return data
	}

	modified := e.dedupRecursive(parsed, deduper)
	if b, err := json.Marshal(modified); err == nil {
		return b
	}
	return data
}

// dedupRecursive walks a parsed JSON structure and deduplicates string values
// that look like log output (contain newlines and are longer than 500 chars).
func (e *Engine) dedupRecursive(node any, deduper *LogDeduplicator) any {
	switch v := node.(type) {
	case string:
		// Only dedup strings that look like multi-line log output
		if len(v) > 500 && strings.Count(v, "\n") > 10 {
			return deduper.Dedup(v)
		}
		return v
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, val := range v {
			result[key] = e.dedupRecursive(val, deduper)
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = e.dedupRecursive(item, deduper)
		}
		return result
	default:
		return v
	}
}
