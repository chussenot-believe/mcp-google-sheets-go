# Resources

## Summary

The server exposes one MCP resource template: `spreadsheet://{spreadsheet_id}/info`. Reading this URI returns JSON metadata for a spreadsheet (title, sheets, grid properties). It is suitable for cheap discovery before issuing more expensive tool calls like `get_sheet_data`.

## Resource template

| URI template | `spreadsheet://{spreadsheet_id}/info` |
| --- | --- |
| Name | Spreadsheet Info |
| MIME type | `application/json` |
| Handler source | [`resources.go`](../resources.go) |

### Template variables

| Variable | Description |
| --- | --- |
| `spreadsheet_id` | The spreadsheet ID extracted from the Google Sheets URL: `https://docs.google.com/spreadsheets/d/{spreadsheet_id}/edit`. |

## Return shape

The resource body is a JSON object:

```json
{
  "title": "Quarterly Sales Report Q4",
  "sheets": [
    {
      "title": "Sheet1",
      "sheetId": 0,
      "gridProperties": {
        "rowCount": 1000,
        "columnCount": 26
      }
    },
    {
      "title": "Summary",
      "sheetId": 1234567890
    }
  ]
}
```

| Field | Type | Description |
| --- | --- | --- |
| `title` | string | The spreadsheet's display title. `"Unknown"` if the property is missing. |
| `sheets` | array of object | One entry per sheet/tab. |
| `sheets[].title` | string | Sheet/tab name. |
| `sheets[].sheetId` | integer | Numeric sheet identifier used by the batchUpdate API. |
| `sheets[].gridProperties` | object (optional) | Row/column counts, frozen rows/columns. Present only when the Sheets API returns it. |

## Errors

| Condition | Error returned |
| --- | --- |
| Spreadsheet not found, or principal lacks read permission | `fetch spreadsheet: googleapi: Error 404: …` or `Error 403: …` |
| Template variable missing | `missing template argument: spreadsheet_id` |

## When to prefer the resource over a tool

| Goal | Use |
| --- | --- |
| Discover sheets and grid sizes before reading data | This resource. |
| Just enumerate tab names | `list_sheets` tool. |
| Read actual cell values | `get_sheet_data` tool. |
| Get headers + first rows of every tab | `get_multiple_spreadsheet_summary` tool. |

The resource is the cheapest call when the agent only needs metadata.
