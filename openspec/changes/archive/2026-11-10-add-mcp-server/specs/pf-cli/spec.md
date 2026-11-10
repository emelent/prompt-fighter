# Spec Delta

## ADDED Requirements

### Requirement: pf mcp
`pf mcp [--server URL]` SHALL run the MCP server over stdio. `--server` is the base URL of the `pf serve` HTTP API used by the match tools, and defaults to `http://127.0.0.1:8080`. An empty value disables the match tools; they then return an explanatory tool error. The command SHALL write only MCP messages to stdout, and SHALL exit 0 when stdin is closed. An invalid flag or an unparsable URL SHALL print an error to stderr and exit 2.

#### Scenario: Default server URL
- **WHEN** `pf mcp` is run with no flags
- **THEN** the match tools target `http://127.0.0.1:8080`

#### Scenario: Invalid URL
- **WHEN** `pf mcp --server ://bad` is run
- **THEN** it prints an error to stderr and exits 2
