# Configuration

## Summary

The server is configured by environment variables (for secrets and deployment-shaped knobs) and CLI flags (for transport and tool filtering). There is no config file.

## Environment variables

### Authentication

| Variable | Purpose | Default |
| --- | --- | --- |
| `CREDENTIALS_CONFIG` | Base64 JSON of a service account or OAuth client. Highest priority. | unset |
| `SERVICE_ACCOUNT_PATH` | Path to a service-account JSON key. | `service_account.json` |
| `CREDENTIALS_PATH` | Path to an OAuth desktop-client JSON. | `credentials.json` |
| `TOKEN_PATH` | Where to cache the OAuth refresh token (mode `0600`). | `token.json` |
| `GOOGLE_APPLICATION_CREDENTIALS` | Google's standard ADC variable. | unset |

See [authentication.md](authentication.md) for resolution order and trade-offs.

### Drive context

| Variable | Purpose | Default |
| --- | --- | --- |
| `DRIVE_FOLDER_ID` | Default folder ID for `list_spreadsheets` and `create_spreadsheet`. | unset (i.e., My Drive root) |

### Tool filtering

| Variable | Purpose | Default |
| --- | --- | --- |
| `ENABLED_TOOLS` | Comma-separated tool allow-list. Equivalent to `--include-tools`. | unset (all enabled) |

See [Tool filtering](#tool-filtering) below.

### Sharing controls

| Variable | Purpose | Default |
| --- | --- | --- |
| `ALLOWED_SHARE_DOMAINS` | Comma-separated email domains permitted as `share_spreadsheet` recipients. If unset, no domain restriction is applied. | unset |

Example: `ALLOWED_SHARE_DOMAINS=example.com,believe.com`.

### Transport (SSE only)

| Variable | Purpose | Default |
| --- | --- | --- |
| `HOST` (alias `FASTMCP_HOST`) | Bind address for the SSE server. | `127.0.0.1` |
| `PORT` (alias `FASTMCP_PORT`) | Bind port for the SSE server. | `8000` |

The default `127.0.0.1` is intentional. See [security.md](security.md#sse-transport-binds-localhost-by-default) for the rationale before changing it to `0.0.0.0`.

## CLI flags

```
mcp-google-sheets-go [--transport <stdio|sse>] [--include-tools <list>]
```

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--transport` | string | `stdio` | Transport to serve over. `stdio` is the standard MCP transport for desktop clients; `sse` is HTTP-based. |
| `--include-tools` | string | empty | Comma-separated list of tool names to register. Overrides `ENABLED_TOOLS`. |

## Tool filtering

The default behaviour registers all 20 tools. This costs roughly 13 K tokens in the model's context window before any conversation starts. If the agent only uses a subset, restrict the set:

### Via CLI flag

```bash
./mcp-google-sheets-go --include-tools get_sheet_data,update_cells,list_spreadsheets
```

### Via environment variable

```bash
ENABLED_TOOLS=get_sheet_data,update_cells,list_spreadsheets ./mcp-google-sheets-go
```

### Precedence

`--include-tools` overrides `ENABLED_TOOLS`. If neither is set, all tools are registered.

### Common subsets

| Use case | Suggested tools |
| --- | --- |
| Read-only agent | `get_sheet_data`, `get_sheet_formulas`, `list_sheets`, `list_spreadsheets`, `find_in_spreadsheet`, `get_multiple_spreadsheet_summary` |
| Light edit | Above + `update_cells`, `add_rows` |
| Full CRUD | Above + `create_spreadsheet`, `create_sheet`, `batch_update_cells`, `rename_sheet`, `copy_sheet` |
| Administrative | All tools |

The full list is in [tools.md](tools.md).

## Value-input policy

`update_cells` and `batch_update_cells` accept an optional `value_input_option` argument.

| Value | Behaviour |
| --- | --- |
| `RAW` (default) | Values written verbatim. Leading `=` is treated as text, not a formula. |
| `USER_ENTERED` | Values parsed as if a human typed them into the UI. `=SUM(A1:A10)` becomes a formula; `5/9/2024` becomes a date. |

The default is `RAW` to prevent silent formula-based data exfiltration via `IMPORTDATA`, `IMPORTXML`, or `HYPERLINK`. See [security.md](security.md#sheets-formula-injection).

## Logging

The server logs to stderr at INFO level. There is no log-level flag in the current build.
