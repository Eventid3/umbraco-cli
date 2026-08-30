package cmd

import (
	"fmt"

	"github.com/Eventid3/umbraco-cli/internal/output"
	"github.com/spf13/cobra"
)

// Document represents a minimal Umbraco document/content node (tree response shape).
// The tree endpoint has no top-level "name" — display name lives in variants[0].name
// (documents are variant-by-culture content, confirmed against a live instance) —
// so Name is populated from Variants after decoding rather than via a json tag.
type Document struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	HasChildren bool   `json:"hasChildren"`
	IsFolder    bool   `json:"isFolder"`
	Variants    []struct {
		Name string `json:"name"`
	} `json:"variants"`
}

// resolveName fills in Name from the first variant when the tree response has no
// top-level name field (documents/media), and is a no-op otherwise (doctypes/blueprints).
func (d *Document) resolveName() {
	if d.Name == "" && len(d.Variants) > 0 {
		d.Name = d.Variants[0].Name
	}
}

type documentListResponse struct {
	Total int        `json:"total"`
	Items []Document `json:"items"`
}

var contentCmd = &cobra.Command{
	Use:   "content",
	Short: "Manage Umbraco content (documents)",
}

var contentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List documents",
	Long: `List documents at the content root, or pass --parent to list the
children of a specific document instead (for browsing an existing site's
tree, or finding where to place new test content).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		skip, _ := cmd.Flags().GetInt("skip")
		take, _ := cmd.Flags().GetInt("take")
		parent, _ := cmd.Flags().GetString("parent")

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}

		var path string
		if parent != "" {
			path = fmt.Sprintf("/tree/document/children?parentId=%s&skip=%d&take=%d", parent, skip, take)
		} else {
			path = fmt.Sprintf("/tree/document/root?skip=%d&take=%d", skip, take)
		}

		var result documentListResponse
		if err := client.Get(path, &result); err != nil {
			return err
		}
		for i := range result.Items {
			result.Items[i].resolveName()
		}

		if output.IsJSON() {
			output.JSON(result)
			return nil
		}

		output.Line("Total: %d", result.Total)
		if len(result.Items) == 0 {
			output.Line("No documents found.")
			return nil
		}
		headers := []string{"ID", "NAME", "HAS CHILDREN", "IS FOLDER"}
		rows := make([][]string, 0, len(result.Items))
		for _, d := range result.Items {
			rows = append(rows, []string{d.ID, d.Name, boolStr(d.HasChildren), boolStr(d.IsFolder)})
		}
		output.Table(headers, rows)
		return nil
	},
}

var contentGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a document by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var doc map[string]interface{}
		if err := client.Get("/document/"+args[0], &doc); err != nil {
			return err
		}
		output.JSON(doc)
		return nil
	},
}

var contentCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a document (pass JSON body via --file or stdin with --file=-)",
	Long: `Create a document. The JSON body must contain at minimum:

  {
    "documentType": {"id": "<doctype-uuid>"},
    "template": null,
    "values": [],
    "variants": [{"culture": null, "segment": null, "name": "My Page"}]
  }`,
	RunE: func(cmd *cobra.Command, args []string) error {
		body, err := readJSONInput(cmd, "file")
		if err != nil {
			return err
		}
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		id, err := client.PostCreate("/document", body)
		if err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": id})
		} else {
			output.Line("Document created: %s", id)
		}
		return nil
	},
}

var contentUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a document (pass JSON body via --file or stdin with --file=-)",
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
		if err := client.Put("/document/"+args[0], body, nil); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": args[0]})
		} else {
			output.Line("Document %s updated.", args[0])
		}
		return nil
	},
}

var contentDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a document",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		if err := client.Delete("/document/" + args[0]); err != nil {
			return err
		}
		output.Line("Document %s deleted.", args[0])
		return nil
	},
}

var contentPublishCmd = &cobra.Command{
	Use:   "publish <id>",
	Short: "Publish a document",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		cultures, _ := cmd.Flags().GetStringSlice("cultures")
		body := map[string]interface{}{"publishSchedules": []interface{}{}}
		if len(cultures) > 0 {
			schedules := make([]map[string]string, len(cultures))
			for i, c := range cultures {
				schedules[i] = map[string]string{"culture": c}
			}
			body["publishSchedules"] = schedules
		}
		var result map[string]interface{}
		if err := client.Put("/document/"+args[0]+"/publish", body, &result); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(result)
		} else {
			output.Line("Document %s published.", args[0])
		}
		return nil
	},
}

var contentUnpublishCmd = &cobra.Command{
	Use:   "unpublish <id>",
	Short: "Unpublish a document",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		cultures, _ := cmd.Flags().GetStringSlice("cultures")
		body := map[string]interface{}{"cultures": cultures}
		var result map[string]interface{}
		if err := client.Put("/document/"+args[0]+"/unpublish", body, &result); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(result)
		} else {
			output.Line("Document %s unpublished.", args[0])
		}
		return nil
	},
}

var contentValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a document body without creating it (pass JSON via --file or stdin with --file=-)",
	Long: `Sends a document-creation payload to the Management API's validation
endpoint. The document is checked for schema/property errors but never
persisted — useful for iterating on a --file body before running
'content create', especially when hand-building test content.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		body, err := readJSONInput(cmd, "file")
		if err != nil {
			return err
		}
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		if err := client.Post("/document/validate", body, nil); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]bool{"valid": true})
		} else {
			output.Line("Document is valid.")
		}
		return nil
	},
}

func init() {
	contentListCmd.Flags().Int("skip", 0, "Number of items to skip")
	contentListCmd.Flags().Int("take", 100, "Number of items to return")
	contentListCmd.Flags().String("parent", "", "Parent document ID (list its children instead of the root)")

	contentCreateCmd.Flags().String("file", "", "Path to JSON file (use - for stdin)")
	contentUpdateCmd.Flags().String("file", "", "Path to JSON file (use - for stdin)")
	contentValidateCmd.Flags().String("file", "", "Path to JSON file (use - for stdin)")

	contentPublishCmd.Flags().StringSlice("cultures", []string{}, "Cultures to publish (empty = all)")
	contentUnpublishCmd.Flags().StringSlice("cultures", []string{}, "Cultures to unpublish (empty = all)")

	contentCmd.AddCommand(contentListCmd)
	contentCmd.AddCommand(contentGetCmd)
	contentCmd.AddCommand(contentCreateCmd)
	contentCmd.AddCommand(contentUpdateCmd)
	contentCmd.AddCommand(contentDeleteCmd)
	contentCmd.AddCommand(contentPublishCmd)
	contentCmd.AddCommand(contentUnpublishCmd)
	contentCmd.AddCommand(contentValidateCmd)
}
