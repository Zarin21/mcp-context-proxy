package compress

// Strategy is the type of compression to apply
type Strategy string

const (
	StrategyJSONPrune Strategy = "json_prune"
	StrategyLogDedup  Strategy = "log_dedup"
	StrategyTruncate  Strategy = "truncate"
	StrategyNone      Strategy = "none"
)

// Rule defines compression behavior for a tool
type Rule struct {
	Strategies   []Strategy `yaml:"strategies"`
	MaxTokens    int        `yaml:"max_tokens"`      // token budget (default 4096)
	MaxDepth     int        `yaml:"max_depth"`       // JSON prune depth limit (default 5)
	MaxStringLen int        `yaml:"max_string_len"`  // truncate strings longer than this (default 200)
	MaxArrayLen  int        `yaml:"max_array_len"`   // collapse arrays longer than this (default 20)
	RemoveNulls  bool       `yaml:"remove_nulls"`    // strip null fields
	RemoveEmpty  bool       `yaml:"remove_empty"`    // strip empty strings/arrays/objects
}

// DefaultRule returns sensible defaults
func DefaultRule() Rule {
	return Rule{
		Strategies:   []Strategy{StrategyJSONPrune, StrategyLogDedup, StrategyTruncate},
		MaxTokens:    4096,
		MaxDepth:     5,
		MaxStringLen: 200,
		MaxArrayLen:  20,
		RemoveNulls:  true,
		RemoveEmpty:  true,
	}
}
