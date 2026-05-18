# mcp-google-sheets-go documentation

## Purpose

This directory documents `mcp-google-sheets-go`, a Model Context Protocol (MCP) server that exposes Google Sheets and Google Drive operations as tools and resources for MCP clients (Claude Desktop, IDE integrations, custom agents).

## Audience

These docs are written for two readers:

1. **AI agents** invoking the server and needing to look up tool semantics, parameters, and error conditions.
2. **Operators** integrating, deploying, and securing the server.

Each page is self-contained. Headers are stable and descriptive so agents can navigate by anchor.

## Document index

| Document | Use when you need to |
| --- | --- |
| [getting-started.md](getting-started.md) | Install, build, and run the server for the first time. |
| [authentication.md](authentication.md) | Choose between service-account, OAuth, base64, and ADC auth and configure credentials. |
| [configuration.md](configuration.md) | Look up every environment variable and CLI flag. |
| [tools.md](tools.md) | Look up tool names, parameters, return shapes, and error conditions. |
| [resources.md](resources.md) | Read the URI template for spreadsheet metadata. |
| [security.md](security.md) | Understand the threat model, the security fixes baked in, and required hardening. |
| [deployment.md](deployment.md) | Wire the server into Claude Desktop, run it under Docker, or expose it over SSE. |
| [development.md](development.md) | Run mise tasks, tests, CI; understand the source layout. |
| [troubleshooting.md](troubleshooting.md) | Diagnose authentication, permission, transport, and quota errors. |

## Conventions used in these docs

- **Tool names** are in `lower_snake_case` matching the MCP wire name (e.g. `get_sheet_data`).
- **Environment variables** are in `UPPER_SNAKE_CASE` (e.g. `SERVICE_ACCOUNT_PATH`).
- **Code blocks** are tagged with the language (`bash`, `json`, `go`, `toml`, `yaml`).
- **Parameter tables** list `Name`, `Type`, `Required`, `Description`.
- **Errors** are documented as the literal string the server returns, where stable.

## Related reading

- Upstream Python project: [`xing5/mcp-google-sheets`](https://github.com/xing5/mcp-google-sheets) — feature parity is maintained.
- MCP specification: [modelcontextprotocol.io](https://modelcontextprotocol.io).
- Go SDK used by this server: [`mark3labs/mcp-go`](https://github.com/mark3labs/mcp-go).
- Google APIs: [Sheets v4](https://developers.google.com/sheets/api), [Drive v3](https://developers.google.com/drive/api).
