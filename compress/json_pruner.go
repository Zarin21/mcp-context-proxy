package compress

import (
	"encoding/json"
	"fmt"
	"strings"
)

// JSONPruner prunes JSON structures to reduce token count
type JSONPruner struct {
	Rule Rule
}

// Prune takes raw JSON bytes and returns pruned JSON bytes
func (p *JSONPruner) Prune(data []byte) ([]byte, error) {
	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}

	pruned := p.pruneNode(parsed, 0)
	return json.Marshal(pruned)
}

func (p *JSONPruner) pruneNode(node any, depth int) any {
	if depth >= p.Rule.MaxDepth {
		switch node.(type) {
		case map[string]any:
			return "[object truncated]"
		case []any:
			return "[array truncated]"
		}
	}

	switch v := node.(type) {
	case string:
		// Only truncate single-line strings (URLs, hashes, descriptions).
		// Multi-line strings (logs, code) are handled by the token-budget truncation
		// which preserves fetch_more handles for the deferred content.
		if p.Rule.MaxStringLen > 0 && len(v) > p.Rule.MaxStringLen && !strings.Contains(v, "\n") {
			return v[:p.Rule.MaxStringLen] + "...[truncated]"
		}
		if p.Rule.RemoveEmpty && v == "" {
			return nil
		}
		return v
	case map[string]any:
		if p.Rule.RemoveEmpty && len(v) == 0 {
			return nil
		}
		res := make(map[string]any)
		for key, val := range v {
			if val == nil && p.Rule.RemoveNulls {
				continue
			}
			prunedVal := p.pruneNode(val, depth+1)
			if prunedVal == nil && p.Rule.RemoveNulls {
				continue
			}
			res[key] = prunedVal
		}
		if p.Rule.RemoveEmpty && len(res) == 0 {
			return nil
		}
		return res
	case []any:
		if p.Rule.RemoveEmpty && len(v) == 0 {
			return nil
		}
		var res []any
		for i, item := range v {
			if p.Rule.MaxArrayLen > 0 && i >= p.Rule.MaxArrayLen {
				res = append(res, fmt.Sprintf("...and %d more items", len(v)-p.Rule.MaxArrayLen))
				break
			}
			prunedItem := p.pruneNode(item, depth+1)
			if prunedItem == nil && p.Rule.RemoveNulls {
				continue
			}
			res = append(res, prunedItem)
		}
		if p.Rule.RemoveEmpty && len(res) == 0 {
			return nil
		}
		return res
	default:
		return v
	}
}
