package cmd

import (
	"github.com/zarinsubah/mcp-context-proxy/bench"
	"github.com/zarinsubah/mcp-context-proxy/compress"
)

// RunBench runs the benchmark suite
func RunBench() error {
	engine := compress.NewEngine(nil)
	runner := bench.NewRunner(engine)
	results := runner.Run()
	bench.PrintReport(results)
	return nil
}
