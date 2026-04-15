package output

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
)

var jsonMode bool

// SetJSON enables JSON output mode.
func SetJSON(enabled bool) {
	jsonMode = enabled
}

// IsJSON returns true if JSON mode is active.
func IsJSON() bool {
	return jsonMode
}

// JSON prints v as indented JSON to stdout.
func JSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// Error prints an error, respecting JSON mode.
func Error(err error) {
	if jsonMode {
		JSON(map[string]string{"error": err.Error()})
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
	}
}

// Table writes tab-separated rows to stdout using tabwriter for alignment.
// headers is a slice of column names; rows is a slice of string slices.
func Table(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	// Print header
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, h)
	}
	fmt.Fprintln(w)
	// Separator
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		for range h {
			fmt.Fprint(w, "-")
		}
	}
	fmt.Fprintln(w)
	// Rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, cell)
		}
		fmt.Fprintln(w)
	}
	w.Flush()
}

// Line prints a single line to stdout.
func Line(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}
