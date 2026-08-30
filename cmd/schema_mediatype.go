package cmd

import (
	"github.com/Eventid3/umbraco-cli/internal/output"
	"github.com/spf13/cobra"
)

var schemaMediaTypeCmd = &cobra.Command{
	Use:   "mediatype",
	Short: "Manage media types",
}

var schemaMediaTypeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List media types",
	RunE: func(cmd *cobra.Command, args []string) error {
		skip, _ := cmd.Flags().GetInt("skip")
		take, _ := cmd.Flags().GetInt("take")

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result map[string]interface{}
		if err := client.Get("/tree/media-type/root?skip="+itoa(skip)+"&take="+itoa(take), &result); err != nil {
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

var schemaMediaTypeGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a media type by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result map[string]interface{}
		if err := client.Get("/media-type/"+args[0], &result); err != nil {
			return err
		}
		output.JSON(result)
		return nil
	},
}

var schemaMediaTypeCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a media type from a JSON file",
	Long: `Create a media type. The body shape mirrors a document type, except it
has no allowedTemplates/defaultTemplate and uses "allowedMediaTypes" in place
of "allowedDocumentTypes":

  {
    "name": "My Media Type",
    "alias": "myMediaType",
    "icon": "icon-document",
    "allowedAsRoot": true,
    "variesByCulture": false,
    "variesBySegment": false,
    "isElement": false,
    "allowedInLibrary": true,
    "properties": [],
    "containers": [],
    "compositions": [],
    "allowedMediaTypes": []
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
		id, err := client.PostCreate("/media-type", body)
		if err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": id})
		} else {
			output.Line("Media type created: %s", id)
		}
		return nil
	},
}

var schemaMediaTypeUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a media type from a JSON file",
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
		if err := client.Put("/media-type/"+args[0], body, nil); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": args[0], "status": "updated"})
		} else {
			output.Line("Media type %s updated.", args[0])
		}
		return nil
	},
}

var schemaMediaTypeDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a media type",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		if err := client.Delete("/media-type/" + args[0]); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": args[0], "status": "deleted"})
		} else {
			output.Line("Media type %s deleted.", args[0])
		}
		return nil
	},
}

func init() {
	schemaMediaTypeListCmd.Flags().Int("skip", 0, "Number of items to skip")
	schemaMediaTypeListCmd.Flags().Int("take", 100, "Number of items to return")

	schemaMediaTypeCreateCmd.Flags().String("file", "", "Path to JSON file (use - for stdin)")
	schemaMediaTypeUpdateCmd.Flags().String("file", "", "Path to JSON file (use - for stdin)")

	schemaMediaTypeCmd.AddCommand(schemaMediaTypeListCmd)
	schemaMediaTypeCmd.AddCommand(schemaMediaTypeGetCmd)
	schemaMediaTypeCmd.AddCommand(schemaMediaTypeCreateCmd)
	schemaMediaTypeCmd.AddCommand(schemaMediaTypeUpdateCmd)
	schemaMediaTypeCmd.AddCommand(schemaMediaTypeDeleteCmd)

	schemaCmd.AddCommand(schemaMediaTypeCmd)
}
