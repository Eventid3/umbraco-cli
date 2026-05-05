package cmd

import (
	"fmt"

	"github.com/Eventid3/umbraco-cli/internal/output"
	"github.com/spf13/cobra"
)

// Document represents a minimal Umbraco document/content node (tree response shape).
type Document struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	HasChildren bool   `json:"hasChildren"`
	IsFolder    bool   `json:"isFolder"`
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
	RunE: func(cmd *cobra.Command, args []string) error {
		skip, _ := cmd.Flags().GetInt("skip")
		take, _ := cmd.Flags().GetInt("take")

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}

		var result documentListResponse
		path := fmt.Sprintf("/tree/document/root?skip=%d&take=%d", skip, take)
		if err := client.Get(path, &result); err != nil {
			return err
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

func init() {
	contentListCmd.Flags().Int("skip", 0, "Number of items to skip")
	contentListCmd.Flags().Int("take", 100, "Number of items to return")

	contentCreateCmd.Flags().String("file", "", "Path to JSON file (use - for stdin)")
	contentUpdateCmd.Flags().String("file", "", "Path to JSON file (use - for stdin)")

	contentPublishCmd.Flags().StringSlice("cultures", []string{}, "Cultures to publish (empty = all)")
	contentUnpublishCmd.Flags().StringSlice("cultures", []string{}, "Cultures to unpublish (empty = all)")

	contentCmd.AddCommand(contentListCmd)
	contentCmd.AddCommand(contentGetCmd)
	contentCmd.AddCommand(contentCreateCmd)
	contentCmd.AddCommand(contentUpdateCmd)
	contentCmd.AddCommand(contentDeleteCmd)
	contentCmd.AddCommand(contentPublishCmd)
	contentCmd.AddCommand(contentUnpublishCmd)
}
