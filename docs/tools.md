# Tools reference

## Summary

This page documents every tool the server registers. Each entry follows the same shape so an agent can resolve a question about any tool by reading exactly one section.

### Shape of each entry

- **Purpose** — one sentence on what the tool does.
- **When to use** — when this tool is the right choice.
- **Parameters** — table of name, type, required flag, description.
- **Returns** — JSON shape with field documentation.
- **Errors** — observable error strings.
- **Notes** — security, performance, or behavioural caveats.

### Tool index

| Category | Tools |
| --- | --- |
| Read sheet values | [`get_sheet_data`](#get_sheet_data), [`get_sheet_formulas`](#get_sheet_formulas), [`get_multiple_sheet_data`](#get_multiple_sheet_data), [`get_multiple_spreadsheet_summary`](#get_multiple_spreadsheet_summary), [`find_in_spreadsheet`](#find_in_spreadsheet) |
| Write sheet values | [`update_cells`](#update_cells), [`batch_update_cells`](#batch_update_cells), [`batch_update`](#batch_update) |
| Sheet structure | [`list_sheets`](#list_sheets), [`create_sheet`](#create_sheet), [`rename_sheet`](#rename_sheet), [`copy_sheet`](#copy_sheet), [`add_rows`](#add_rows), [`add_columns`](#add_columns) |
| Spreadsheet lifecycle | [`create_spreadsheet`](#create_spreadsheet), [`list_spreadsheets`](#list_spreadsheets), [`search_spreadsheets`](#search_spreadsheets) |
| Drive | [`list_folders`](#list_folders), [`share_spreadsheet`](#share_spreadsheet) |
| Visualization | [`add_chart`](#add_chart) |

---

## get_sheet_data

**Purpose:** Read values (and optionally full grid metadata) from a range of a sheet.

**When to use:** The agent needs to inspect cells. Default to `include_grid_data: false` for token efficiency.

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `spreadsheet_id` | string | yes | ID from the spreadsheet URL. |
| `sheet` | string | yes | Sheet/tab name (e.g. `Sheet1`). |
| `range` | string | no | A1 notation (e.g. `A1:C10`). Omit to read the whole tab. |
| `include_grid_data` | bool | no | If `true`, return formatting and metadata. Much larger response. Default `false`. |

### Returns

If `include_grid_data` is `false`:

```json
{
  "spreadsheetId": "abc",
  "valueRanges": [
    {
      "range": "Sheet1!A1:C2",
      "values": [["h1","h2","h3"],["v1","v2","v3"]]
    }
  ]
}
```

If `include_grid_data` is `true`: the full [`spreadsheets.get`](https://developers.google.com/sheets/api/reference/rest/v4/spreadsheets/get) response shape.

### Errors

- `Unable to parse range` — sheet name or range is malformed.
- `Requested entity was not found` — wrong `spreadsheet_id`.
- `The caller does not have permission` — credentials lack access.

---

## get_sheet_formulas

**Purpose:** Read cell formulas (not their evaluated values) from a range.

**When to use:** The agent must audit or rewrite formulas, or display formula source.

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `spreadsheet_id` | string | yes | Spreadsheet ID. |
| `sheet` | string | yes | Sheet/tab name. |
| `range` | string | no | A1 range. Omit for whole tab. |

### Returns

A 2-D array of strings:

```json
[
  ["=A1+B1", "", "=SUM(C2:C10)"],
  ["=NOW()", "", ""]
]
```

Empty strings indicate cells without formulas.

---

## update_cells

**Purpose:** Write a rectangular block of values to a specific range. Overwrites existing data.

**When to use:** Targeted updates with known coordinates.

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `spreadsheet_id` | string | yes | Spreadsheet ID. |
| `sheet` | string | yes | Sheet name. |
| `range` | string | yes | A1 range (e.g. `A1:C3`). |
| `data` | array of array | yes | 2-D array of values to write. |
| `value_input_option` | string | no | `RAW` (default) or `USER_ENTERED`. |

### Returns

The Sheets API [`values.update`](https://developers.google.com/sheets/api/reference/rest/v4/spreadsheets.values/update) response:

```json
{
  "spreadsheetId": "abc",
  "updatedRange": "Sheet1!A1:C3",
  "updatedRows": 3,
  "updatedColumns": 3,
  "updatedCells": 9
}
```

### Notes

- `RAW` is the default to prevent silent formula evaluation. A string `"=IMPORTDATA(...)"` will be stored as text, not executed.
- Use `value_input_option: "USER_ENTERED"` only when the agent intentionally wants Sheets to parse strings as formulas, dates, or numbers. See [security.md](security.md#sheets-formula-injection).

---

## batch_update_cells

**Purpose:** Update several disjoint ranges in a single API call.

**When to use:** The agent needs to write to multiple ranges; one call is more efficient than several `update_cells` calls.

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `spreadsheet_id` | string | yes | Spreadsheet ID. |
| `sheet` | string | yes | Sheet name applied as prefix to every range. |
| `ranges` | object | yes | Map of A1 range → 2-D array of values. |
| `value_input_option` | string | no | `RAW` (default) or `USER_ENTERED`. |

### Example `ranges`

```json
{
  "A1:B2": [[1, 2], [3, 4]],
  "D5":    [["Hello"]]
}
```

### Returns

The Sheets API [`values.batchUpdate`](https://developers.google.com/sheets/api/reference/rest/v4/spreadsheets.values/batchUpdate) response.

### Notes

Same `RAW` default and rationale as `update_cells`.

---

## batch_update

**Purpose:** Execute an arbitrary `spreadsheets.batchUpdate` request — full Sheets API surface for structural and formatting changes.

**When to use:** The operation is not covered by a higher-level tool (e.g. conditional formatting, protected ranges, named ranges).

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `spreadsheet_id` | string | yes | Spreadsheet ID. |
| `requests` | array of object | yes | List of [Request](https://developers.google.com/sheets/api/reference/rest/v4/spreadsheets/request) objects. |

### Returns

The raw [`batchUpdate`](https://developers.google.com/sheets/api/reference/rest/v4/spreadsheets/batchUpdate) response with one reply per request.

### Errors

- `requests list cannot be empty` — pass at least one request.
- `invalid requests: …` — the request body did not match the API schema.

### Notes

This tool is the escape hatch. Prefer the more specific tools when they exist; they are less error-prone.

---

## add_rows

**Purpose:** Insert empty rows at a position in a sheet.

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `spreadsheet_id` | string | yes | Spreadsheet ID. |
| `sheet` | string | yes | Sheet name. |
| `count` | integer | yes | Number of rows to insert (must be > 0). |
| `start_row` | integer | no | 0-based row index. Default `0` (top). |

### Returns

`spreadsheets.batchUpdate` response.

### Notes

When `start_row > 0`, formatting is inherited from the row before; otherwise inserted rows are blank.

---

## add_columns

**Purpose:** Insert empty columns at a position in a sheet.

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `spreadsheet_id` | string | yes | Spreadsheet ID. |
| `sheet` | string | yes | Sheet name. |
| `count` | integer | yes | Number of columns to insert. |
| `start_column` | integer | no | 0-based column index. Default `0`. |

### Returns

`spreadsheets.batchUpdate` response.

---

## list_sheets

**Purpose:** Return the names of every sheet/tab in a spreadsheet.

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `spreadsheet_id` | string | yes | Spreadsheet ID. |

### Returns

A JSON array of strings, e.g. `["Sheet1", "Summary", "Q4"]`.

---

## create_sheet

**Purpose:** Add a new tab to an existing spreadsheet.

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `spreadsheet_id` | string | yes | Spreadsheet ID. |
| `title` | string | yes | Name for the new tab. |

### Returns

```json
{
  "sheetId": 1234567890,
  "title": "New Tab",
  "index": 2,
  "spreadsheetId": "abc"
}
```

---

## rename_sheet

**Purpose:** Rename an existing tab.

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `spreadsheet` | string | yes | Spreadsheet ID. |
| `sheet` | string | yes | Current tab name. |
| `new_name` | string | yes | New tab name. |

### Returns

`spreadsheets.batchUpdate` response.

### Notes

The parameter is named `spreadsheet` (not `spreadsheet_id`) for parity with the Python upstream.

---

## copy_sheet

**Purpose:** Copy a tab from one spreadsheet into another, optionally renaming it.

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `src_spreadsheet` | string | yes | Source spreadsheet ID. |
| `src_sheet` | string | yes | Source tab name. |
| `dst_spreadsheet` | string | yes | Destination spreadsheet ID. |
| `dst_sheet` | string | yes | Desired tab name in destination. |

### Returns

```json
{
  "copy":   { /* copyTo response */ },
  "rename": { /* optional batchUpdate response, present only when renaming */ }
}
```

### Errors

- `source sheet: sheet "X" not found` — `src_sheet` does not exist in `src_spreadsheet`.

---

## get_multiple_sheet_data

**Purpose:** Read several ranges across potentially different spreadsheets in one tool call.

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `queries` | array of object | yes | Each entry: `{spreadsheet_id, sheet, range}`. |

### Returns

A list of objects, each echoing the query plus either `data` (2-D array of values) or `error`:

```json
[
  { "spreadsheet_id": "abc", "sheet": "Sheet1", "range": "A1:B2", "data": [["a","b"],["c","d"]] },
  { "spreadsheet_id": "xyz", "sheet": "Bad",    "range": "Z9",    "error": "..." }
]
```

### Errors

Per-query errors appear in the `error` field; the overall call still succeeds.

---

## get_multiple_spreadsheet_summary

**Purpose:** For a list of spreadsheets, return the title, every tab name, the header row, and the first few rows.

**When to use:** The agent has a list of spreadsheet IDs and needs to know what is inside each one without paying the cost of `get_sheet_data` for each tab.

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `spreadsheet_ids` | array of string | yes | Spreadsheet IDs to summarize. |
| `rows_to_fetch` | integer | no | Number of rows including the header. Default `5`, minimum `1`. |

### Returns

```json
[
  {
    "spreadsheet_id": "abc",
    "title": "Sales",
    "error": null,
    "sheets": [
      {
        "title": "Q4",
        "sheet_id": 0,
        "headers": ["Region","Revenue"],
        "first_rows": [["EU","1234"],["US","2345"]]
      }
    ]
  }
]
```

---

## find_in_spreadsheet

**Purpose:** Search cell values for a substring and return each match's address.

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `spreadsheet_id` | string | yes | Spreadsheet ID. |
| `query` | string | yes | Substring to look for. |
| `sheet` | string | no | Restrict to one tab. Omit to search every tab. |
| `case_sensitive` | bool | no | Default `false`. |
| `max_results` | integer | no | Default `50`. |

### Returns

```json
[
  { "sheet": "Sheet1", "cell": "B12", "value": "needle in haystack" }
]
```

### Notes

The search reads all values in each candidate tab, then filters client-side. For very large spreadsheets prefer `search_spreadsheets` (Drive full-text search) or supply a `sheet` filter.

---

## create_spreadsheet

**Purpose:** Create a new spreadsheet in Drive.

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `title` | string | yes | Desired title. |
| `folder_id` | string | no | Drive folder to create in. Falls back to `DRIVE_FOLDER_ID`, then Drive root. |

### Returns

```json
{
  "spreadsheetId": "abc",
  "title": "Quarterly Report",
  "folder": "folder-id-or-root"
}
```

---

## list_spreadsheets

**Purpose:** List spreadsheets in a Drive folder.

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `folder_id` | string | no | Drive folder ID. Falls back to `DRIVE_FOLDER_ID`. If neither is set, lists from My Drive root. |

### Returns

```json
[
  { "id": "abc", "title": "Sales" },
  { "id": "def", "title": "Inventory" }
]
```

Sorted by `modifiedTime` descending.

### Notes

`folder_id` is escaped before being interpolated into the Drive query string. See [security.md](security.md#drive-query-language-injection).

---

## search_spreadsheets

**Purpose:** Find spreadsheets by substring match against name or content.

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `query` | string | yes | Search term. |
| `max_results` | integer | no | 1–100, default `20`. |

### Returns

```json
[
  {
    "id": "abc",
    "name": "Q4 Sales",
    "created_time": "2024-01-02T03:04:05Z",
    "modified_time": "2024-05-01T12:00:00Z",
    "owners": ["owner@example.com"],
    "web_link": "https://docs.google.com/..."
  }
]
```

### Notes

- `query` is escaped before being interpolated into the Drive query string — injection-safe.
- Underlying query: `mimeType='application/vnd.google-apps.spreadsheet' and (name contains '…' or fullText contains '…')`.

---

## list_folders

**Purpose:** List folders inside a Drive folder (or My Drive root).

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `parent_folder_id` | string | no | Drive folder ID. Omit to list at the root. |

### Returns

```json
[
  { "id": "folder-id", "name": "AI Managed Sheets", "parent": "root" }
]
```

---

## share_spreadsheet

**Purpose:** Grant `reader`, `commenter`, or `writer` permission on a spreadsheet to a list of users.

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `spreadsheet_id` | string | yes | Spreadsheet ID. |
| `recipients` | array of object | yes | Each `{email_address, role}`. |
| `send_notification` | bool | no | Default `true`. |

`role` must be one of `reader`, `commenter`, `writer`. Missing `role` defaults to `writer`.

### Returns

```json
{
  "successes": [
    { "email_address": "user@example.com", "role": "writer", "permissionId": "..." }
  ],
  "failures": [
    { "email_address": "bad@x.com", "error": "Recipient domain not in ALLOWED_SHARE_DOMAINS" }
  ]
}
```

### Notes

- If `ALLOWED_SHARE_DOMAINS` is set, recipients outside the allow-list are rejected with the error above. This is a guardrail against prompt-injection–driven data exfiltration. See [security.md](security.md#share_spreadsheet-allow-list).
- Invalid roles produce a per-recipient failure rather than a tool-level error.

---

## add_chart

**Purpose:** Insert a chart (column, bar, line, area, pie, scatter, combo, or histogram) into a sheet.

### Parameters

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `spreadsheet_id` | string | yes | Spreadsheet ID. |
| `sheet` | string | yes | Sheet containing the data. |
| `chart_type` | string | yes | `COLUMN`, `BAR`, `LINE`, `AREA`, `PIE`, `SCATTER`, `COMBO`, `HISTOGRAM`. |
| `data_range` | string | yes | A1 range, e.g. `A1:C10`. First column is the domain; remaining columns become series. First row is treated as headers. |
| `title` | string | no | Chart title. |
| `x_axis_label` | string | no | Bottom-axis label. Ignored for `PIE`. |
| `y_axis_label` | string | no | Left-axis label. Ignored for `PIE`. |
| `position_x` | integer | no | Pixel offset from anchor (default `0`). |
| `position_y` | integer | no | Pixel offset from anchor (default `0`). |
| `width` | integer | no | Chart width in pixels (default `600`). |
| `height` | integer | no | Chart height in pixels (default `400`). |

### Returns

```json
{
  "success": true,
  "message": "Chart \"Monthly Sales\" added successfully",
  "chartId": 1234,
  "result": { /* full batchUpdate response */ }
}
```

### Errors

- `Invalid chart type "..."` — `chart_type` is not one of the supported values.
- `Failed to add chart: ...` — the underlying batchUpdate failed (e.g., range outside sheet bounds).
