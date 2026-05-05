package cmd

import (
	"github.com/Eventid3/umbraco-cli/internal/output"
	"github.com/spf13/cobra"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Inspect Umbraco schema (document types, data types)",
}

// -- document-type subcommands --

var schemaDocTypeCmd = &cobra.Command{
	Use:   "doctype",
	Short: "Manage document types",
}

var schemaDocTypeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List document types",
	RunE: func(cmd *cobra.Command, args []string) error {
		skip, _ := cmd.Flags().GetInt("skip")
		take, _ := cmd.Flags().GetInt("take")

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result map[string]interface{}
		if err := client.Get("/tree/document-type/root?skip="+itoa(skip)+"&take="+itoa(take), &result); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(result)
			return nil
		}
		// Generic table: id + name from items array
		printGenericList(result)
		return nil
	},
}

var schemaDocTypeGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a document type by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result map[string]interface{}
		if err := client.Get("/document-type/"+args[0], &result); err != nil {
			return err
		}
		output.JSON(result)
		return nil
	},
}

var schemaDocTypeCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a document type from a JSON file",
	RunE: func(cmd *cobra.Command, args []string) error {
		body, err := readJSONInput(cmd, "file")
		if err != nil {
			return err
		}
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		id, err := client.PostCreate("/document-type", body)
		if err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": id})
		} else {
			output.Line("Document type created: %s", id)
		}
		return nil
	},
}

var schemaDocTypeUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a document type from a JSON file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body, err := readJSONInput(cmd, "file")
		if err != nil {
			return err
		}
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		if err := client.Put("/document-type/"+args[0], body, nil); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": args[0], "status": "updated"})
		} else {
			output.Line("Document type %s updated.", args[0])
		}
		return nil
	},
}

var schemaDocTypeDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a document type",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		if err := client.Delete("/document-type/" + args[0]); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": args[0], "status": "deleted"})
		} else {
			output.Line("Document type %s deleted.", args[0])
		}
		return nil
	},
}

// -- data-type subcommands --

var schemaDataTypeCmd = &cobra.Command{
	Use:   "datatype",
	Short: "Manage data types",
}

var schemaDataTypeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List data types",
	RunE: func(cmd *cobra.Command, args []string) error {
		skip, _ := cmd.Flags().GetInt("skip")
		take, _ := cmd.Flags().GetInt("take")

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result map[string]interface{}
		if err := client.Get("/filter/data-type?skip="+itoa(skip)+"&take="+itoa(take), &result); err != nil {
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
			output.Line("No items found.")
			return nil
		}
		headers := []string{"ID", "NAME", "EDITOR ALIAS"}
		rows := make([][]string, 0, len(items))
		for _, raw := range items {
			item, _ := raw.(map[string]interface{})
			id, _ := item["id"].(string)
			name, _ := item["name"].(string)
			editorAlias, _ := item["editorAlias"].(string)
			rows = append(rows, []string{id, name, editorAlias})
		}
		output.Table(headers, rows)
		return nil
	},
}

var schemaDataTypeGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a data type by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result map[string]interface{}
		if err := client.Get("/data-type/"+args[0], &result); err != nil {
			return err
		}
		output.JSON(result)
		return nil
	},
}

var schemaDataTypeCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a data type from a JSON file",
	RunE: func(cmd *cobra.Command, args []string) error {
		body, err := readJSONInput(cmd, "file")
		if err != nil {
			return err
		}
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		id, err := client.PostCreate("/data-type", body)
		if err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": id})
		} else {
			output.Line("Data type created: %s", id)
		}
		return nil
	},
}

var schemaDataTypeUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a data type from a JSON file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body, err := readJSONInput(cmd, "file")
		if err != nil {
			return err
		}
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		if err := client.Put("/data-type/"+args[0], body, nil); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": args[0], "status": "updated"})
		} else {
			output.Line("Data type %s updated.", args[0])
		}
		return nil
	},
}

var schemaDataTypeDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a data type",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		if err := client.Delete("/data-type/" + args[0]); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": args[0], "status": "deleted"})
		} else {
			output.Line("Data type %s deleted.", args[0])
		}
		return nil
	},
}

func init() {
	schemaDocTypeListCmd.Flags().Int("skip", 0, "Number of items to skip")
	schemaDocTypeListCmd.Flags().Int("take", 100, "Number of items to return")
	schemaDataTypeListCmd.Flags().Int("skip", 0, "Number of items to skip")
	schemaDataTypeListCmd.Flags().Int("take", 100, "Number of items to return")

	schemaDocTypeCreateCmd.Flags().String("file", "", "Path to JSON file (use - for stdin)")
	schemaDocTypeUpdateCmd.Flags().String("file", "", "Path to JSON file (use - for stdin)")
	schemaDataTypeCreateCmd.Flags().String("file", "", "Path to JSON file (use - for stdin)")
	schemaDataTypeUpdateCmd.Flags().String("file", "", "Path to JSON file (use - for stdin)")

	schemaDocTypeCmd.AddCommand(schemaDocTypeListCmd)
	schemaDocTypeCmd.AddCommand(schemaDocTypeGetCmd)
	schemaDocTypeCmd.AddCommand(schemaDocTypeCreateCmd)
	schemaDocTypeCmd.AddCommand(schemaDocTypeUpdateCmd)
	schemaDocTypeCmd.AddCommand(schemaDocTypeDeleteCmd)

	schemaDataTypeCmd.AddCommand(schemaDataTypeListCmd)
	schemaDataTypeCmd.AddCommand(schemaDataTypeGetCmd)
	schemaDataTypeCmd.AddCommand(schemaDataTypeCreateCmd)
	schemaDataTypeCmd.AddCommand(schemaDataTypeUpdateCmd)
	schemaDataTypeCmd.AddCommand(schemaDataTypeDeleteCmd)

	schemaCmd.AddCommand(schemaDocTypeCmd)
	schemaCmd.AddCommand(schemaDataTypeCmd)
}

// printGenericList prints a basic id/name table from a generic API list response.
func printGenericList(result map[string]interface{}) {
	items, _ := result["items"].([]interface{})
	total, _ := result["total"].(float64)
	output.Line("Total: %d", int(total))
	if len(items) == 0 {
		output.Line("No items found.")
		return
	}
	headers := []string{"ID", "NAME"}
	rows := make([][]string, 0, len(items))
	for _, raw := range items {
		item, _ := raw.(map[string]interface{})
		id, _ := item["id"].(string)
		name, _ := item["name"].(string)
		rows = append(rows, []string{id, name})
	}
	output.Table(headers, rows)
}
