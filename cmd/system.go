package cmd

import (
	"github.com/Eventid3/umbraco-cli/internal/output"
	"github.com/spf13/cobra"
)

var systemCmd = &cobra.Command{
	Use:   "system",
	Short: "Umbraco server information and operations",
}

var systemInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Print server and version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result map[string]interface{}
		if err := client.Get("/server/information", &result); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(result)
			return nil
		}
		printKeyValue(result)
		return nil
	},
}

var systemHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Run all health checks",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		// List all health check groups first.
		var groups map[string]interface{}
		if err := client.Get("/health-check-group", &groups); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(groups)
			return nil
		}
		items, _ := groups["items"].([]interface{})
		if len(items) == 0 {
			output.Line("No health check groups found.")
			return nil
		}
		headers := []string{"GROUP"}
		rows := make([][]string, 0, len(items))
		for _, raw := range items {
			item, _ := raw.(map[string]interface{})
			name, _ := item["name"].(string)
			rows = append(rows, []string{name})
		}
		output.Table(headers, rows)
		return nil
	},
}

var systemCacheRebuildCmd = &cobra.Command{
	Use:   "cache-rebuild",
	Short: "Rebuild the published content cache",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		if err := client.Post("/published-cache/rebuild", nil, nil); err != nil {
			return err
		}
		output.Line("Published cache rebuild triggered.")
		return nil
	},
}

func init() {
	systemCmd.AddCommand(systemInfoCmd)
	systemCmd.AddCommand(systemHealthCmd)
	systemCmd.AddCommand(systemCacheRebuildCmd)
}

// printKeyValue prints a flat map as key: value lines.
func printKeyValue(m map[string]interface{}) {
	for k, v := range m {
		output.Line("%-30s %v", k, v)
	}
}
