# MCP Context Proxy

> **A Go proxy between MCP agents and servers that cuts tool-result tokens by 92% with no drop in task success.**

## The Problem

When LLM agents call MCP tools, the results can be *massive*:

- **GitHub `search_repositories`** → 40KB JSON with 30 repos, each with 50+ URL fields, null permissions, license metadata
- **Filesystem `read_file`** → 15KB of source code dumped verbatim
- **Log server `read_logs`** → 5,000 repetitive log lines, 108K tokens

All of this lands in the agent's context window. It bloats costs, degrades reasoning, and hits context limits — even though 90% of the data is noise.

## The Solution

**mcp-context-proxy** sits transparently between your agent and MCP servers. It intercepts `tools/call` responses and applies structure-aware compression:

```
┌─────────┐     JSON-RPC      ┌──────────────────┐     JSON-RPC      ┌────────────┐
│  Agent   │ ◄──────────────► │ mcp-context-proxy │ ◄──────────────► │ MCP Server │
│ (Client) │     stdio/HTTP   │   Compression &   │     stdio/HTTP   │ (GitHub,   │
│          │                  │   Pagination      │                  │  FS, Logs) │
└─────────┘                   └──────────────────┘                   └────────────┘
```

Nothing is lost — only **deferred**. Truncated content gets a `fetch_more` handle so agents can retrieve it on demand.

## Benchmark Results

```
$ mcp-context-proxy bench
```

| Task                 | Raw Tokens | Compressed | Reduction | Success | Handle |
|----------------------|-----------|------------|-----------|---------|--------|
| search_repositories  |    30,670 |      4,096 |    86.6%  |   ✅    |   ✅   |
| list_issues          |    16,858 |      4,096 |    75.7%  |   ✅    |   ✅   |
| get_pull_request     |        46 |         46 |     0.0%  |   ✅    |   —    |
| read_file            |    10,225 |      4,096 |    59.9%  |   ✅    |   ✅   |
| read_logs            |   108,441 |      4,096 |    96.2%  |   ✅    |   ✅   |
| search_logs          |   108,441 |      4,096 |    96.2%  |   ✅    |   ✅   |

**Total: 274,681 → 20,526 tokens (92.5% reduction)**
**Task Success Rate: 6/6 (100%)**

"Success" means the compressed result still contains all information needed to complete the task (repo names, issue titles, error messages, function signatures).

## How It Works

### 1. Schema-Aware JSON Pruning
- Removes `null` fields and empty strings/arrays/objects
- Truncates long single-line strings (URLs, SHA hashes, base64 blobs)
- Collapses arrays beyond a configurable length → `"...and 20 more items"`
- Enforces depth limits on nested structures

### 2. Log Deduplication
- Normalizes lines by stripping timestamps and replacing numbers with placeholders
- Consecutive lines matching the same pattern collapse to `"[line] (repeated N times)"`
- 5,000 health-check lines → ~50 representative lines + counts

### 3. Token-Budget Truncation
- Per-tool configurable token budget (default: 4096)
- If pruning + dedup still exceed the budget, hard-truncate to fit
- Overflow data stored in memory with a pagination handle

### 4. `fetch_more` Virtual Tool
- Automatically injected into the agent's tool list
- Agent calls `fetch_more(handle_id)` to retrieve deferred content
- Supports byte-offset pagination for very large results
- Handles expire with configurable TTL (default: 10 minutes)

## Quick Start

### Build
```bash
go build -o mcp-context-proxy .
```

### Run the proxy
```bash
./mcp-context-proxy proxy [config.json]
```

### Run benchmarks
```bash
./mcp-context-proxy bench
```

### Use with Claude Desktop / any MCP client

In your MCP client config, point the tool server at the proxy instead of the real server:

```json
{
  "mcpServers": {
    "github-proxied": {
      "command": "./mcp-context-proxy",
      "args": ["proxy", "config.json"]
    }
  }
}
```

## Configuration

`config.json`:

```json
{
  "listen": ":9800",
  "transport": "stdio",
  "handle_ttl": "10m",
  "default_rule": {
    "strategies": ["json_prune", "log_dedup", "truncate"],
    "max_tokens": 4096,
    "max_depth": 5,
    "max_string_len": 200,
    "max_array_len": 20,
    "remove_nulls": true,
    "remove_empty": true
  },
  "tool_rules": {
    "read_file": {
      "strategies": ["log_dedup", "truncate"],
      "max_tokens": 8192,
      "max_depth": 10,
      "max_string_len": 500,
      "max_array_len": 50,
      "remove_nulls": false,
      "remove_empty": false
    }
  },
  "servers": [
    {
      "name": "github",
      "transport": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"]
    }
  ]
}
```

### Configuration Fields

| Field | Description | Default |
|-------|-------------|---------|
| `listen` | HTTP listen address (for HTTP mode) | `:9800` |
| `transport` | Proxy-to-agent transport: `stdio` or `http` | `stdio` |
| `handle_ttl` | How long pagination handles survive | `10m` |
| `default_rule` | Default compression rule for all tools | See above |
| `tool_rules` | Per-tool compression rule overrides | `{}` |
| `servers` | List of upstream MCP servers to proxy | `[]` |

### Per-Tool Rule Fields

| Field | Description | Default |
|-------|-------------|---------|
| `strategies` | Compression pipeline steps | `["json_prune","log_dedup","truncate"]` |
| `max_tokens` | Token budget for this tool's results | `4096` |
| `max_depth` | Max JSON nesting depth before truncation | `5` |
| `max_string_len` | Truncate single-line strings beyond this | `200` |
| `max_array_len` | Collapse arrays beyond this length | `20` |
| `remove_nulls` | Strip null-valued fields | `true` |
| `remove_empty` | Strip empty strings/arrays/objects | `true` |

## Project Structure

```
mcp-context-proxy/
├── main.go                  # CLI entry point (proxy / bench)
├── cmd/
│   ├── proxy.go             # proxy subcommand wiring
│   └── bench.go             # benchmark subcommand
├── compress/
│   ├── engine.go            # Compression pipeline orchestrator
│   ├── json_pruner.go       # Schema-aware JSON pruning
│   ├── log_dedup.go         # Log line deduplication
│   ├── token_counter.go     # ~4 bytes/token estimation
│   └── rules.go             # Per-tool compression rules
├── pagination/
│   ├── store.go             # In-memory handle store with TTL
│   └── handler.go           # fetch_more tool implementation
├── proxy/
│   ├── server.go            # MCP proxy server (agent-facing)
│   ├── client.go            # MCP client (upstream-facing)
│   ├── router.go            # JSON-RPC routing + request tracking
│   ├── stdio.go             # stdio transport
│   └── http.go              # HTTP transport
├── bench/
│   ├── runner.go            # Benchmark orchestrator
│   ├── mock_server.go       # Realistic mock MCP servers
│   └── tasks.go             # Fixed task definitions + validators
├── config/
│   └── config.go            # Configuration loading
└── config.json              # Default config
```

## License

MIT
