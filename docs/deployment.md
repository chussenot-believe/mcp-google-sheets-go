# Deployment

## Summary

This page covers three deployment shapes:

1. **Local binary** invoked by an MCP client over stdio (Claude Desktop, IDE plugins).
2. **Docker container** serving SSE behind a reverse proxy.
3. **Kubernetes / managed runtime** using a service account.

Whichever shape you pick, read [security.md](security.md) first.

## Local binary (stdio)

### Build

```bash
mise run build
# produces ./mcp-google-sheets-go
```

### Run directly

```bash
SERVICE_ACCOUNT_PATH=/abs/path/key.json \
DRIVE_FOLDER_ID=xyz \
./mcp-google-sheets-go --transport stdio
```

### Wire into Claude Desktop

Add to `claude_desktop_config.json` under `mcpServers`:

```json
{
  "mcpServers": {
    "google-sheets": {
      "command": "/absolute/path/to/mcp-google-sheets-go",
      "args": ["--transport", "stdio"],
      "env": {
        "SERVICE_ACCOUNT_PATH": "/absolute/path/to/key.json",
        "DRIVE_FOLDER_ID": "your-folder-id"
      }
    }
  }
}
```

Restart Claude Desktop. The tools appear in the MCP picker.

### Tool filtering example

Reduce token usage by registering only the tools you need:

```json
{
  "mcpServers": {
    "google-sheets": {
      "command": "/absolute/path/to/mcp-google-sheets-go",
      "args": ["--include-tools", "get_sheet_data,update_cells,list_spreadsheets"],
      "env": {
        "SERVICE_ACCOUNT_PATH": "/abs/key.json"
      }
    }
  }
}
```

## Docker container (SSE)

### Build the image

```bash
docker build -t mcp-google-sheets-go:local .
```

The Dockerfile produces a distroless image running as `nonroot`. See [Dockerfile](../Dockerfile).

### Run with mounted credentials

```bash
docker run --rm \
  -p 127.0.0.1:8000:8000 \
  -v /abs/path/key.json:/run/secrets/key.json:ro \
  -e SERVICE_ACCOUNT_PATH=/run/secrets/key.json \
  -e DRIVE_FOLDER_ID=xyz \
  mcp-google-sheets-go:local
```

The container binds `127.0.0.1:8000` by default. Publishing only on the loopback interface prevents accidental network exposure.

### Run with base64 credentials

```bash
b64=$(base64 -w 0 /abs/path/key.json)
docker run --rm -p 127.0.0.1:8000:8000 \
  -e CREDENTIALS_CONFIG="$b64" \
  -e DRIVE_FOLDER_ID=xyz \
  mcp-google-sheets-go:local
```

This avoids a volume mount but exposes the secret in the container's environment block.

### Exposing remotely

If you need a remote endpoint, **do not** simply switch the publish to `-p 8000:8000`. Instead:

1. Terminate TLS at a reverse proxy (Caddy, Nginx, Envoy).
2. Enforce authentication at the proxy (bearer token, OIDC, mTLS).
3. Have the proxy forward to `127.0.0.1:8000`.

The MCP protocol itself does not authenticate the client.

## Kubernetes / Cloud Run / managed runtime

### Recommended setup

- Use Workload Identity (GKE) or a service-account binding (Cloud Run) so the workload picks up credentials via ADC — no JSON key in the container.
- Set `DRIVE_FOLDER_ID` via a Kubernetes `ConfigMap`.
- Run the SSE transport with `HOST=0.0.0.0` **only when** the Service / Ingress in front enforces auth.

### Example pod spec fragment

```yaml
containers:
  - name: mcp-google-sheets
    image: registry.example.com/mcp-google-sheets-go:1.0.0
    args: ["--transport", "sse"]
    env:
      - name: HOST
        value: "0.0.0.0"   # safe ONLY because the Service in front authenticates
      - name: DRIVE_FOLDER_ID
        valueFrom:
          configMapKeyRef:
            name: mcp-config
            key: drive_folder_id
      - name: ALLOWED_SHARE_DOMAINS
        value: "example.com"
    ports:
      - containerPort: 8000
    securityContext:
      runAsNonRoot: true
      readOnlyRootFilesystem: true
      allowPrivilegeEscalation: false
      capabilities:
        drop: ["ALL"]
```

### Drive folder permission

The workload's service account must be granted access to `DRIVE_FOLDER_ID`. In Workload Identity setups, this is done by sharing the folder with the GSA email — not the KSA.

## Health checks

The server does not currently expose an HTTP `/health` endpoint. For Kubernetes liveness/readiness, use a TCP probe against the SSE port:

```yaml
readinessProbe:
  tcpSocket:
    port: 8000
```

## Upgrades

This is a single static binary. Replace the binary or rebuild the image; there is no migration state to manage.

If you change Go versions or dependency versions, run `mise run ci` locally before publishing to confirm nothing regressed.
