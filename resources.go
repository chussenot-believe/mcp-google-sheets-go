package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerResources wires up the `spreadsheet://{spreadsheet_id}/info` URI
// template, matching the Python upstream. The handler returns a JSON document
// with the spreadsheet title and sheets metadata.
func registerResources(s *server.MCPServer, d *toolDeps) {
	template := mcp.NewResourceTemplate(
		"spreadsheet://{spreadsheet_id}/info",
		"Spreadsheet Info",
		mcp.WithTemplateDescription("Basic metadata for a Google Spreadsheet."),
		mcp.WithTemplateMIMEType("application/json"),
	)
	s.AddResourceTemplate(template, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		sid, err := templateArg(req, "spreadsheet_id")
		if err != nil {
			return nil, err
		}
		ss, err := d.sheets.Spreadsheets.Get(sid).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("fetch spreadsheet: %w", err)
		}
		info := map[string]any{
			"title":  "Unknown",
			"sheets": []any{},
		}
		if ss.Properties != nil && ss.Properties.Title != "" {
			info["title"] = ss.Properties.Title
		}
		sheetList := make([]map[string]any, 0, len(ss.Sheets))
		for _, sh := range ss.Sheets {
			if sh.Properties == nil {
				continue
			}
			entry := map[string]any{
				"title":   sh.Properties.Title,
				"sheetId": sh.Properties.SheetId,
			}
			if sh.Properties.GridProperties != nil {
				entry["gridProperties"] = sh.Properties.GridProperties
			}
			sheetList = append(sheetList, entry)
		}
		info["sheets"] = sheetList

		body, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal info: %w", err)
		}
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      fmt.Sprintf("spreadsheet://%s/info", sid),
				MIMEType: "application/json",
				Text:     string(body),
			},
		}, nil
	})
}

// templateArg extracts a URI template variable. mark3labs/mcp-go parses RFC
// 6570 templates and stores each variable as a []string.
func templateArg(req mcp.ReadResourceRequest, key string) (string, error) {
	raw, ok := req.Params.Arguments[key]
	if !ok {
		return "", fmt.Errorf("missing template argument: %s", key)
	}
	switch v := raw.(type) {
	case string:
		return v, nil
	case []string:
		if len(v) == 0 {
			return "", fmt.Errorf("empty template argument: %s", key)
		}
		return v[0], nil
	case []any:
		if len(v) == 0 {
			return "", fmt.Errorf("empty template argument: %s", key)
		}
		s, _ := v[0].(string)
		return s, nil
	default:
		return "", fmt.Errorf("unexpected type for template argument %s: %T", key, raw)
	}
}
