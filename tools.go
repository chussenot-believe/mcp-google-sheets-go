package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
)

type toolDeps struct {
	sheets   *sheets.Service
	drive    *drive.Service
	folderID string
	enabled  map[string]bool // nil => all enabled
}

func (d *toolDeps) isEnabled(name string) bool {
	if d.enabled == nil {
		return true
	}
	return d.enabled[name]
}

func registerAllTools(s *server.MCPServer, d *toolDeps) {
	registerGetSheetData(s, d)
	registerGetSheetFormulas(s, d)
	registerUpdateCells(s, d)
	registerBatchUpdateCells(s, d)
	registerAddRows(s, d)
	registerAddColumns(s, d)
	registerListSheets(s, d)
	registerCopySheet(s, d)
	registerRenameSheet(s, d)
	registerGetMultipleSheetData(s, d)
	registerGetMultipleSpreadsheetSummary(s, d)
	registerCreateSpreadsheet(s, d)
	registerCreateSheet(s, d)
	registerListSpreadsheets(s, d)
	registerShareSpreadsheet(s, d)
	registerListFolders(s, d)
	registerSearchSpreadsheets(s, d)
	registerFindInSpreadsheet(s, d)
	registerBatchUpdate(s, d)
	registerAddChart(s, d)
}

// -- Argument helpers ---------------------------------------------------------

func argString(args map[string]any, key string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}

func argStringRequired(args map[string]any, key string) (string, error) {
	v, ok := args[key].(string)
	if !ok || v == "" {
		return "", fmt.Errorf("missing required argument: %s", key)
	}
	return v, nil
}

func argBool(args map[string]any, key string, def bool) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return def
}

func argInt(args map[string]any, key string, def int) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	}
	return def
}

func argIntPtr(args map[string]any, key string) *int {
	if _, ok := args[key]; !ok {
		return nil
	}
	v := argInt(args, key, 0)
	return &v
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshal: %v", err)), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

func errResult(format string, a ...any) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError(fmt.Sprintf(format, a...)), nil
}

// -- Sheet ID lookup ----------------------------------------------------------

func getSheetID(ctx context.Context, svc *sheets.Service, spreadsheetID, sheetName string) (int64, error) {
	ss, err := svc.Spreadsheets.Get(spreadsheetID).Context(ctx).
		Fields("sheets(properties(title,sheetId))").Do()
	if err != nil {
		return 0, err
	}
	for _, sh := range ss.Sheets {
		if sh.Properties != nil && sh.Properties.Title == sheetName {
			return sh.Properties.SheetId, nil
		}
	}
	return 0, fmt.Errorf("sheet %q not found", sheetName)
}

// -- Tools: read --------------------------------------------------------------

func registerGetSheetData(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("get_sheet_data") {
		return
	}
	tool := mcp.NewTool("get_sheet_data",
		mcp.WithDescription("Get data from a specific sheet in a Google Spreadsheet."),
		mcp.WithString("spreadsheet_id", mcp.Required(), mcp.Description("Spreadsheet ID from the URL")),
		mcp.WithString("sheet", mcp.Required(), mcp.Description("Sheet/tab name")),
		mcp.WithString("range", mcp.Description("Optional A1 notation range")),
		mcp.WithBoolean("include_grid_data", mcp.Description("Include full grid metadata (larger response)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		sid, err := argStringRequired(args, "spreadsheet_id")
		if err != nil {
			return errResult("%v", err)
		}
		sheet, err := argStringRequired(args, "sheet")
		if err != nil {
			return errResult("%v", err)
		}
		fullRange := sheet
		if r := argString(args, "range"); r != "" {
			fullRange = fmt.Sprintf("%s!%s", sheet, r)
		}
		if argBool(args, "include_grid_data", false) {
			res, err := d.sheets.Spreadsheets.Get(sid).
				Ranges(fullRange).IncludeGridData(true).Context(ctx).Do()
			if err != nil {
				return errResult("%v", err)
			}
			return jsonResult(res)
		}
		res, err := d.sheets.Spreadsheets.Values.Get(sid, fullRange).Context(ctx).Do()
		if err != nil {
			return errResult("%v", err)
		}
		return jsonResult(map[string]any{
			"spreadsheetId": sid,
			"valueRanges": []map[string]any{{
				"range":  fullRange,
				"values": res.Values,
			}},
		})
	})
}

func registerGetSheetFormulas(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("get_sheet_formulas") {
		return
	}
	tool := mcp.NewTool("get_sheet_formulas",
		mcp.WithDescription("Get formulas from a specific sheet."),
		mcp.WithString("spreadsheet_id", mcp.Required()),
		mcp.WithString("sheet", mcp.Required()),
		mcp.WithString("range"),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		sid, err := argStringRequired(args, "spreadsheet_id")
		if err != nil {
			return errResult("%v", err)
		}
		sheet, err := argStringRequired(args, "sheet")
		if err != nil {
			return errResult("%v", err)
		}
		fullRange := sheet
		if r := argString(args, "range"); r != "" {
			fullRange = fmt.Sprintf("%s!%s", sheet, r)
		}
		res, err := d.sheets.Spreadsheets.Values.Get(sid, fullRange).
			ValueRenderOption("FORMULA").Context(ctx).Do()
		if err != nil {
			return errResult("%v", err)
		}
		return jsonResult(res.Values)
	})
}

func registerListSheets(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("list_sheets") {
		return
	}
	tool := mcp.NewTool("list_sheets",
		mcp.WithDescription("List all sheet/tab names in a spreadsheet."),
		mcp.WithString("spreadsheet_id", mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sid, err := argStringRequired(req.GetArguments(), "spreadsheet_id")
		if err != nil {
			return errResult("%v", err)
		}
		ss, err := d.sheets.Spreadsheets.Get(sid).
			Fields("sheets(properties(title))").Context(ctx).Do()
		if err != nil {
			return errResult("%v", err)
		}
		names := make([]string, 0, len(ss.Sheets))
		for _, sh := range ss.Sheets {
			if sh.Properties != nil {
				names = append(names, sh.Properties.Title)
			}
		}
		return jsonResult(names)
	})
}

func registerGetMultipleSheetData(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("get_multiple_sheet_data") {
		return
	}
	tool := mcp.NewTool("get_multiple_sheet_data",
		mcp.WithDescription("Get data from multiple ranges across spreadsheets."),
		mcp.WithArray("queries", mcp.Required(),
			mcp.Description("List of {spreadsheet_id, sheet, range} objects")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		queries, ok := args["queries"].([]any)
		if !ok {
			return errResult("queries must be an array")
		}
		results := make([]map[string]any, 0, len(queries))
		for _, raw := range queries {
			q, _ := raw.(map[string]any)
			sid, _ := q["spreadsheet_id"].(string)
			sheet, _ := q["sheet"].(string)
			rng, _ := q["range"].(string)
			entry := map[string]any{
				"spreadsheet_id": sid,
				"sheet":          sheet,
				"range":          rng,
			}
			if sid == "" || sheet == "" || rng == "" {
				entry["error"] = "Missing required keys (spreadsheet_id, sheet, range)"
				results = append(results, entry)
				continue
			}
			fullRange := fmt.Sprintf("%s!%s", sheet, rng)
			res, err := d.sheets.Spreadsheets.Values.Get(sid, fullRange).Context(ctx).Do()
			if err != nil {
				entry["error"] = err.Error()
			} else {
				entry["data"] = res.Values
			}
			results = append(results, entry)
		}
		return jsonResult(results)
	})
}

func registerGetMultipleSpreadsheetSummary(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("get_multiple_spreadsheet_summary") {
		return
	}
	tool := mcp.NewTool("get_multiple_spreadsheet_summary",
		mcp.WithDescription("Get title, sheet names, headers, and first rows for multiple spreadsheets."),
		mcp.WithArray("spreadsheet_ids", mcp.Required()),
		mcp.WithNumber("rows_to_fetch", mcp.Description("How many rows including header (default 5)")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		idsAny, ok := args["spreadsheet_ids"].([]any)
		if !ok {
			return errResult("spreadsheet_ids must be an array")
		}
		rowsToFetch := argInt(args, "rows_to_fetch", 5)
		if rowsToFetch < 1 {
			rowsToFetch = 1
		}

		summaries := make([]map[string]any, 0, len(idsAny))
		for _, raw := range idsAny {
			sid, _ := raw.(string)
			summary := map[string]any{"spreadsheet_id": sid, "sheets": []any{}, "title": nil, "error": nil}
			if sid == "" {
				summary["error"] = "empty spreadsheet_id"
				summaries = append(summaries, summary)
				continue
			}
			ss, err := d.sheets.Spreadsheets.Get(sid).
				Fields("properties.title,sheets(properties(title,sheetId))").Context(ctx).Do()
			if err != nil {
				summary["error"] = fmt.Sprintf("Error fetching spreadsheet %s: %v", sid, err)
				summaries = append(summaries, summary)
				continue
			}
			if ss.Properties != nil {
				summary["title"] = ss.Properties.Title
			}
			sheetSummaries := []map[string]any{}
			for _, sh := range ss.Sheets {
				if sh.Properties == nil {
					continue
				}
				title := sh.Properties.Title
				entry := map[string]any{
					"title":      title,
					"sheet_id":   sh.Properties.SheetId,
					"headers":    []any{},
					"first_rows": []any{},
				}
				if title == "" {
					entry["error"] = "Sheet title not found"
					sheetSummaries = append(sheetSummaries, entry)
					continue
				}
				rng := fmt.Sprintf("%s!A1:%d", title, rowsToFetch)
				res, err := d.sheets.Spreadsheets.Values.Get(sid, rng).Context(ctx).Do()
				if err != nil {
					entry["error"] = fmt.Sprintf("Error fetching data for sheet %s: %v", title, err)
				} else if len(res.Values) > 0 {
					entry["headers"] = res.Values[0]
					if len(res.Values) > 1 {
						entry["first_rows"] = res.Values[1:]
					}
				}
				sheetSummaries = append(sheetSummaries, entry)
			}
			summary["sheets"] = sheetSummaries
			summaries = append(summaries, summary)
		}
		return jsonResult(summaries)
	})
}

func registerFindInSpreadsheet(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("find_in_spreadsheet") {
		return
	}
	tool := mcp.NewTool("find_in_spreadsheet",
		mcp.WithDescription("Find cells containing a value in a spreadsheet."),
		mcp.WithString("spreadsheet_id", mcp.Required()),
		mcp.WithString("query", mcp.Required()),
		mcp.WithString("sheet", mcp.Description("Optional sheet to limit search")),
		mcp.WithBoolean("case_sensitive"),
		mcp.WithNumber("max_results", mcp.Description("Default 50")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		sid, err := argStringRequired(args, "spreadsheet_id")
		if err != nil {
			return errResult("%v", err)
		}
		query, err := argStringRequired(args, "query")
		if err != nil {
			return errResult("%v", err)
		}
		sheetFilter := argString(args, "sheet")
		caseSensitive := argBool(args, "case_sensitive", false)
		maxResults := argInt(args, "max_results", 50)

		ss, err := d.sheets.Spreadsheets.Get(sid).
			Fields("sheets(properties(title,sheetId))").Context(ctx).Do()
		if err != nil {
			return errResult("%v", err)
		}
		var toSearch []string
		for _, sh := range ss.Sheets {
			if sh.Properties == nil {
				continue
			}
			if sheetFilter == "" || sh.Properties.Title == sheetFilter {
				toSearch = append(toSearch, sh.Properties.Title)
			}
		}
		if len(toSearch) == 0 {
			return errResult("Sheet %q not found", sheetFilter)
		}
		needle := query
		if !caseSensitive {
			needle = strings.ToLower(query)
		}
		results := []map[string]any{}
		for _, name := range toSearch {
			if len(results) >= maxResults {
				break
			}
			res, err := d.sheets.Spreadsheets.Values.Get(sid, name).Context(ctx).Do()
			if err != nil {
				continue
			}
			for rowIdx, row := range res.Values {
				if len(results) >= maxResults {
					break
				}
				for colIdx, cell := range row {
					if len(results) >= maxResults {
						break
					}
					cellStr := fmt.Sprintf("%v", cell)
					hay := cellStr
					if !caseSensitive {
						hay = strings.ToLower(cellStr)
					}
					if strings.Contains(hay, needle) {
						results = append(results, map[string]any{
							"sheet": name,
							"cell":  fmt.Sprintf("%s%d", columnIndexToLetter(colIdx), rowIdx+1),
							"value": cell,
						})
					}
				}
			}
		}
		return jsonResult(results)
	})
}

// -- Tools: write -------------------------------------------------------------

// chooseValueInputOption defaults to RAW so a malicious or accidental "=..."
// string is not silently evaluated as a Sheets formula (which can call
// IMPORTDATA / IMPORTXML / HYPERLINK to exfiltrate data). Callers must opt in
// to formula evaluation by passing "USER_ENTERED".
func chooseValueInputOption(args map[string]any) string {
	if v, ok := args["value_input_option"].(string); ok {
		switch strings.ToUpper(v) {
		case "USER_ENTERED":
			return "USER_ENTERED"
		case "RAW":
			return "RAW"
		}
	}
	return "RAW"
}

func registerUpdateCells(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("update_cells") {
		return
	}
	tool := mcp.NewTool("update_cells",
		mcp.WithDescription("Write data to a range. Values are written as RAW by default; pass value_input_option=USER_ENTERED to interpret formulas."),
		mcp.WithString("spreadsheet_id", mcp.Required()),
		mcp.WithString("sheet", mcp.Required()),
		mcp.WithString("range", mcp.Required()),
		mcp.WithArray("data", mcp.Required(), mcp.Description("2D array of values")),
		mcp.WithString("value_input_option", mcp.Description("RAW (default) or USER_ENTERED")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		sid, err := argStringRequired(args, "spreadsheet_id")
		if err != nil {
			return errResult("%v", err)
		}
		sheet, err := argStringRequired(args, "sheet")
		if err != nil {
			return errResult("%v", err)
		}
		rng, err := argStringRequired(args, "range")
		if err != nil {
			return errResult("%v", err)
		}
		dataAny, ok := args["data"].([]any)
		if !ok {
			return errResult("data must be a 2D array")
		}
		values := toValueMatrix(dataAny)
		fullRange := fmt.Sprintf("%s!%s", sheet, rng)
		res, err := d.sheets.Spreadsheets.Values.
			Update(sid, fullRange, &sheets.ValueRange{Values: values}).
			ValueInputOption(chooseValueInputOption(args)).
			Context(ctx).Do()
		if err != nil {
			return errResult("%v", err)
		}
		return jsonResult(res)
	})
}

func registerBatchUpdateCells(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("batch_update_cells") {
		return
	}
	tool := mcp.NewTool("batch_update_cells",
		mcp.WithDescription("Update multiple ranges in one call. RAW value input by default."),
		mcp.WithString("spreadsheet_id", mcp.Required()),
		mcp.WithString("sheet", mcp.Required()),
		mcp.WithObject("ranges", mcp.Required(),
			mcp.Description("Map of range string => 2D array of values")),
		mcp.WithString("value_input_option", mcp.Description("RAW (default) or USER_ENTERED")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		sid, err := argStringRequired(args, "spreadsheet_id")
		if err != nil {
			return errResult("%v", err)
		}
		sheet, err := argStringRequired(args, "sheet")
		if err != nil {
			return errResult("%v", err)
		}
		rangesObj, ok := args["ranges"].(map[string]any)
		if !ok {
			return errResult("ranges must be an object")
		}
		var data []*sheets.ValueRange
		for rng, v := range rangesObj {
			vals, _ := v.([]any)
			data = append(data, &sheets.ValueRange{
				Range:  fmt.Sprintf("%s!%s", sheet, rng),
				Values: toValueMatrix(vals),
			})
		}
		res, err := d.sheets.Spreadsheets.Values.BatchUpdate(sid, &sheets.BatchUpdateValuesRequest{
			ValueInputOption: chooseValueInputOption(args),
			Data:             data,
		}).Context(ctx).Do()
		if err != nil {
			return errResult("%v", err)
		}
		return jsonResult(res)
	})
}

func registerAddRows(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("add_rows") {
		return
	}
	tool := mcp.NewTool("add_rows",
		mcp.WithDescription("Insert empty rows at a position."),
		mcp.WithString("spreadsheet_id", mcp.Required()),
		mcp.WithString("sheet", mcp.Required()),
		mcp.WithNumber("count", mcp.Required()),
		mcp.WithNumber("start_row"),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return insertDimension(ctx, d, req, "ROWS")
	})
}

func registerAddColumns(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("add_columns") {
		return
	}
	tool := mcp.NewTool("add_columns",
		mcp.WithDescription("Insert empty columns at a position."),
		mcp.WithString("spreadsheet_id", mcp.Required()),
		mcp.WithString("sheet", mcp.Required()),
		mcp.WithNumber("count", mcp.Required()),
		mcp.WithNumber("start_column"),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return insertDimension(ctx, d, req, "COLUMNS")
	})
}

func insertDimension(ctx context.Context, d *toolDeps, req mcp.CallToolRequest, dimension string) (*mcp.CallToolResult, error) {
	args := req.GetArguments()
	sid, err := argStringRequired(args, "spreadsheet_id")
	if err != nil {
		return errResult("%v", err)
	}
	sheet, err := argStringRequired(args, "sheet")
	if err != nil {
		return errResult("%v", err)
	}
	count := argInt(args, "count", 0)
	if count <= 0 {
		return errResult("count must be a positive integer")
	}
	startKey := "start_row"
	if dimension == "COLUMNS" {
		startKey = "start_column"
	}
	startPtr := argIntPtr(args, startKey)
	start := 0
	if startPtr != nil {
		start = *startPtr
	}
	sheetID, err := getSheetID(ctx, d.sheets, sid, sheet)
	if err != nil {
		return errResult("%v", err)
	}
	res, err := d.sheets.Spreadsheets.BatchUpdate(sid, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{{
			InsertDimension: &sheets.InsertDimensionRequest{
				Range: &sheets.DimensionRange{
					SheetId:    sheetID,
					Dimension:  dimension,
					StartIndex: int64(start),
					EndIndex:   int64(start + count),
				},
				InheritFromBefore: startPtr != nil && start > 0,
			},
		}},
	}).Context(ctx).Do()
	if err != nil {
		return errResult("%v", err)
	}
	return jsonResult(res)
}

func registerCopySheet(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("copy_sheet") {
		return
	}
	tool := mcp.NewTool("copy_sheet",
		mcp.WithDescription("Copy a sheet from one spreadsheet to another."),
		mcp.WithString("src_spreadsheet", mcp.Required()),
		mcp.WithString("src_sheet", mcp.Required()),
		mcp.WithString("dst_spreadsheet", mcp.Required()),
		mcp.WithString("dst_sheet", mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		srcSS, err := argStringRequired(args, "src_spreadsheet")
		if err != nil {
			return errResult("%v", err)
		}
		srcSheet, err := argStringRequired(args, "src_sheet")
		if err != nil {
			return errResult("%v", err)
		}
		dstSS, err := argStringRequired(args, "dst_spreadsheet")
		if err != nil {
			return errResult("%v", err)
		}
		dstSheet, err := argStringRequired(args, "dst_sheet")
		if err != nil {
			return errResult("%v", err)
		}
		srcSheetID, err := getSheetID(ctx, d.sheets, srcSS, srcSheet)
		if err != nil {
			return errResult("source sheet: %v", err)
		}
		copied, err := d.sheets.Spreadsheets.Sheets.CopyTo(srcSS, srcSheetID,
			&sheets.CopySheetToAnotherSpreadsheetRequest{DestinationSpreadsheetId: dstSS}).
			Context(ctx).Do()
		if err != nil {
			return errResult("%v", err)
		}
		out := map[string]any{"copy": copied}
		if copied != nil && copied.Title != dstSheet {
			renamed, err := d.sheets.Spreadsheets.BatchUpdate(dstSS, &sheets.BatchUpdateSpreadsheetRequest{
				Requests: []*sheets.Request{{
					UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
						Properties: &sheets.SheetProperties{
							SheetId: copied.SheetId,
							Title:   dstSheet,
						},
						Fields: "title",
					},
				}},
			}).Context(ctx).Do()
			if err != nil {
				return errResult("rename: %v", err)
			}
			out["rename"] = renamed
		}
		return jsonResult(out)
	})
}

func registerRenameSheet(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("rename_sheet") {
		return
	}
	tool := mcp.NewTool("rename_sheet",
		mcp.WithDescription("Rename an existing sheet/tab."),
		mcp.WithString("spreadsheet", mcp.Required()),
		mcp.WithString("sheet", mcp.Required()),
		mcp.WithString("new_name", mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		sid, err := argStringRequired(args, "spreadsheet")
		if err != nil {
			return errResult("%v", err)
		}
		sheet, err := argStringRequired(args, "sheet")
		if err != nil {
			return errResult("%v", err)
		}
		newName, err := argStringRequired(args, "new_name")
		if err != nil {
			return errResult("%v", err)
		}
		sheetID, err := getSheetID(ctx, d.sheets, sid, sheet)
		if err != nil {
			return errResult("%v", err)
		}
		res, err := d.sheets.Spreadsheets.BatchUpdate(sid, &sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{{
				UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
					Properties: &sheets.SheetProperties{SheetId: sheetID, Title: newName},
					Fields:     "title",
				},
			}},
		}).Context(ctx).Do()
		if err != nil {
			return errResult("%v", err)
		}
		return jsonResult(res)
	})
}

func registerCreateSheet(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("create_sheet") {
		return
	}
	tool := mcp.NewTool("create_sheet",
		mcp.WithDescription("Add a new sheet/tab to a spreadsheet."),
		mcp.WithString("spreadsheet_id", mcp.Required()),
		mcp.WithString("title", mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		sid, err := argStringRequired(args, "spreadsheet_id")
		if err != nil {
			return errResult("%v", err)
		}
		title, err := argStringRequired(args, "title")
		if err != nil {
			return errResult("%v", err)
		}
		res, err := d.sheets.Spreadsheets.BatchUpdate(sid, &sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{{
				AddSheet: &sheets.AddSheetRequest{
					Properties: &sheets.SheetProperties{Title: title},
				},
			}},
		}).Context(ctx).Do()
		if err != nil {
			return errResult("%v", err)
		}
		newProps := res.Replies[0].AddSheet.Properties
		return jsonResult(map[string]any{
			"sheetId":       newProps.SheetId,
			"title":         newProps.Title,
			"index":         newProps.Index,
			"spreadsheetId": sid,
		})
	})
}

func registerBatchUpdate(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("batch_update") {
		return
	}
	tool := mcp.NewTool("batch_update",
		mcp.WithDescription("Execute an arbitrary spreadsheets.batchUpdate. Pass `requests` as a list of request objects."),
		mcp.WithString("spreadsheet_id", mcp.Required()),
		mcp.WithArray("requests", mcp.Required()),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		sid, err := argStringRequired(args, "spreadsheet_id")
		if err != nil {
			return errResult("%v", err)
		}
		reqsAny, ok := args["requests"].([]any)
		if !ok || len(reqsAny) == 0 {
			return errResult("requests list cannot be empty")
		}
		// Re-marshal through JSON into typed requests for the API client.
		raw, err := json.Marshal(map[string]any{"requests": reqsAny})
		if err != nil {
			return errResult("marshal requests: %v", err)
		}
		body := &sheets.BatchUpdateSpreadsheetRequest{}
		if err := json.Unmarshal(raw, body); err != nil {
			return errResult("invalid requests: %v", err)
		}
		res, err := d.sheets.Spreadsheets.BatchUpdate(sid, body).Context(ctx).Do()
		if err != nil {
			return errResult("%v", err)
		}
		return jsonResult(res)
	})
}

// -- Tools: Drive -------------------------------------------------------------

func registerCreateSpreadsheet(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("create_spreadsheet") {
		return
	}
	tool := mcp.NewTool("create_spreadsheet",
		mcp.WithDescription("Create a new spreadsheet, optionally in a specified folder."),
		mcp.WithString("title", mcp.Required()),
		mcp.WithString("folder_id"),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		title, err := argStringRequired(args, "title")
		if err != nil {
			return errResult("%v", err)
		}
		folderID := argString(args, "folder_id")
		if folderID == "" {
			folderID = d.folderID
		}
		file := &drive.File{
			Name:     title,
			MimeType: "application/vnd.google-apps.spreadsheet",
		}
		if folderID != "" {
			file.Parents = []string{folderID}
		}
		created, err := d.drive.Files.Create(file).
			SupportsAllDrives(true).Fields("id,name,parents").Context(ctx).Do()
		if err != nil {
			return errResult("%v", err)
		}
		folder := "root"
		if len(created.Parents) > 0 {
			folder = created.Parents[0]
		}
		return jsonResult(map[string]any{
			"spreadsheetId": created.Id,
			"title":         created.Name,
			"folder":        folder,
		})
	})
}

func registerListSpreadsheets(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("list_spreadsheets") {
		return
	}
	tool := mcp.NewTool("list_spreadsheets",
		mcp.WithDescription("List spreadsheets in the configured (or specified) Drive folder."),
		mcp.WithString("folder_id"),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		folderID := argString(req.GetArguments(), "folder_id")
		if folderID == "" {
			folderID = d.folderID
		}
		q := "mimeType='application/vnd.google-apps.spreadsheet'"
		if folderID != "" {
			q += fmt.Sprintf(" and '%s' in parents", escapeDriveQuery(folderID))
		}
		res, err := d.drive.Files.List().
			Q(q).Spaces("drive").
			IncludeItemsFromAllDrives(true).SupportsAllDrives(true).
			Fields("files(id,name)").OrderBy("modifiedTime desc").Context(ctx).Do()
		if err != nil {
			return errResult("%v", err)
		}
		out := make([]map[string]string, 0, len(res.Files))
		for _, f := range res.Files {
			out = append(out, map[string]string{"id": f.Id, "title": f.Name})
		}
		return jsonResult(out)
	})
}

func registerListFolders(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("list_folders") {
		return
	}
	tool := mcp.NewTool("list_folders",
		mcp.WithDescription("List folders within the given parent (or My Drive root)."),
		mcp.WithString("parent_folder_id"),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		parent := argString(req.GetArguments(), "parent_folder_id")
		q := "mimeType='application/vnd.google-apps.folder'"
		if parent != "" {
			q += fmt.Sprintf(" and '%s' in parents", escapeDriveQuery(parent))
		} else {
			q += " and 'root' in parents"
		}
		res, err := d.drive.Files.List().
			Q(q).Spaces("drive").
			IncludeItemsFromAllDrives(true).SupportsAllDrives(true).
			Fields("files(id,name,parents)").OrderBy("name").Context(ctx).Do()
		if err != nil {
			return errResult("%v", err)
		}
		out := make([]map[string]string, 0, len(res.Files))
		for _, f := range res.Files {
			parentID := "root"
			if len(f.Parents) > 0 {
				parentID = f.Parents[0]
			}
			out = append(out, map[string]string{"id": f.Id, "name": f.Name, "parent": parentID})
		}
		return jsonResult(out)
	})
}

func registerSearchSpreadsheets(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("search_spreadsheets") {
		return
	}
	tool := mcp.NewTool("search_spreadsheets",
		mcp.WithDescription("Search spreadsheets by name or content."),
		mcp.WithString("query", mcp.Required()),
		mcp.WithNumber("max_results", mcp.Description("1-100, default 20")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		query, err := argStringRequired(args, "query")
		if err != nil {
			return errResult("%v", err)
		}
		max := argInt(args, "max_results", 20)
		if max < 1 {
			max = 1
		}
		if max > 100 {
			max = 100
		}
		safe := escapeDriveQuery(query)
		q := fmt.Sprintf(
			"mimeType='application/vnd.google-apps.spreadsheet' and (name contains '%s' or fullText contains '%s')",
			safe, safe,
		)
		res, err := d.drive.Files.List().
			Q(q).PageSize(int64(max)).Spaces("drive").
			IncludeItemsFromAllDrives(true).SupportsAllDrives(true).
			Fields("files(id,name,createdTime,modifiedTime,owners,webViewLink)").
			OrderBy("modifiedTime desc").Context(ctx).Do()
		if err != nil {
			return errResult("search failed: %v", err)
		}
		out := make([]map[string]any, 0, len(res.Files))
		for _, f := range res.Files {
			owners := []string{}
			for _, o := range f.Owners {
				owners = append(owners, o.EmailAddress)
			}
			out = append(out, map[string]any{
				"id":            f.Id,
				"name":          f.Name,
				"created_time":  f.CreatedTime,
				"modified_time": f.ModifiedTime,
				"owners":        owners,
				"web_link":      f.WebViewLink,
			})
		}
		return jsonResult(out)
	})
}

func registerShareSpreadsheet(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("share_spreadsheet") {
		return
	}
	tool := mcp.NewTool("share_spreadsheet",
		mcp.WithDescription("Share a spreadsheet with users. If ALLOWED_SHARE_DOMAINS is set, recipients outside those domains are rejected."),
		mcp.WithString("spreadsheet_id", mcp.Required()),
		mcp.WithArray("recipients", mcp.Required(),
			mcp.Description("List of {email_address, role} objects. Role: reader|commenter|writer")),
		mcp.WithBoolean("send_notification", mcp.Description("Default true")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		sid, err := argStringRequired(args, "spreadsheet_id")
		if err != nil {
			return errResult("%v", err)
		}
		recipientsAny, ok := args["recipients"].([]any)
		if !ok {
			return errResult("recipients must be an array")
		}
		notify := argBool(args, "send_notification", true)
		allowedDomains := parseAllowedDomains()

		successes := []map[string]any{}
		failures := []map[string]any{}
		for _, raw := range recipientsAny {
			rcpt, _ := raw.(map[string]any)
			email, _ := rcpt["email_address"].(string)
			role, _ := rcpt["role"].(string)
			if role == "" {
				role = "writer"
			}
			if email == "" {
				failures = append(failures, map[string]any{
					"email_address": nil, "error": "Missing email_address",
				})
				continue
			}
			if role != "reader" && role != "commenter" && role != "writer" {
				failures = append(failures, map[string]any{
					"email_address": email,
					"error":         fmt.Sprintf("Invalid role %q. Must be reader, commenter, or writer.", role),
				})
				continue
			}
			if allowedDomains != nil && !domainAllowed(email, allowedDomains) {
				failures = append(failures, map[string]any{
					"email_address": email,
					"error":         "Recipient domain not in ALLOWED_SHARE_DOMAINS",
				})
				continue
			}
			permission := &drive.Permission{
				Type:         "user",
				Role:         role,
				EmailAddress: email,
			}
			created, err := d.drive.Permissions.Create(sid, permission).
				SendNotificationEmail(notify).Fields("id").Context(ctx).Do()
			if err != nil {
				failures = append(failures, map[string]any{
					"email_address": email, "error": fmt.Sprintf("Failed to share: %v", err),
				})
				continue
			}
			successes = append(successes, map[string]any{
				"email_address": email,
				"role":          role,
				"permissionId":  created.Id,
			})
		}
		return jsonResult(map[string]any{"successes": successes, "failures": failures})
	})
}

// parseAllowedDomains returns nil if ALLOWED_SHARE_DOMAINS is unset (i.e., no
// restriction); otherwise a lowercase set of permitted domains.
func parseAllowedDomains() map[string]bool {
	raw := os.Getenv("ALLOWED_SHARE_DOMAINS")
	if raw == "" {
		return nil
	}
	out := map[string]bool{}
	for _, d := range strings.Split(raw, ",") {
		d = strings.TrimSpace(strings.ToLower(d))
		if d != "" {
			out[d] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func domainAllowed(email string, allowed map[string]bool) bool {
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return false
	}
	return allowed[strings.ToLower(email[at+1:])]
}

// -- Tool: chart --------------------------------------------------------------

func registerAddChart(s *server.MCPServer, d *toolDeps) {
	if !d.isEnabled("add_chart") {
		return
	}
	tool := mcp.NewTool("add_chart",
		mcp.WithDescription("Add a chart to a spreadsheet."),
		mcp.WithString("spreadsheet_id", mcp.Required()),
		mcp.WithString("sheet", mcp.Required()),
		mcp.WithString("chart_type", mcp.Required(),
			mcp.Description("COLUMN|BAR|LINE|AREA|PIE|SCATTER|COMBO|HISTOGRAM")),
		mcp.WithString("data_range", mcp.Required()),
		mcp.WithString("title"),
		mcp.WithString("x_axis_label"),
		mcp.WithString("y_axis_label"),
		mcp.WithNumber("position_x"),
		mcp.WithNumber("position_y"),
		mcp.WithNumber("width"),
		mcp.WithNumber("height"),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		sid, err := argStringRequired(args, "spreadsheet_id")
		if err != nil {
			return errResult("%v", err)
		}
		sheet, err := argStringRequired(args, "sheet")
		if err != nil {
			return errResult("%v", err)
		}
		chartType := strings.ToUpper(argString(args, "chart_type"))
		valid := map[string]bool{
			"COLUMN": true, "BAR": true, "LINE": true, "AREA": true,
			"PIE": true, "SCATTER": true, "COMBO": true, "HISTOGRAM": true,
		}
		if !valid[chartType] {
			return errResult("Invalid chart type %q", chartType)
		}
		dataRange, err := argStringRequired(args, "data_range")
		if err != nil {
			return errResult("%v", err)
		}
		sheetID, err := getSheetID(ctx, d.sheets, sid, sheet)
		if err != nil {
			return errResult("%v", err)
		}
		idx, err := parseA1Notation(dataRange)
		if err != nil {
			return errResult("%v", err)
		}

		title := argString(args, "title")
		xLabel := argString(args, "x_axis_label")
		yLabel := argString(args, "y_axis_label")
		width := int64(argInt(args, "width", 600))
		height := int64(argInt(args, "height", 400))
		posX := int64(argInt(args, "position_x", 0))
		posY := int64(argInt(args, "position_y", 0))

		source := &sheets.GridRange{SheetId: sheetID}
		if idx.StartRowIndex != nil {
			source.StartRowIndex = *idx.StartRowIndex
			source.ForceSendFields = append(source.ForceSendFields, "StartRowIndex")
		}
		if idx.EndRowIndex != nil {
			source.EndRowIndex = *idx.EndRowIndex
		}
		if idx.StartColumnIndex != nil {
			source.StartColumnIndex = *idx.StartColumnIndex
			source.ForceSendFields = append(source.ForceSendFields, "StartColumnIndex")
		}
		if idx.EndColumnIndex != nil {
			source.EndColumnIndex = *idx.EndColumnIndex
		}

		domain, series := splitGridRange(source)
		spec := &sheets.ChartSpec{Title: title}

		if chartType == "PIE" {
			spec.PieChart = &sheets.PieChartSpec{
				LegendPosition: "RIGHT_LEGEND",
				Domain: &sheets.ChartData{
					SourceRange: &sheets.ChartSourceRange{Sources: []*sheets.GridRange{domain}},
				},
				Series: &sheets.ChartData{
					SourceRange: &sheets.ChartSourceRange{Sources: []*sheets.GridRange{series[0]}},
				},
			}
		} else {
			basic := &sheets.BasicChartSpec{
				ChartType:      chartType,
				LegendPosition: "RIGHT_LEGEND",
				HeaderCount:    1,
				Domains: []*sheets.BasicChartDomain{{
					Domain: &sheets.ChartData{
						SourceRange: &sheets.ChartSourceRange{Sources: []*sheets.GridRange{domain}},
					},
				}},
			}
			for _, sr := range series {
				basic.Series = append(basic.Series, &sheets.BasicChartSeries{
					Series:     &sheets.ChartData{SourceRange: &sheets.ChartSourceRange{Sources: []*sheets.GridRange{sr}}},
					TargetAxis: "LEFT_AXIS",
				})
			}
			bottom := &sheets.BasicChartAxis{Position: "BOTTOM_AXIS"}
			if xLabel != "" {
				bottom.Title = xLabel
			}
			left := &sheets.BasicChartAxis{Position: "LEFT_AXIS"}
			if yLabel != "" {
				left.Title = yLabel
			}
			basic.Axis = []*sheets.BasicChartAxis{bottom, left}
			spec.BasicChart = basic
		}

		req2 := &sheets.Request{
			AddChart: &sheets.AddChartRequest{
				Chart: &sheets.EmbeddedChart{
					Spec: spec,
					Position: &sheets.EmbeddedObjectPosition{
						OverlayPosition: &sheets.OverlayPosition{
							AnchorCell:    &sheets.GridCoordinate{SheetId: sheetID},
							OffsetXPixels: posX,
							OffsetYPixels: posY,
							WidthPixels:   width,
							HeightPixels:  height,
						},
					},
				},
			},
		}
		res, err := d.sheets.Spreadsheets.BatchUpdate(sid, &sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{req2},
		}).Context(ctx).Do()
		if err != nil {
			return errResult("Failed to add chart: %v", err)
		}
		var chartID int64
		if len(res.Replies) > 0 && res.Replies[0].AddChart != nil && res.Replies[0].AddChart.Chart != nil {
			chartID = res.Replies[0].AddChart.Chart.ChartId
		}
		msg := chartType
		if title != "" {
			msg = title
		}
		return jsonResult(map[string]any{
			"success": true,
			"message": fmt.Sprintf("Chart %q added successfully", msg),
			"chartId": chartID,
			"result":  res,
		})
	})
}

func splitGridRange(src *sheets.GridRange) (*sheets.GridRange, []*sheets.GridRange) {
	if src.EndColumnIndex-src.StartColumnIndex <= 1 {
		return src, []*sheets.GridRange{src}
	}
	domain := *src
	domain.EndColumnIndex = src.StartColumnIndex + 1
	var series []*sheets.GridRange
	for c := src.StartColumnIndex + 1; c < src.EndColumnIndex; c++ {
		copyR := *src
		copyR.StartColumnIndex = c
		copyR.EndColumnIndex = c + 1
		series = append(series, &copyR)
	}
	return &domain, series
}

// -- Conversion ---------------------------------------------------------------

func toValueMatrix(in []any) [][]any {
	out := make([][]any, 0, len(in))
	for _, row := range in {
		r, ok := row.([]any)
		if !ok {
			out = append(out, []any{row})
			continue
		}
		out = append(out, r)
	}
	return out
}
