# Authentication

## Summary

The server obtains an OAuth2 token from one of four credential sources, in priority order. It then uses that token to talk to both the Sheets API and the Drive API.

## Resolution order

The server checks each source in this order and uses the first one that yields valid credentials:

| Order | Source | Trigger |
| --- | --- | --- |
| 1 | `CREDENTIALS_CONFIG` | Environment variable is set and contains base64 JSON. |
| 2 | `SERVICE_ACCOUNT_PATH` | Path resolves to a readable service-account JSON key. |
| 3 | `CREDENTIALS_PATH` + `TOKEN_PATH` | OAuth client JSON is present; opens browser flow if no cached token. |
| 4 | Application Default Credentials (ADC) | Always tried as a fallback. |

Source code: [`auth.go`](../auth.go), function `resolveTokenSource`.

## OAuth scopes requested

The server always requests the following scopes:

- `https://www.googleapis.com/auth/spreadsheets`
- `https://www.googleapis.com/auth/drive`

If your credentials grant narrower scopes, the relevant tools will fail with a permission error from the Google API.

## Method A — Service account (recommended for servers)

### When to use

- The server runs unattended.
- You want a stable, long-lived identity that is not tied to a user account.
- You can manage a Google Drive folder shared with the service account.

### Setup

1. Cloud console → IAM & Admin → Service Accounts → Create service account.
2. Manage Keys → Add Key → JSON → download.
3. Create a Drive folder, copy its ID from the URL.
4. Right-click the folder → Share → enter the service account's `client_email`, role `Editor`. Uncheck "Notify people".
5. Export environment variables:
   ```bash
   export SERVICE_ACCOUNT_PATH="/absolute/path/to/key.json"
   export DRIVE_FOLDER_ID="the-folder-id"
   ```

### Notes

- The service account only sees files explicitly shared with its address. `list_spreadsheets` returns the spreadsheets inside `DRIVE_FOLDER_ID`.
- The service account email looks like `name@project-id.iam.gserviceaccount.com`.

## Method B — OAuth 2.0 desktop flow

### When to use

- Interactive use on a developer workstation.
- You need to act as a real user (e.g., reach private spreadsheets without explicit sharing).

### Setup

1. Cloud console → APIs & Services → OAuth consent screen → External → fill required fields → add scopes `auth/spreadsheets` and `auth/drive`.
2. Credentials → Create credentials → OAuth client ID → Application type **Desktop app** → download JSON.
3. Place the JSON at `credentials.json` or export `CREDENTIALS_PATH`.
4. The first run opens a browser; subsequent runs reuse the cached token at `TOKEN_PATH` (default `token.json`).

### Token storage

The cached token is written with file mode `0600` to limit exposure on multi-user hosts. See [security.md](security.md#oauth-token-file-permissions).

### Re-authentication

If the cached token cannot be refreshed (revoked, expired beyond refresh-token TTL, scope change), the server reopens the browser flow. Delete the token file to force reauthentication.

## Method C — Base64 credentials (containers, CI)

### When to use

- The runtime is a container or CI job where mounting a JSON file is awkward.
- You can inject secrets via environment variables.

### Setup

```bash
# Linux/macOS
b64=$(base64 -w 0 your_credentials.json)
export CREDENTIALS_CONFIG="$b64"
```

The decoded JSON may be either a service-account key or an OAuth client. The server detects which based on the JSON shape.

### Trade-offs

- ✅ No filesystem secret to manage.
- ⚠️ Credentials appear in `/proc/<pid>/environ`, container introspection, and shell history. Treat with the same caution as any secret in env vars.

## Method D — Application Default Credentials (ADC)

### When to use

- Running on Google Cloud (GKE, Cloud Run, Compute Engine) — the metadata server provides credentials automatically.
- Local development with `gcloud auth application-default login`.
- You set `GOOGLE_APPLICATION_CREDENTIALS` (Google's standard env var) to a service-account key path.

### Setup for local development

```bash
gcloud auth application-default login \
  --scopes=https://www.googleapis.com/auth/cloud-platform,\
https://www.googleapis.com/auth/spreadsheets,\
https://www.googleapis.com/auth/drive
gcloud auth application-default set-quota-project YOUR_PROJECT_ID
```

### ADC lookup order (Google's chain)

1. `GOOGLE_APPLICATION_CREDENTIALS` file
2. `gcloud auth application-default login` credentials at `~/.config/gcloud/application_default_credentials.json`
3. GCE/GKE metadata server

## Verifying which method was used

When the server starts it logs which credential source it loaded, e.g.:

```
Using service account authentication (/abs/path/key.json)
```

If you see `Falling back to Application Default Credentials`, none of the earlier methods produced credentials.

## Common authentication errors

| Error | Likely cause |
| --- | --- |
| `no credentials available` | All four sources failed. Set at least one. |
| `googleapi: Error 403: The caller does not have permission` | The credentials are valid but the file is not shared with the principal. |
| `googleapi: Error 404` on `list_spreadsheets` | `DRIVE_FOLDER_ID` points to a folder the principal cannot see, or the folder ID is wrong. |
| Browser opens but does not redirect | Network blocks the OAuth callback to `127.0.0.1:<random-port>`. Run OAuth flow on a workstation with browser access, then copy the resulting `token.json` to the deployment host. |
