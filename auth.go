package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var oauthScopes = []string{sheets.SpreadsheetsScope, drive.DriveScope}

// authenticate resolves credentials using the documented priority chain:
//  1. CREDENTIALS_CONFIG (base64 JSON, service account or OAuth client)
//  2. SERVICE_ACCOUNT_PATH (service account key file)
//  3. CREDENTIALS_PATH + TOKEN_PATH (OAuth client; interactive flow on miss)
//  4. Application Default Credentials (GOOGLE_APPLICATION_CREDENTIALS, gcloud,
//     metadata server)
func authenticate(ctx context.Context) (*sheets.Service, *drive.Service, error) {
	tokenSource, err := resolveTokenSource(ctx)
	if err != nil {
		return nil, nil, err
	}

	sheetsSvc, err := sheets.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, nil, fmt.Errorf("build sheets service: %w", err)
	}
	driveSvc, err := drive.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, nil, fmt.Errorf("build drive service: %w", err)
	}
	return sheetsSvc, driveSvc, nil
}

func resolveTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	if cfg := os.Getenv("CREDENTIALS_CONFIG"); cfg != "" {
		raw, err := base64.StdEncoding.DecodeString(cfg)
		if err != nil {
			return nil, fmt.Errorf("decode CREDENTIALS_CONFIG: %w", err)
		}
		creds, err := google.CredentialsFromJSON(ctx, raw, oauthScopes...)
		if err != nil {
			return nil, fmt.Errorf("parse CREDENTIALS_CONFIG: %w", err)
		}
		log.Println("Using credentials from CREDENTIALS_CONFIG")
		return creds.TokenSource, nil
	}

	saPath := envOrDefault("SERVICE_ACCOUNT_PATH", "service_account.json")
	if fileExists(saPath) {
		raw, err := os.ReadFile(saPath)
		if err == nil {
			creds, err := google.CredentialsFromJSON(ctx, raw, oauthScopes...)
			if err == nil {
				log.Printf("Using service account authentication (%s)", saPath)
				return creds.TokenSource, nil
			}
			log.Printf("Service account at %s could not be loaded: %v", saPath, err)
		}
	}

	credsPath := envOrDefault("CREDENTIALS_PATH", "credentials.json")
	tokenPath := envOrDefault("TOKEN_PATH", "token.json")
	if fileExists(credsPath) {
		ts, err := oauthTokenSource(ctx, credsPath, tokenPath)
		if err == nil {
			return ts, nil
		}
		log.Printf("OAuth flow failed: %v — falling through to ADC", err)
	}

	log.Println("Falling back to Application Default Credentials")
	creds, err := google.FindDefaultCredentials(ctx, oauthScopes...)
	if err != nil {
		return nil, fmt.Errorf("no credentials available: %w", err)
	}
	return creds.TokenSource, nil
}

func oauthTokenSource(ctx context.Context, credsPath, tokenPath string) (oauth2.TokenSource, error) {
	raw, err := os.ReadFile(credsPath)
	if err != nil {
		return nil, fmt.Errorf("read OAuth client: %w", err)
	}
	cfg, err := google.ConfigFromJSON(raw, oauthScopes...)
	if err != nil {
		return nil, fmt.Errorf("parse OAuth client: %w", err)
	}

	if tok, err := loadToken(tokenPath); err == nil {
		ts := cfg.TokenSource(ctx, tok)
		// Force a refresh to validate.
		if refreshed, err := ts.Token(); err == nil {
			if refreshed.AccessToken != tok.AccessToken {
				_ = saveToken(tokenPath, refreshed)
			}
			return ts, nil
		}
		log.Println("Cached OAuth token invalid; restarting interactive flow")
	}

	tok, err := interactiveOAuthFlow(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := saveToken(tokenPath, tok); err != nil {
		log.Printf("Warning: failed to persist OAuth token: %v", err)
	}
	return cfg.TokenSource(ctx, tok), nil
}

func interactiveOAuthFlow(ctx context.Context, cfg *oauth2.Config) (*oauth2.Token, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("bind callback listener: %w", err)
	}
	addr := listener.Addr().(*net.TCPAddr)
	redirect := fmt.Sprintf("http://127.0.0.1:%d/callback", addr.Port)
	cfg.RedirectURL = redirect

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if e := r.URL.Query().Get("error"); e != "" {
			errCh <- fmt.Errorf("oauth callback error: %s", e)
			http.Error(w, e, http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("oauth callback missing code")
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}
		_, _ = io.WriteString(w, "Authentication successful — you can close this tab.")
		codeCh <- code
	})
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(listener) }()
	defer srv.Shutdown(context.Background())

	url := cfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Fprintf(os.Stderr, "Open this URL in your browser to authorize:\n%s\n", url)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		return nil, err
	case code := <-codeCh:
		return cfg.Exchange(ctx, code)
	}
}

func loadToken(path string) (*oauth2.Token, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(tok); err != nil {
		return nil, err
	}
	return tok, nil
}

// saveToken writes the OAuth token with restrictive (0600) permissions to
// avoid leaking the refresh token on multi-user hosts. This addresses one of
// the findings of the security review against the upstream Python project.
func saveToken(path string, tok *oauth2.Token) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(tok)
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
