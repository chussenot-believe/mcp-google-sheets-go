package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/server"
)

const serverName = "Google Spreadsheet"
const serverVersion = "0.1.0"

func main() {
	transport := flag.String("transport", "stdio", "MCP transport: stdio or sse")
	includeTools := flag.String("include-tools", "", "Comma-separated list of tool names to enable (default: all)")
	flag.Parse()

	enabled := parseEnabledTools(*includeTools)

	ctx := context.Background()
	sheetsSvc, driveSvc, err := authenticate(ctx)
	if err != nil {
		log.Fatalf("authentication failed: %v", err)
	}

	folderID := os.Getenv("DRIVE_FOLDER_ID")

	s := server.NewMCPServer(
		serverName,
		serverVersion,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
	)

	deps := &toolDeps{
		sheets:   sheetsSvc,
		drive:    driveSvc,
		folderID: folderID,
		enabled:  enabled,
	}
	registerAllTools(s, deps)
	registerResources(s, deps)

	switch strings.ToLower(*transport) {
	case "stdio":
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("stdio server: %v", err)
		}
	case "sse":
		// Default to 127.0.0.1 (not 0.0.0.0) to avoid accidentally exposing
		// an unauthenticated MCP endpoint to the network. Operators who need
		// remote access should opt in by setting HOST=0.0.0.0 and fronting
		// the server with their own auth.
		host := envOrDefault("HOST", envOrDefault("FASTMCP_HOST", "127.0.0.1"))
		portStr := envOrDefault("PORT", envOrDefault("FASTMCP_PORT", "8000"))
		port, err := strconv.Atoi(portStr)
		if err != nil {
			port = 8000
		}
		addr := fmt.Sprintf("%s:%d", host, port)
		log.Printf("Starting SSE transport on %s", addr)
		sse := server.NewSSEServer(s)
		if err := sse.Start(addr); err != nil {
			log.Fatalf("sse server: %v", err)
		}
	default:
		log.Fatalf("unknown transport: %s", *transport)
	}
}

// parseEnabledTools returns nil if no filtering is requested (= all tools
// enabled), otherwise a set of enabled tool names.
func parseEnabledTools(flagVal string) map[string]bool {
	raw := flagVal
	if raw == "" {
		raw = os.Getenv("ENABLED_TOOLS")
	}
	if raw == "" {
		return nil
	}
	out := map[string]bool{}
	for _, t := range strings.Split(raw, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			out[t] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
