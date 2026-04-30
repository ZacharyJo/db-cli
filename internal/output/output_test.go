package output_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ZacharyJo/db-cli/internal/output"
)

// fakeRows implements a minimal sql.Rows-like interface via a custom type.
// Since output.PrintRows accepts *sql.Rows directly, we test via the
// exported Printer.PrintRows indirectly by checking the formatted output
// using a bytes.Buffer as Out.

// For lightweight unit testing without a real DB, we test the internal
// formatting helpers by calling Printer.PrintError and Printer.PrintResult
// (which use sql.Result), and verify format-switching via New().

func newPrinter(format string, buf *bytes.Buffer) *output.Printer {
	return &output.Printer{Format: format, Out: buf}
}

func TestNew_DefaultFormat(t *testing.T) {
	p := output.New("")
	if p.Format != output.FormatTable {
		t.Errorf("expected default format %q, got %q", output.FormatTable, p.Format)
	}
}

func TestNew_JSONFormat(t *testing.T) {
	p := output.New(output.FormatJSON)
	if p.Format != output.FormatJSON {
		t.Errorf("expected %q, got %q", output.FormatJSON, p.Format)
	}
}

func TestNew_CSVFormat(t *testing.T) {
	p := output.New(output.FormatCSV)
	if p.Format != output.FormatCSV {
		t.Errorf("expected %q, got %q", output.FormatCSV, p.Format)
	}
}

func TestPrintError(t *testing.T) {
	var buf bytes.Buffer
	p := newPrinter(output.FormatTable, &buf)
	p.PrintError(errors.New("connection refused"))
	if !strings.Contains(buf.String(), "ERROR") {
		t.Errorf("expected ERROR prefix, got: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "connection refused") {
		t.Errorf("expected error message in output, got: %q", buf.String())
	}
}

func TestPrintResult(t *testing.T) {
	var buf bytes.Buffer
	p := newPrinter(output.FormatTable, &buf)
	p.PrintResult(fakeResult{rowsAffected: 3}, 50*time.Millisecond)
	out := buf.String()
	if !strings.Contains(out, "3 row(s)") {
		t.Errorf("expected row count in output, got: %q", out)
	}
	if !strings.Contains(out, "Query OK") {
		t.Errorf("expected 'Query OK' in output, got: %q", out)
	}
}

// fakeResult implements sql.Result for testing.
type fakeResult struct {
	rowsAffected int64
}

func (f fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (f fakeResult) RowsAffected() (int64, error) { return f.rowsAffected, nil }
