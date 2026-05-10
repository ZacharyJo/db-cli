package output

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
)

// Format constants.
const (
	FormatTable = "table"
	FormatJSON  = "json"
	FormatCSV   = "csv"
)

// Printer renders query results to stdout in the chosen format.
type Printer struct {
	Format string
	Out    io.Writer
}

// New returns a Printer writing to stdout with the given format.
func New(format string) *Printer {
	if format == "" {
		format = FormatTable
	}
	return &Printer{Format: format, Out: os.Stdout}
}

// PrintRows renders all rows from a *sql.Rows result set.
// It also prints row count and elapsed time.
func (p *Printer) PrintRows(rows *sql.Rows, elapsed time.Duration) error {
	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	// Collect all rows into a [][]string for table/CSV/JSON rendering.
	var data [][]string
	vals := make([]interface{}, len(cols))
	ptrs := make([]interface{}, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		row := make([]string, len(cols))
		for i, v := range vals {
			row[i] = formatValue(v)
		}
		data = append(data, row)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	switch p.Format {
	case FormatJSON:
		err = p.printJSON(cols, data)
	case FormatCSV:
		err = p.printCSV(cols, data)
	default:
		err = p.printTable(cols, data)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(p.Out, "%d row(s) in %s\n\n", len(data), elapsed.Round(time.Millisecond))
	return nil
}

// PrintResult prints the outcome of a non-SELECT statement.
func (p *Printer) PrintResult(res sql.Result, elapsed time.Duration) {
	rows, _ := res.RowsAffected()
	fmt.Fprintf(p.Out, "Query OK, %d row(s) affected (%s)\n\n", rows, elapsed.Round(time.Millisecond))
}

// PrintError prints an error in a consistent format.
func (p *Printer) PrintError(err error) {
	fmt.Fprintf(p.Out, "ERROR: %v\n", err)
}

func (p *Printer) printTable(cols []string, data [][]string) error {
	tw := tablewriter.NewWriter(p.Out)
	tw.SetHeader(cols)
	tw.SetAutoFormatHeaders(false)
	tw.SetAutoWrapText(false)
	tw.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	tw.SetCenterSeparator("|")
	for _, row := range data {
		tw.Append(row)
	}
	tw.Render()
	return nil
}

func (p *Printer) printCSV(cols []string, data [][]string) error {
	w := csv.NewWriter(p.Out)
	if err := w.Write(cols); err != nil {
		return err
	}
	for _, row := range data {
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func (p *Printer) printJSON(cols []string, data [][]string) error {
	records := make([]map[string]string, 0, len(data))
	for _, row := range data {
		rec := make(map[string]string, len(cols))
		for i, col := range cols {
			rec[col] = row[i]
		}
		records = append(records, rec)
	}
	enc := json.NewEncoder(p.Out)
	enc.SetIndent("", "  ")
	return enc.Encode(records)
}

func formatValue(v interface{}) string {
	if v == nil {
		return "NULL"
	}
	var s string
	switch t := v.(type) {
	case []byte:
		s = string(t)
	case time.Time:
		return t.Format("2006-01-02 15:04:05")
	default:
		s = fmt.Sprintf("%v", t)
	}
	return strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(s)
}
