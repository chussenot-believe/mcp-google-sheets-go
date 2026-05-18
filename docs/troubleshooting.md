# Troubleshooting

## Summary

This page maps observable symptoms to root causes and concrete fixes. Find the section that matches the error you are seeing.

## Authentication

### Symptom: `no credentials available`

**Cause:** All four credential sources failed.

**Fix:**
- Confirm at least one of `CREDENTIALS_CONFIG`, `SERVICE_ACCOUNT_PATH`, `CREDENTIALS_PATH`, or `GOOGLE_APPLICATION_CREDENTIALS` is set.
- For ADC on a workstation, run:
  ```bash
  gcloud auth application-default login --scopes=https://www.googleapis.com/auth/spreadsheets,https://www.googleapis.com/auth/drive
  ```

### Symptom: `googleapi: Error 403: The caller does not have permission`

**Cause:** Credentials are valid but the principal cannot access this spreadsheet or folder.

**Fix:**
- If using a service account, share the spreadsheet (or its parent folder) with the service account's `client_email`.
- If using OAuth, confirm the user has access through the Drive UI.

### Symptom: `googleapi: Error 401: Request had invalid authentication credentials`

**Cause:** Cached OAuth token is expired beyond its refresh-token TTL, or the credentials JSON is malformed.

**Fix:**
- Delete `token.json` (path from `TOKEN_PATH`) and reauthenticate.
- Re-download the credentials JSON from the Cloud console; ensure it is the **OAuth client** JSON, not the consent-screen export.

### Symptom: Browser opens but never redirects back

**Cause:** The OAuth callback to `127.0.0.1:<random-port>` is blocked by the network — common on remote servers and locked-down VPNs.

**Fix:** Run the interactive flow on a workstation with browser access, then copy the resulting `token.json` to the deployment host.

## Drive folder / `DRIVE_FOLDER_ID`

### Symptom: `list_spreadsheets` returns an empty array

**Cause:** Either `DRIVE_FOLDER_ID` is unset (the principal has nothing in My Drive root that matches) or the folder is not shared with the principal.

**Fix:**
- Confirm the folder ID is correct (the portion after `folders/` in the URL).
- For service accounts, share the folder with the service account address.

### Symptom: `googleapi: Error 404` on `list_spreadsheets`

**Cause:** The folder ID does not exist or the principal has no visibility into it.

**Fix:** Visit the folder URL while logged in as the principal — if you see "Sorry, the file you have requested does not exist", the ID is wrong or unshared.

## Tool-call failures

### Symptom: `Sheet "X" not found`

**Cause:** The sheet/tab name does not exist in the spreadsheet, or is misspelled (Sheets tab names are case-sensitive).

**Fix:** Call `list_sheets` first, then use the exact name from the response.

### Symptom: `Unable to parse range`

**Cause:** Malformed A1 notation, or the `sheet` name contains characters that need quoting (spaces, dashes).

**Fix:**
- Pass `sheet` and `range` as separate arguments; the server joins them.
- If the sheet name has spaces, the server quotes it automatically.

### Symptom: `Invalid chart type "..."`

**Cause:** `chart_type` in `add_chart` is outside the allowed set.

**Fix:** Use one of: `COLUMN`, `BAR`, `LINE`, `AREA`, `PIE`, `SCATTER`, `COMBO`, `HISTOGRAM`.

### Symptom: `Recipient domain not in ALLOWED_SHARE_DOMAINS`

**Cause:** `ALLOWED_SHARE_DOMAINS` is set and the recipient's domain is not on the list.

**Fix:** Either update the recipient list or — only if intentional — extend `ALLOWED_SHARE_DOMAINS`. See [security.md](security.md#share_spreadsheet-allow-list).

## Transport

### Symptom: MCP client cannot connect to SSE endpoint from another machine

**Cause:** The server binds `127.0.0.1` by default.

**Fix:**
1. Confirm the network exposure is intentional and the endpoint is protected by an authenticated reverse proxy.
2. Set `HOST=0.0.0.0`.
3. Restart the server.

If you cannot front the server with auth, do **not** expose it.

### Symptom: stdio transport hangs

**Cause:** The MCP client did not finish its initialise handshake — typically a malformed JSON-RPC frame.

**Fix:** Run the binary by hand and feed it a valid `initialize` request to confirm it responds. The server logs go to stderr.

## Quotas

### Symptom: `googleapi: Error 429: Quota exceeded`

**Cause:** The Google Cloud project's per-minute quota for Sheets or Drive was exceeded.

**Fix:**
- In the Cloud console → APIs & Services → Quotas, raise the relevant per-minute quotas.
- Reduce concurrent tool calls from the agent.
- Cache `list_sheets` / `get_sheet_data` results client-side where appropriate.

## Build and CI

### Symptom: `gofmt` exits non-zero in CI

**Cause:** A file was committed without running `gofmt`.

**Fix:** `mise run fmt && git commit -a --amend --no-edit` then push.

### Symptom: `go test` race detector flags a data race in a new tool

**Cause:** The new handler is accessing shared state without synchronisation.

**Fix:** The Sheets and Drive service objects are safe for concurrent use; do not share other mutable state across handler goroutines.

### Symptom: `mise install` downloads Go on every CI run

**Cause:** The cache key for the mise data dir does not match.

**Fix:** Confirm `mise.toml` and `go.sum` are unchanged. If they are, inspect cache eviction policy on the GitLab runner.

## Still stuck?

1. Reproduce with the binary running directly under stdio so logs land on your terminal.
2. Capture the failing tool call's parameters.
3. Open an issue with: the tool name, parameters (redacted), the server log line, and the resulting error.
