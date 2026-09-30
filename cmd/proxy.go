package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/zarinsubah/mcp-context-proxy/compress"
	"github.com/zarinsubah/mcp-context-proxy/config"
	"github.com/zarinsubah/mcp-context-proxy/pagination"
	"github.com/zarinsubah/mcp-context-proxy/proxy"
)

// RunProxy starts the proxy server
func RunProxy(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[mcp-context-proxy] warning: failed to load config %s: %v — using defaults\n", configPath, err)
		cfg = config.Default()
	}

	// Build compression rules from config
	rules := make(map[string]compress.Rule)
	for toolName, tr := range cfg.ToolRules {
		strategies := make([]compress.Strategy, len(tr.Strategies))
		for i, s := range tr.Strategies {
			strategies[i] = compress.Strategy(s)
		}
		rules[toolName] = compress.Rule{
			Strategies:   strategies,
			MaxTokens:    tr.MaxTokens,
			MaxDepth:     tr.MaxDepth,
			MaxStringLen: tr.MaxStringLen,
			MaxArrayLen:  tr.MaxArrayLen,
			RemoveNulls:  tr.RemoveNulls,
			RemoveEmpty:  tr.RemoveEmpty,
		}
	}

	engine := compress.NewEngine(rules)

	// Parse TTL
	ttl := 10 * time.Minute
	if cfg.HandleTTL != "" {
		if d, err := time.ParseDuration(cfg.HandleTTL); err == nil {
			ttl = d
		}
	}
	store := pagination.NewStore(ttl)
	handler := pagination.NewHandler(store)

	server := proxy.NewServer(cfg, engine, store, handler)

	// Connect to upstream MCP servers
	ctx := context.Background()
	for _, sc := range cfg.Servers {
		var transport proxy.Transport
		var tErr error

		switch sc.Transport {
		case "stdio":
			transport, tErr = proxy.NewStdioTransport(ctx, sc.Command, sc.Args)
		case "http":
			transport, tErr = proxy.NewHTTPTransport(ctx, sc.URL)
		default:
			fmt.Fprintf(os.Stderr, "[mcp-context-proxy] unknown transport %q for server %q\n", sc.Transport, sc.Name)
			continue
		}

		if tErr != nil {
			fmt.Fprintf(os.Stderr, "[mcp-context-proxy] failed to connect to %q: %v\n", sc.Name, tErr)
			continue
		}

		client := proxy.NewClient(sc.Name, transport)
		if err := client.Initialize(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "[mcp-context-proxy] failed to initialize %q: %v\n", sc.Name, err)
			transport.Close()
			continue
		}

		server.AddClient(client)
		fmt.Fprintf(os.Stderr, "[mcp-context-proxy] connected to upstream server %q\n", sc.Name)
	}

	fmt.Fprintf(os.Stderr, "[mcp-context-proxy] ready — %d upstream server(s)\n", len(cfg.Servers))
	return server.RunStdio(ctx)
}
