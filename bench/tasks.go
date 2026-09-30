package bench

import "strings"

// Task represents a benchmark task
type Task struct {
	Name        string
	Description string
	ToolName    string
	Args        map[string]any
	// ValidateResult checks if the compressed result still contains the essential info
	ValidateResult func(result string) bool
}

// GetTasks returns the fixed set of benchmark tasks
func GetTasks() []Task {
	return []Task{
		{
			Name:        "search_repositories",
			Description: "Search for Go repos",
			ToolName:    "search_repositories",
			Args:        map[string]any{"query": "language:go"},
			ValidateResult: func(result string) bool {
				return strings.Contains(result, "repo-1")
			},
		},
		{
			Name:        "list_issues",
			Description: "List issues",
			ToolName:    "list_issues",
			Args:        map[string]any{"repo": "owner/repo"},
			ValidateResult: func(result string) bool {
				return strings.Contains(result, "Found a bug")
			},
		},
		{
			Name:        "get_pull_request",
			Description: "Get a PR",
			ToolName:    "get_pull_request",
			Args:        map[string]any{"repo": "owner/repo", "pr_number": 1},
			ValidateResult: func(result string) bool {
				return strings.Contains(result, "Fix context length issue")
			},
		},
		{
			Name:        "read_file",
			Description: "Read a large file",
			ToolName:    "read_file",
			Args:        map[string]any{"path": "main.go"},
			ValidateResult: func(result string) bool {
				return strings.Contains(result, "ExampleFunction0")
			},
		},
		{
			Name:        "read_logs",
			Description: "Read 5000 log lines",
			ToolName:    "read_logs",
			Args:        map[string]any{"service": "backend"},
			ValidateResult: func(result string) bool {
				return strings.Contains(result, "Database connection lost")
			},
		},
		{
			Name:        "search_logs",
			Description: "Search for errors in logs",
			ToolName:    "search_logs",
			Args:        map[string]any{"query": "ERROR"},
			ValidateResult: func(result string) bool {
				return strings.Contains(result, "Database connection lost")
			},
		},
	}
}
