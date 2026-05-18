# mcp-google-sheets-go

Go port of [`xing5/mcp-google-sheets`](https://github.com/xing5/mcp-google-sheets) — a Model Context Protocol server that exposes Google Sheets and Drive as tools for an MCP client (Claude Desktop, etc.).

This port keeps the same 20 tools and the `spreadsheet://{id}/info` resource semantics, and bakes in the security fixes identified in the upstream review:

| Fix | Where |
| --- | --- |
| Drive `q=` query escaping (prevents Drive query-language injection) | `escapeDriveQuery` in `helpers.go`, applied in `list_spreadsheets`, `list_folders`, `search_spreadsheets` |
| `valueInputOption` defaults to `RAW` (no silent formula evaluation) | `chooseValueInputOption` in `tools.go`, used by `update_cells` and `batch_update_cells` |
| SSE transport binds `127.0.0.1` by default | `main.go` |
| OAuth token written with `0600` permissions | `saveToken` in `auth.go` |
| Optional `ALLOWED_SHARE_DOMAINS` allow-list for `share_spreadsheet` | `parseAllowedDomains` in `tools.go` |
| Container runs as non-root, distroless base | `Dockerfile` |

## Build & run

```bash
mise install        # provisions the pinned Go toolchain
mise run build      # go build ./...
mise run test       # go test -race -count=1 ./...
mise run ci         # fmt-check + vet + build + test (what GitLab CI runs)

SERVICE_ACCOUNT_PATH=/path/to/key.json DRIVE_FOLDER_ID=xxx ./mcp-google-sheets-go
```

For SSE:

```bash
./mcp-google-sheets --transport sse
# HOST=127.0.0.1 PORT=8000 by default
```

Tool filtering:

```bash
./mcp-google-sheets --include-tools get_sheet_data,update_cells,list_spreadsheets
# or
ENABLED_TOOLS=get_sheet_data,update_cells ./mcp-google-sheets
```

## Authentication

Same priority chain as the Python original:

1. `CREDENTIALS_CONFIG` — Base64-encoded JSON (service account or OAuth client)
2. `SERVICE_ACCOUNT_PATH` — path to a service-account key file
3. `CREDENTIALS_PATH` (+ `TOKEN_PATH`) — OAuth desktop client; opens a browser on first run
4. Application Default Credentials (`GOOGLE_APPLICATION_CREDENTIALS`, `gcloud auth application-default login`, GCE/GKE metadata)

## Environment variables

| Variable | Purpose |
| --- | --- |
| `SERVICE_ACCOUNT_PATH` | Path to service-account key (default `service_account.json`) |
| `CREDENTIALS_PATH` | OAuth client JSON (default `credentials.json`) |
| `TOKEN_PATH` | OAuth token cache (default `token.json`, written 0600) |
| `CREDENTIALS_CONFIG` | Base64 credentials JSON |
| `DRIVE_FOLDER_ID` | Default Drive folder for `list_spreadsheets` / `create_spreadsheet` |
| `ENABLED_TOOLS` | Comma-separated tool allow-list |
| `ALLOWED_SHARE_DOMAINS` | Comma-separated domain allow-list for `share_spreadsheet` |
| `HOST` / `PORT` | SSE bind address (defaults `127.0.0.1:8000`) |

## Tools

`get_sheet_data`, `get_sheet_formulas`, `update_cells`, `batch_update_cells`, `add_rows`, `add_columns`, `list_sheets`, `copy_sheet`, `rename_sheet`, `get_multiple_sheet_data`, `get_multiple_spreadsheet_summary`, `create_spreadsheet`, `create_sheet`, `list_spreadsheets`, `share_spreadsheet`, `list_folders`, `search_spreadsheets`, `find_in_spreadsheet`, `batch_update`, `add_chart`.

## Documentation

Detailed docs live in [`docs/`](docs/README.md):

- [Getting started](docs/getting-started.md)
- [Authentication](docs/authentication.md)
- [Configuration](docs/configuration.md)
- [Tools reference](docs/tools.md)
- [Resources reference](docs/resources.md)
- [Security model](docs/security.md)
- [Deployment](docs/deployment.md)
- [Development](docs/development.md)
- [Troubleshooting](docs/troubleshooting.md)

## License

MIT (inherits from upstream).
