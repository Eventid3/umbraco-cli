package cmd

import (
	"net/url"

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

var systemLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Search recent server log entries",
	Long: `Reads recent entries from Umbraco's log viewer — useful for diagnosing
an existing setup (startup errors, property editor exceptions, etc.) without
shelling into the server's log files.

--from/--to accept ISO 8601 datetimes (e.g. 2026-08-01T00:00:00Z).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		skip, _ := cmd.Flags().GetInt("skip")
		take, _ := cmd.Flags().GetInt("take")
		levels, _ := cmd.Flags().GetStringSlice("level")
		search, _ := cmd.Flags().GetString("search")
		order, _ := cmd.Flags().GetString("order")
		from, _ := cmd.Flags().GetString("from")
		to, _ := cmd.Flags().GetString("to")

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		q := url.Values{}
		q.Set("skip", itoa(skip))
		q.Set("take", itoa(take))
		if order != "" {
			q.Set("orderDirection", order)
		}
		if search != "" {
			q.Set("filterExpression", search)
		}
		for _, l := range levels {
			q.Add("logLevel", l)
		}
		if from != "" {
			q.Set("startDate", from)
		}
		if to != "" {
			q.Set("endDate", to)
		}

		var result map[string]interface{}
		if err := client.Get("/log-viewer/log?"+q.Encode(), &result); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(result)
			return nil
		}
		items, _ := result["items"].([]interface{})
		total, _ := result["total"].(float64)
		output.Line("Total: %d", int(total))
		if len(items) == 0 {
			output.Line("No log entries found.")
			return nil
		}
		headers := []string{"TIMESTAMP", "LEVEL", "MESSAGE"}
		rows := make([][]string, 0, len(items))
		for _, raw := range items {
			item, _ := raw.(map[string]interface{})
			ts, _ := item["timestamp"].(string)
			level, _ := item["level"].(string)
			msg, _ := item["renderedMessage"].(string)
			rows = append(rows, []string{ts, level, msg})
		}
		output.Table(headers, rows)
		return nil
	},
}

var systemLogLevelsCmd = &cobra.Command{
	Use:   "log-levels",
	Short: "Show a count of log entries per level",
	Long:  `--from/--to accept ISO 8601 datetimes (e.g. 2026-08-01T00:00:00Z).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		from, _ := cmd.Flags().GetString("from")
		to, _ := cmd.Flags().GetString("to")

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		q := url.Values{}
		if from != "" {
			q.Set("startDate", from)
		}
		if to != "" {
			q.Set("endDate", to)
		}
		path := "/log-viewer/level-count"
		if enc := q.Encode(); enc != "" {
			path += "?" + enc
		}

		var result map[string]interface{}
		if err := client.Get(path, &result); err != nil {
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

func init() {
	systemLogsCmd.Flags().Int("skip", 0, "Number of items to skip")
	systemLogsCmd.Flags().Int("take", 100, "Number of items to return")
	systemLogsCmd.Flags().StringSlice("level", []string{}, "Filter by level: Verbose, Debug, Information, Warning, Error, Fatal (repeatable)")
	systemLogsCmd.Flags().String("search", "", "Filter expression to match against log messages")
	systemLogsCmd.Flags().String("order", "Descending", "Order by timestamp: Ascending or Descending")
	systemLogsCmd.Flags().String("from", "", "Only show entries at or after this ISO 8601 datetime")
	systemLogsCmd.Flags().String("to", "", "Only show entries at or before this ISO 8601 datetime")

	systemLogLevelsCmd.Flags().String("from", "", "Only count entries at or after this ISO 8601 datetime")
	systemLogLevelsCmd.Flags().String("to", "", "Only count entries at or before this ISO 8601 datetime")

	systemCmd.AddCommand(systemInfoCmd)
	systemCmd.AddCommand(systemHealthCmd)
	systemCmd.AddCommand(systemCacheRebuildCmd)
	systemCmd.AddCommand(systemLogsCmd)
	systemCmd.AddCommand(systemLogLevelsCmd)
}

// printKeyValue prints a flat map as key: value lines.
func printKeyValue(m map[string]interface{}) {
	for k, v := range m {
		output.Line("%-30s %v", k, v)
	}
}
