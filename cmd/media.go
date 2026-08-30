package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/Eventid3/umbraco-cli/internal/output"
	"github.com/spf13/cobra"
)

// mediaItem is the tree response shape. Like documents, media has no top-level
// "name" — it lives in variants[0].name (confirmed against a live instance) — so
// Name is populated from Variants after decoding rather than via a json tag.
type mediaItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	HasChildren bool   `json:"hasChildren"`
	IsFolder    bool   `json:"isFolder"`
	Variants    []struct {
		Name string `json:"name"`
	} `json:"variants"`
}

func (m *mediaItem) resolveName() {
	if m.Name == "" && len(m.Variants) > 0 {
		m.Name = m.Variants[0].Name
	}
}

type mediaListResponse struct {
	Total int         `json:"total"`
	Items []mediaItem `json:"items"`
}

var mediaCmd = &cobra.Command{
	Use:   "media",
	Short: "Manage Umbraco media items",
}

var mediaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List media items",
	Long: `List media items at the library root, or pass --parent to list the
contents of a specific media folder instead.`,
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
			path = fmt.Sprintf("/tree/media/children?parentId=%s&skip=%d&take=%d", parent, skip, take)
		} else {
			path = fmt.Sprintf("/tree/media/root?skip=%d&take=%d", skip, take)
		}
		var result mediaListResponse
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
			output.Line("No media items found.")
			return nil
		}
		headers := []string{"ID", "NAME", "HAS CHILDREN", "IS FOLDER"}
		rows := make([][]string, 0, len(result.Items))
		for _, m := range result.Items {
			rows = append(rows, []string{m.ID, m.Name, boolStr(m.HasChildren), boolStr(m.IsFolder)})
		}
		output.Table(headers, rows)
		return nil
	},
}

var mediaGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a media item by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result map[string]interface{}
		if err := client.Get("/media/"+args[0], &result); err != nil {
			return err
		}
		output.JSON(result)
		return nil
	},
}

var mediaDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a media item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		if err := client.Delete("/media/" + args[0]); err != nil {
			return err
		}
		output.Line("Media item %s deleted.", args[0])
		return nil
	},
}

var mediaUploadCmd = &cobra.Command{
	Use:   "upload <file>",
	Short: "Upload a file as a media item",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		parentID, _ := cmd.Flags().GetString("parent")
		mediaTypeAlias, _ := cmd.Flags().GetString("media-type")

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}

		// Step 1: upload the file as a temporary file.
		tempID, err := client.UploadTempFile(filePath)
		if err != nil {
			return fmt.Errorf("temp file upload failed: %w", err)
		}

		// Step 2: create the media item referencing the temp file.
		fileName := filepath.Base(filePath)
		body := map[string]interface{}{
			"contentTypeAlias": mediaTypeAlias,
			"parent":           nil,
			"values": []map[string]interface{}{
				{
					"alias": "umbracoFile",
					"value": map[string]interface{}{
						"temporaryFileId": tempID,
						"fileName":        fileName,
					},
				},
				{
					"alias": "umbracoName",
					"value": fileName,
				},
			},
		}
		if parentID != "" {
			body["parent"] = map[string]string{"id": parentID}
		}

		id, err := client.PostCreate("/media", body)
		if err != nil {
			return fmt.Errorf("media creation failed: %w", err)
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": id, "name": fileName})
		} else {
			output.Line("Media item created: %s (%s)", fileName, id)
		}
		return nil
	},
}

func init() {
	mediaListCmd.Flags().Int("skip", 0, "Number of items to skip")
	mediaListCmd.Flags().Int("take", 100, "Number of items to return")
	mediaListCmd.Flags().String("parent", "", "Parent media folder ID (list its contents instead of the root)")
	mediaUploadCmd.Flags().String("parent", "", "Parent media folder ID")
	mediaUploadCmd.Flags().String("media-type", "Image", "Media type alias (e.g. Image, File, Video)")

	mediaCmd.AddCommand(mediaListCmd)
	mediaCmd.AddCommand(mediaGetCmd)
	mediaCmd.AddCommand(mediaDeleteCmd)
	mediaCmd.AddCommand(mediaUploadCmd)
}
