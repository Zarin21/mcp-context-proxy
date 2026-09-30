package compress

// EstimateTokens returns an approximate token count for the given text.
// Uses the ~4 bytes per token heuristic common for GPT-style tokenizers.
func EstimateTokens(text string) int {
	return len(text) / 4
}

// EstimateTokensBytes same but for []byte
func EstimateTokensBytes(data []byte) int {
	return len(data) / 4
}
