package bench

import (
	"encoding/json"
	"fmt"

	"github.com/zarinsubah/mcp-context-proxy/compress"
)

// Result holds benchmark results for a single task
type Result struct {
	TaskName         string  `json:"task_name"`
	RawTokens        int     `json:"raw_tokens"`
	CompressedTokens int     `json:"compressed_tokens"`
	Ratio            float64 `json:"ratio_percent"` // percentage reduction
	TaskSuccess      bool    `json:"task_success"`
	HandleCreated    bool    `json:"handle_created"`
}

// Runner executes the benchmark suite
type Runner struct {
	engine *compress.Engine
}

// NewRunner creates a benchmark runner
func NewRunner(engine *compress.Engine) *Runner {
	return &Runner{engine: engine}
}

// Run executes all benchmark tasks and returns results
func (r *Runner) Run() []Result {
	tasks := GetTasks()
	results := make([]Result, 0, len(tasks))

	githubServer := &MockGitHubServer{}
	fsServer := &MockFilesystemServer{}
	logServer := &MockLogServer{}

	for _, task := range tasks {
		var raw json.RawMessage
		argsJSON, _ := json.Marshal(task.Args)
		switch task.ToolName {
		case "search_repositories", "list_issues", "get_pull_request":
			raw = githubServer.HandleToolCall(task.ToolName, argsJSON)
		case "read_file":
			raw = fsServer.HandleToolCall(task.ToolName, argsJSON)
		case "read_logs", "search_logs":
			raw = logServer.HandleToolCall(task.ToolName, argsJSON)
		}

		rawTokens := compress.EstimateTokensBytes(raw)

		compressed, overflow, stats, _ := r.engine.Compress(task.ToolName, raw)

		compTokens := compress.EstimateTokensBytes(compressed)
		var ratio float64
		if rawTokens > 0 {
			ratio = float64(rawTokens-compTokens) / float64(rawTokens) * 100
		}

		// Validate against the compressed result
		success := task.ValidateResult(string(compressed))

		hasHandle := overflow != nil || stats.Truncated

		results = append(results, Result{
			TaskName:         task.Name,
			RawTokens:        rawTokens,
			CompressedTokens: compTokens,
			Ratio:            ratio,
			TaskSuccess:      success,
			HandleCreated:    hasHandle,
		})
	}

	return results
}

// PrintReport prints a formatted markdown table of results
func PrintReport(results []Result) {
	fmt.Println("## MCP Context Proxy — Benchmark Results")
	fmt.Println()
	fmt.Println("| Task | Raw Tokens | Compressed | Reduction | Success | Handle |")
	fmt.Println("|------|-----------|------------|-----------|---------|--------|")

	var totalRaw, totalComp int
	var successCount int

	for _, r := range results {
		totalRaw += r.RawTokens
		totalComp += r.CompressedTokens
		if r.TaskSuccess {
			successCount++
		}

		successIcon := "❌"
		if r.TaskSuccess {
			successIcon = "✅"
		}
		handleIcon := "—"
		if r.HandleCreated {
			handleIcon = "✅"
		}

		fmt.Printf("| %-20s | %8d | %8d | %5.1f%% | %s | %s |\n",
			r.TaskName, r.RawTokens, r.CompressedTokens, r.Ratio, successIcon, handleIcon)
	}

	fmt.Println()
	avgReduction := 0.0
	if totalRaw > 0 {
		avgReduction = float64(totalRaw-totalComp) / float64(totalRaw) * 100
	}
	fmt.Printf("**Total: %d → %d tokens (%.1f%% reduction)**\n", totalRaw, totalComp, avgReduction)
	fmt.Printf("**Task Success Rate: %d/%d (%.0f%%)**\n", successCount, len(results),
		float64(successCount)/float64(len(results))*100)
}
