package cmd

import (
	"crypto/rand"
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

// nullableString returns nil for an empty string, otherwise the string itself.
// Used to build request bodies where the Management API expects null rather
// than "" for an unset optional field (e.g. property validation messages).
func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// newUUID generates a random RFC 4122 version 4 UUID string, for client-side
// generated IDs (e.g. new document type properties/containers).
func newUUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("cannot generate UUID: %v", err))
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
