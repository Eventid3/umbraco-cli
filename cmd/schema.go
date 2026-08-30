package cmd

import (
	"net/url"
	"strings"

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

var schemaDocTypeSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search document types by name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skip, _ := cmd.Flags().GetInt("skip")
		take, _ := cmd.Flags().GetInt("take")

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result map[string]interface{}
		path := "/item/document-type/search?query=" + url.QueryEscape(args[0]) + "&skip=" + itoa(skip) + "&take=" + itoa(take)
		if err := client.Get(path, &result); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(result)
			return nil
		}
		printGenericList(result)
		return nil
	},
}

var schemaDocTypeAllowedChildrenCmd = &cobra.Command{
	Use:   "allowed-children <id>",
	Short: "List document types allowed as children of a document type",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skip, _ := cmd.Flags().GetInt("skip")
		take, _ := cmd.Flags().GetInt("take")

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result map[string]interface{}
		path := "/document-type/" + args[0] + "/allowed-children?skip=" + itoa(skip) + "&take=" + itoa(take)
		if err := client.Get(path, &result); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(result)
			return nil
		}
		printGenericList(result)
		return nil
	},
}

var schemaDocTypeAllowedParentsCmd = &cobra.Command{
	Use:   "allowed-parents <id>",
	Short: "List document type IDs that may be a parent of a document type",
	Long: `Counterpart of allowed-children — useful when validating where a new
test content node of this type is allowed to be placed in an existing tree.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result interface{}
		if err := client.Get("/document-type/"+args[0]+"/allowed-parents", &result); err != nil {
			return err
		}
		output.JSON(result)
		return nil
	},
}

var schemaDocTypeAllowedAtRootCmd = &cobra.Command{
	Use:   "allowed-at-root",
	Short: "List document types allowed at the content root",
	RunE: func(cmd *cobra.Command, args []string) error {
		skip, _ := cmd.Flags().GetInt("skip")
		take, _ := cmd.Flags().GetInt("take")

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result map[string]interface{}
		path := "/document-type/allowed-at-root?skip=" + itoa(skip) + "&take=" + itoa(take)
		if err := client.Get(path, &result); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(result)
			return nil
		}
		printGenericList(result)
		return nil
	},
}

var schemaDocTypeCompositionRefsCmd = &cobra.Command{
	Use:   "composition-refs <id>",
	Short: "List document types that use this document type as a composition",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result interface{}
		if err := client.Get("/document-type/"+args[0]+"/composition-references", &result); err != nil {
			return err
		}
		output.JSON(result)
		return nil
	},
}

var schemaDocTypeAvailableCompositionsCmd = &cobra.Command{
	Use:   "available-compositions <id>",
	Short: "List document types that can be composed into a document type",
	Long: `Reads the target document type's current properties and compositions,
then asks the Management API which other document types are compatible
compositions — i.e. which ones would not introduce clashing property
aliases or composition cycles.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		doc, err := fetchDocTypeMap(client, args[0])
		if err != nil {
			return err
		}

		isElement, _ := doc["isElement"].(bool)
		properties, _ := doc["properties"].([]interface{})
		aliases := make([]string, 0, len(properties))
		for _, raw := range properties {
			if p, ok := raw.(map[string]interface{}); ok {
				if alias, ok := p["alias"].(string); ok {
					aliases = append(aliases, alias)
				}
			}
		}
		compositions, _ := doc["compositions"].([]interface{})
		compositeIDs := make([]string, 0, len(compositions))
		for _, raw := range compositions {
			if c, ok := raw.(map[string]interface{}); ok {
				if dt, ok := c["documentType"].(map[string]interface{}); ok {
					if id, ok := dt["id"].(string); ok {
						compositeIDs = append(compositeIDs, id)
					}
				}
			}
		}

		body := map[string]interface{}{
			"id":                     args[0],
			"isElement":              isElement,
			"currentPropertyAliases": aliases,
			"currentCompositeIds":    compositeIDs,
		}
		var result interface{}
		if err := client.Post("/document-type/available-compositions", body, &result); err != nil {
			return err
		}
		output.JSON(result)
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
	Long: `List data types. Pass --editor-alias to filter by property editor
(e.g. "Umbraco.TextBox") — useful when checking whether an existing site
already has a data type wrapping a given editor before creating a new one.

The filter is applied client-side to the fetched page, so pair it with a
larger --take if the match isn't found on the default page size.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		skip, _ := cmd.Flags().GetInt("skip")
		take, _ := cmd.Flags().GetInt("take")
		editorAlias, _ := cmd.Flags().GetString("editor-alias")

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result map[string]interface{}
		if err := client.Get("/filter/data-type?skip="+itoa(skip)+"&take="+itoa(take), &result); err != nil {
			return err
		}

		items, _ := result["items"].([]interface{})
		total, _ := result["total"].(float64)
		filtered := editorAlias != ""
		if filtered {
			matched := make([]interface{}, 0, len(items))
			for _, raw := range items {
				if item, ok := raw.(map[string]interface{}); ok {
					if alias, _ := item["editorAlias"].(string); strings.EqualFold(alias, editorAlias) {
						matched = append(matched, item)
					}
				}
			}
			items = matched
			result["items"] = items
		}

		if output.IsJSON() {
			output.JSON(result)
			return nil
		}
		if filtered {
			output.Line("Total: %d (filtered from %d on this page)", len(items), int(total))
		} else {
			output.Line("Total: %d", int(total))
		}
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
			alias, _ := item["editorAlias"].(string)
			rows = append(rows, []string{id, name, alias})
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
	schemaDataTypeListCmd.Flags().String("editor-alias", "", "Filter results by editor alias (e.g. Umbraco.TextBox)")

	schemaDocTypeCreateCmd.Flags().String("file", "", "Path to JSON file (use - for stdin)")
	schemaDocTypeUpdateCmd.Flags().String("file", "", "Path to JSON file (use - for stdin)")
	schemaDataTypeCreateCmd.Flags().String("file", "", "Path to JSON file (use - for stdin)")
	schemaDataTypeUpdateCmd.Flags().String("file", "", "Path to JSON file (use - for stdin)")

	schemaDocTypeSearchCmd.Flags().Int("skip", 0, "Number of items to skip")
	schemaDocTypeSearchCmd.Flags().Int("take", 100, "Number of items to return")
	schemaDocTypeAllowedChildrenCmd.Flags().Int("skip", 0, "Number of items to skip")
	schemaDocTypeAllowedChildrenCmd.Flags().Int("take", 100, "Number of items to return")
	schemaDocTypeAllowedAtRootCmd.Flags().Int("skip", 0, "Number of items to skip")
	schemaDocTypeAllowedAtRootCmd.Flags().Int("take", 100, "Number of items to return")

	schemaDocTypeCmd.AddCommand(schemaDocTypeListCmd)
	schemaDocTypeCmd.AddCommand(schemaDocTypeGetCmd)
	schemaDocTypeCmd.AddCommand(schemaDocTypeCreateCmd)
	schemaDocTypeCmd.AddCommand(schemaDocTypeUpdateCmd)
	schemaDocTypeCmd.AddCommand(schemaDocTypeDeleteCmd)
	schemaDocTypeCmd.AddCommand(schemaDocTypeSearchCmd)
	schemaDocTypeCmd.AddCommand(schemaDocTypeAllowedChildrenCmd)
	schemaDocTypeCmd.AddCommand(schemaDocTypeAllowedParentsCmd)
	schemaDocTypeCmd.AddCommand(schemaDocTypeAllowedAtRootCmd)
	schemaDocTypeCmd.AddCommand(schemaDocTypeCompositionRefsCmd)
	schemaDocTypeCmd.AddCommand(schemaDocTypeAvailableCompositionsCmd)

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
