# Getting started

## What you will accomplish

By the end of this page you will have:

1. Installed the build toolchain.
2. Provisioned Google Cloud credentials with the correct API scopes.
3. Built the server binary.
4. Run the server over the stdio transport and confirmed it starts.

## Prerequisites

| Requirement | Why it is needed |
| --- | --- |
| Go 1.23 (pinned in `mise.toml`) | To compile the binary. |
| `mise` (recommended) | Provisions the pinned Go version and exposes the project tasks. |
| A Google Cloud project | To host the Sheets API and Drive API. |
| Either a service-account key OR an OAuth client OR ADC | To authenticate against Google. |

If you do not use `mise`, install Go 1.23 manually and run the `go` commands documented under each `mise run …` line.

## Step 1 — Install mise (optional but recommended)

```bash
curl -fsSL https://mise.run | sh
export PATH="$HOME/.local/bin:$PATH"
```

`mise` provisions the exact Go version from `mise.toml`, so the local toolchain matches CI.

## Step 2 — Enable Google APIs

In the [Google Cloud console](https://console.cloud.google.com/) for your project, enable both:

- **Google Sheets API**
- **Google Drive API**

The Sheets API alone is not enough — `list_spreadsheets`, `create_spreadsheet`, `share_spreadsheet`, `list_folders`, and `search_spreadsheets` use the Drive API.

## Step 3 — Create credentials

Pick one method. See [authentication.md](authentication.md) for full details.

### Quickest path: service account

1. Cloud console → IAM & Admin → Service Accounts → Create.
2. Add JSON key → download to a safe location.
3. Create a Google Drive folder, note its ID from the URL.
4. Share that folder with the service account's `client_email` (Editor).

Export:

```bash
export SERVICE_ACCOUNT_PATH="/absolute/path/to/key.json"
export DRIVE_FOLDER_ID="the-folder-id-from-the-url"
```

## Step 4 — Build the server

```bash
cd mcp-google-sheets-go
mise install            # provisions Go 1.23
mise run build          # equivalent to: go build -v ./...
```

The binary is produced as `./mcp-google-sheets-go` in the project root.

## Step 5 — Run

```bash
./mcp-google-sheets-go --transport stdio
```

The server now waits for MCP JSON-RPC messages on stdin and writes responses to stdout. Logs go to stderr.

To run over Server-Sent Events instead:

```bash
./mcp-google-sheets-go --transport sse
# default bind: 127.0.0.1:8000 — see security.md before changing this
```

## Step 6 — Smoke test the build

```bash
mise run ci    # fmt-check + vet + build + test
```

This is exactly what GitLab CI runs.

## Next steps

- Wire the server into Claude Desktop or another MCP client → [deployment.md](deployment.md).
- Look up specific tool semantics → [tools.md](tools.md).
- Reduce context-window cost by enabling only the tools you need → [configuration.md](configuration.md#tool-filtering).
