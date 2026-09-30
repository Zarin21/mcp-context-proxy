# mcp-context-proxy

A Go proxy that sits between MCP agents and MCP servers and compresses tool results before they hit the agent's context window. Agents call tools like GitHub search or log retrieval and get back 40KB JSON blobs or 5,000 repetitive log lines — all of which count against their context limit. This proxy intercepts those responses and applies structure-aware compression: pruning null fields and redundant URLs from JSON, collapsing repeated log patterns, and enforcing per-tool token budgets. Nothing is lost. Truncated content gets stored behind a pagination handle, and the proxy injects a `fetch_more` tool into the agent's tool list so it can pull the rest on demand.

The compression pipeline has three stages. First, log deduplication normalizes timestamps and numbers to detect repeated patterns, collapsing runs of identical lines into a single representative with a count. Second, JSON pruning strips nulls, empties, and long single-line strings (URLs, SHAs), and collapses arrays past a configurable length. Third, if the result still exceeds the token budget, it hard-truncates and stores the overflow in an in-memory store with TTL expiration. The proxy supports both stdio and HTTP transports, handles the full MCP JSON-RPC lifecycle, and works as a drop-in replacement for any MCP server command in your agent config.

## Benchmarks

```
$ ./mcp-context-proxy bench

| Task                 | Raw Tokens | Compressed | Reduction | Success | Handle |
|----------------------|-----------|------------|-----------|---------|--------|
| search_repositories  |    30,670 |      4,096 |    86.6%  |   ✅    |   ✅   |
| list_issues          |    16,858 |      4,096 |    75.7%  |   ✅    |   ✅   |
| get_pull_request     |        46 |         46 |     0.0%  |   ✅    |   —    |
| read_file            |    10,225 |      4,096 |    59.9%  |   ✅    |   ✅   |
| read_logs            |   108,441 |      4,096 |    96.2%  |   ✅    |   ✅   |
| search_logs          |   108,441 |      4,096 |    96.2%  |   ✅    |   ✅   |

Total: 274,681 → 20,526 tokens (92.5% reduction)
Task Success Rate: 6/6 (100%)
```

Success means the compressed result still contains all information needed to complete the task — repo names, issue titles, error messages, function signatures.

## Usage

```bash
go build -o mcp-context-proxy .
./mcp-context-proxy bench
./mcp-context-proxy proxy config.json
```

Point your MCP client at the proxy instead of the real server:

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

## Config

See `config.json` for the default. Per-tool rules let you set token budgets, JSON depth limits, array collapse thresholds, and which compression strategies to apply.

## License

MIT
