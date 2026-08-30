package cmd

import (
	"net/url"

	"github.com/Eventid3/umbraco-cli/internal/output"
	"github.com/spf13/cobra"
)

// Element types are just document types with isElement:true — there is no
// separate Management API resource for them. These commands are thin,
// self-documenting wrappers around the /document-type endpoints so element
// types (the content models behind Block List / Block Grid blocks) have
// their own clear, discoverable command group instead of being buried in
// 'schema doctype' with a flag to remember.

var schemaElementCmd = &cobra.Command{
	Use:   "element",
	Short: "Manage element types (document types used as Block List/Grid content models)",
}

var schemaElementListCmd = &cobra.Command{
	Use:   "list",
	Short: "List element types",
	Long: `Fetches the document type tree and filters to isElement:true entries
client-side, so the filter only applies within the fetched page — pair with
a larger --take if an expected element type isn't showing up, or use
'schema element search' instead for an exact server-side match.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		skip, _ := cmd.Flags().GetInt("skip")
		take, _ := cmd.Flags().GetInt("take")

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result map[string]interface{}
		path := "/tree/document-type/root?skip=" + itoa(skip) + "&take=" + itoa(take)
		if err := client.Get(path, &result); err != nil {
			return err
		}

		items, _ := result["items"].([]interface{})
		matched := make([]interface{}, 0, len(items))
		for _, raw := range items {
			if item, ok := raw.(map[string]interface{}); ok {
				if isElement, _ := item["isElement"].(bool); isElement {
					matched = append(matched, item)
				}
			}
		}
		result["items"] = matched

		if output.IsJSON() {
			output.JSON(result)
			return nil
		}
		printGenericList(result)
		return nil
	},
}

var schemaElementSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search element types by name (exact server-side isElement filter)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skip, _ := cmd.Flags().GetInt("skip")
		take, _ := cmd.Flags().GetInt("take")

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		q := url.Values{}
		q.Set("query", args[0])
		q.Set("isElement", "true")
		q.Set("skip", itoa(skip))
		q.Set("take", itoa(take))

		var result map[string]interface{}
		if err := client.Get("/item/document-type/search?"+q.Encode(), &result); err != nil {
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

var schemaElementGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get an element type by ID",
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

var schemaElementCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an element type from a JSON file (pass via --file or stdin with --file=-)",
	Long: `Same body shape as 'schema doctype create'. If the body doesn't already
set "isElement" or "allowedAsRoot", they default to true/false respectively —
the two settings that make a document type usable as a Block List/Grid
content model instead of a page type. Build up properties and tabs/groups
afterwards with 'schema doctype property add' and 'schema doctype container add'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		raw, err := readJSONInput(cmd, "file")
		if err != nil {
			return err
		}
		body, ok := raw.(map[string]interface{})
		if ok {
			if _, set := body["isElement"]; !set {
				body["isElement"] = true
			}
			if _, set := body["allowedAsRoot"]; !set {
				body["allowedAsRoot"] = false
			}
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
			output.Line("Element type created: %s", id)
		}
		return nil
	},
}

var schemaElementUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an element type from a JSON file (alias for 'schema doctype update')",
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
			output.Line("Element type %s updated.", args[0])
		}
		return nil
	},
}

var schemaElementDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an element type (alias for 'schema doctype delete')",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		if err := client.Delete("/document-type/" + args[0]); err != nil {
			return err
		}
		output.Line("Element type %s deleted.", args[0])
		return nil
	},
}

func init() {
	schemaElementListCmd.Flags().Int("skip", 0, "Number of items to skip")
	schemaElementListCmd.Flags().Int("take", 100, "Number of items to return")
	schemaElementSearchCmd.Flags().Int("skip", 0, "Number of items to skip")
	schemaElementSearchCmd.Flags().Int("take", 100, "Number of items to return")
	schemaElementCreateCmd.Flags().String("file", "", "Path to JSON file (use - for stdin)")
	schemaElementUpdateCmd.Flags().String("file", "", "Path to JSON file (use - for stdin)")

	schemaElementCmd.AddCommand(schemaElementListCmd)
	schemaElementCmd.AddCommand(schemaElementSearchCmd)
	schemaElementCmd.AddCommand(schemaElementGetCmd)
	schemaElementCmd.AddCommand(schemaElementCreateCmd)
	schemaElementCmd.AddCommand(schemaElementUpdateCmd)
	schemaElementCmd.AddCommand(schemaElementDeleteCmd)

	schemaCmd.AddCommand(schemaElementCmd)
}
