package cmd

import (
	"fmt"

	"github.com/Eventid3/umbraco-cli/internal/output"
	"github.com/spf13/cobra"
)

var contentBlueprintCmd = &cobra.Command{
	Use:   "blueprint",
	Short: "Manage content blueprints (reusable starting points for new documents)",
	Long: `A blueprint is a saved content starting point. The fastest way to spin up
repeatable test content is: pick an existing document, turn it into a
blueprint with 'create-from', then use 'scaffold' to get a pre-filled
'content create' body from it for every new test node.`,
}

var contentBlueprintListCmd = &cobra.Command{
	Use:   "list",
	Short: "List content blueprints",
	Long:  `List root-level blueprints, or pass --parent to list a blueprint folder's contents.`,
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
			path = "/tree/document-blueprint/children?parentId=" + parent + "&skip=" + itoa(skip) + "&take=" + itoa(take)
		} else {
			path = "/tree/document-blueprint/root?skip=" + itoa(skip) + "&take=" + itoa(take)
		}
		var result map[string]interface{}
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

var contentBlueprintGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a content blueprint by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result map[string]interface{}
		if err := client.Get("/document-blueprint/"+args[0], &result); err != nil {
			return err
		}
		output.JSON(result)
		return nil
	},
}

var contentBlueprintScaffoldCmd = &cobra.Command{
	Use:   "scaffold <id>",
	Short: "Get a pre-filled 'content create' body from a blueprint",
	Long: `Returns a document body pre-populated with the blueprint's values. Edit the
name/values as needed and pipe straight into 'content create --file -' to
spin up new test content fast.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result interface{}
		if err := client.Get("/document-blueprint/"+args[0]+"/scaffold", &result); err != nil {
			return err
		}
		output.JSON(result)
		return nil
	},
}

var contentBlueprintCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a blueprint directly from a JSON file (pass via --file or stdin with --file=-)",
	Long:  `Body shape mirrors 'content create': {"documentType":{"id":...},"parent":null,"values":[...],"variants":[...]}.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		body, err := readJSONInput(cmd, "file")
		if err != nil {
			return err
		}
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		id, err := client.PostCreate("/document-blueprint", body)
		if err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": id})
		} else {
			output.Line("Blueprint created: %s", id)
		}
		return nil
	},
}

var contentBlueprintCreateFromCmd = &cobra.Command{
	Use:   "create-from <content-id>",
	Short: "Create a blueprint from an existing document",
	Long:  `Turns a real (often hand-picked "golden") document into a reusable blueprint.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		parent, _ := cmd.Flags().GetString("parent")
		if name == "" {
			return fmt.Errorf("--name is required")
		}

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		id := newUUID()
		body := map[string]interface{}{
			"id":       id,
			"name":     name,
			"document": map[string]interface{}{"id": args[0]},
			"parent":   nullableTargetRef(parent),
		}
		if err := client.Post("/document-blueprint/from-document", body, nil); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": id})
		} else {
			output.Line("Blueprint %q created from document %s (id: %s).", name, args[0], id)
		}
		return nil
	},
}

var contentBlueprintDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a content blueprint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		if err := client.Delete("/document-blueprint/" + args[0]); err != nil {
			return err
		}
		output.Line("Blueprint %s deleted.", args[0])
		return nil
	},
}

func init() {
	contentBlueprintListCmd.Flags().Int("skip", 0, "Number of items to skip")
	contentBlueprintListCmd.Flags().Int("take", 100, "Number of items to return")
	contentBlueprintListCmd.Flags().String("parent", "", "Parent blueprint folder ID")

	contentBlueprintCreateCmd.Flags().String("file", "", "Path to JSON file (use - for stdin)")

	contentBlueprintCreateFromCmd.Flags().String("name", "", "Name for the new blueprint (required)")
	contentBlueprintCreateFromCmd.Flags().String("parent", "", "Parent blueprint folder ID")

	contentBlueprintCmd.AddCommand(contentBlueprintListCmd)
	contentBlueprintCmd.AddCommand(contentBlueprintGetCmd)
	contentBlueprintCmd.AddCommand(contentBlueprintScaffoldCmd)
	contentBlueprintCmd.AddCommand(contentBlueprintCreateCmd)
	contentBlueprintCmd.AddCommand(contentBlueprintCreateFromCmd)
	contentBlueprintCmd.AddCommand(contentBlueprintDeleteCmd)

	contentCmd.AddCommand(contentBlueprintCmd)
}
