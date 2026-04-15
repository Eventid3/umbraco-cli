package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// readJSONInput reads a JSON payload from a --file flag.
// Use --file=- to read from stdin.
func readJSONInput(cmd *cobra.Command, flagName string) (interface{}, error) {
	path, err := cmd.Flags().GetString(flagName)
	if err != nil || path == "" {
		return nil, fmt.Errorf("flag --%s is required", flagName)
	}

	var r io.Reader
	if path == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("cannot open file %q: %w", path, err)
		}
		defer f.Close()
		r = f
	}

	var body interface{}
	if err := json.NewDecoder(r).Decode(&body); err != nil {
		return nil, fmt.Errorf("cannot parse JSON: %w", err)
	}
	return body, nil
}

// boolStr converts a bool to "yes" or "no" for table output.
func boolStr(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
