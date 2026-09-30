package main

import (
	"fmt"
	"os"

	"github.com/zarinsubah/mcp-context-proxy/cmd"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "proxy":
		configPath := "config.json"
		if len(os.Args) > 2 {
			configPath = os.Args[2]
		}
		if err := cmd.RunProxy(configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "bench":
		if err := cmd.RunBench(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("mcp-context-proxy - MCP tool result compression proxy")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  mcp-context-proxy proxy [config.json]  - Run the proxy server")
	fmt.Println("  mcp-context-proxy bench                - Run compression benchmarks")
}
