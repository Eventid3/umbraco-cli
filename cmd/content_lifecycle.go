package cmd

import (
	"fmt"
	"net/url"

	"github.com/Eventid3/umbraco-cli/internal/output"
	"github.com/spf13/cobra"
)

// nullableTargetRef builds a {"id": id} reference, or nil for an empty id —
// the shape the Management API expects for an optional parent/target reference.
func nullableTargetRef(id string) interface{} {
	if id == "" {
		return nil
	}
	return map[string]interface{}{"id": id}
}

// printRecycleBinList prints a recycle-bin listing, where the display name is
// nested in variants[0].name rather than a top-level "name" field.
func printRecycleBinList(result map[string]interface{}) {
	items, _ := result["items"].([]interface{})
	total, _ := result["total"].(float64)
	output.Line("Total: %d", int(total))
	if len(items) == 0 {
		output.Line("No items found.")
		return
	}
	headers := []string{"ID", "NAME", "HAS CHILDREN"}
	rows := make([][]string, 0, len(items))
	for _, raw := range items {
		item, _ := raw.(map[string]interface{})
		id, _ := item["id"].(string)
		hasChildren, _ := item["hasChildren"].(bool)
		name := ""
		if variants, ok := item["variants"].([]interface{}); ok && len(variants) > 0 {
			if v0, ok := variants[0].(map[string]interface{}); ok {
				name, _ = v0["name"].(string)
			}
		}
		rows = append(rows, []string{id, name, boolStr(hasChildren)})
	}
	output.Table(headers, rows)
}

// ── copy / move ──────────────────────────────────────────────────────────────

var contentCopyCmd = &cobra.Command{
	Use:   "copy <id>",
	Short: "Copy a document to a new location",
	Long: `Copy a document. --target is the new parent (omit to copy to root).

The Management API does not return the new document's ID in the response —
if this command can't recover one from the response headers, list the
target parent's children afterwards to find the copy.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		relate, _ := cmd.Flags().GetBool("relate-to-original")
		includeDescendants, _ := cmd.Flags().GetBool("include-descendants")

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		body := map[string]interface{}{
			"target":             nullableTargetRef(target),
			"relateToOriginal":   relate,
			"includeDescendants": includeDescendants,
		}
		id, err := client.PostCreate("/document/"+args[0]+"/copy", body)
		if err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"newId": id})
			return nil
		}
		if id != "" {
			output.Line("Document %s copied (new id: %s).", args[0], id)
		} else {
			output.Line("Document %s copied. The API did not return the new ID — list the target parent's children to find it.", args[0])
		}
		return nil
	},
}

var contentMoveCmd = &cobra.Command{
	Use:   "move <id>",
	Short: "Move a document to a new parent",
	Long:  `--target is the new parent (omit to move to root).`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		body := map[string]interface{}{"target": nullableTargetRef(target)}
		if err := client.Put("/document/"+args[0]+"/move", body, nil); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": args[0], "status": "moved"})
		} else {
			output.Line("Document %s moved.", args[0])
		}
		return nil
	},
}

// ── recycle bin ──────────────────────────────────────────────────────────────

var contentTrashCmd = &cobra.Command{
	Use:   "trash <id>",
	Short: "Move a document to the recycle bin",
	Long:  `Unlike 'content delete', this is reversible via 'content restore'.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		if err := client.Put("/document/"+args[0]+"/move-to-recycle-bin", nil, nil); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": args[0], "status": "trashed"})
		} else {
			output.Line("Document %s moved to the recycle bin.", args[0])
		}
		return nil
	},
}

var contentRestoreCmd = &cobra.Command{
	Use:   "restore <id>",
	Short: "Restore a document from the recycle bin",
	Long:  `--target restores under a specific parent; omit it to restore to the document's original parent.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		// Always send an explicit body (never omit it entirely) — the endpoint
		// returns 415 Unsupported Media Type for a bodyless request even when
		// "target" is meant to be absent/null.
		body := map[string]interface{}{"target": nullableTargetRef(target)}
		if err := client.Put("/recycle-bin/document/"+args[0]+"/restore", body, nil); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": args[0], "status": "restored"})
		} else {
			output.Line("Document %s restored.", args[0])
		}
		return nil
	},
}

var contentTrashListCmd = &cobra.Command{
	Use:   "trash-list",
	Short: "List documents in the recycle bin",
	Long:  `List recycle bin root items, or pass --parent to list a trashed folder's contents.`,
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
			path = "/recycle-bin/document/children?parentId=" + parent + "&skip=" + itoa(skip) + "&take=" + itoa(take)
		} else {
			path = "/recycle-bin/document/root?skip=" + itoa(skip) + "&take=" + itoa(take)
		}
		var result map[string]interface{}
		if err := client.Get(path, &result); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(result)
			return nil
		}
		printRecycleBinList(result)
		return nil
	},
}

var contentTrashDeleteCmd = &cobra.Command{
	Use:   "trash-delete <id>",
	Short: "Permanently delete a single item already in the recycle bin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		if err := client.Delete("/recycle-bin/document/" + args[0]); err != nil {
			return err
		}
		output.Line("Recycle bin item %s permanently deleted.", args[0])
		return nil
	},
}

var contentTrashEmptyCmd = &cobra.Command{
	Use:   "trash-empty",
	Short: "Permanently empty the entire document recycle bin",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		if err := client.Delete("/recycle-bin/document"); err != nil {
			return err
		}
		output.Line("Document recycle bin emptied.")
		return nil
	},
}

// ── search / references ──────────────────────────────────────────────────────

var contentSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search documents by name",
	Long:  `Always returns JSON — the search result shape is too nested for a flat table.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skip, _ := cmd.Flags().GetInt("skip")
		take, _ := cmd.Flags().GetInt("take")
		trashed, _ := cmd.Flags().GetBool("trashed")
		culture, _ := cmd.Flags().GetString("culture")
		parent, _ := cmd.Flags().GetString("parent")

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		q := url.Values{}
		q.Set("query", args[0])
		q.Set("skip", itoa(skip))
		q.Set("take", itoa(take))
		if trashed {
			q.Set("trashed", "true")
		}
		if culture != "" {
			q.Set("culture", culture)
		}
		if parent != "" {
			q.Set("parentId", parent)
		}
		var result interface{}
		if err := client.Get("/item/document/search?"+q.Encode(), &result); err != nil {
			return err
		}
		output.JSON(result)
		return nil
	},
}

var contentReferencedByCmd = &cobra.Command{
	Use:   "referenced-by <id>",
	Short: "List items that reference a document (e.g. via a content picker)",
	Long:  `Check this before deleting a document to avoid leaving dangling references. Always returns JSON.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skip, _ := cmd.Flags().GetInt("skip")
		take, _ := cmd.Flags().GetInt("take")
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result interface{}
		if err := client.Get("/document/"+args[0]+"/referenced-by?skip="+itoa(skip)+"&take="+itoa(take), &result); err != nil {
			return err
		}
		output.JSON(result)
		return nil
	},
}

var contentAreReferencedCmd = &cobra.Command{
	Use:   "are-referenced --id <id> [--id <id> ...]",
	Short: "Check which of a set of documents are referenced by other content",
	Long:  `Bulk counterpart of 'referenced-by' — pass --id once per document to check. Always returns JSON.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ids, _ := cmd.Flags().GetStringSlice("id")
		skip, _ := cmd.Flags().GetInt("skip")
		take, _ := cmd.Flags().GetInt("take")
		if len(ids) == 0 {
			return fmt.Errorf("at least one --id is required")
		}
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		q := url.Values{}
		for _, id := range ids {
			q.Add("id", id)
		}
		q.Set("skip", itoa(skip))
		q.Set("take", itoa(take))
		var result interface{}
		if err := client.Get("/document/are-referenced?"+q.Encode(), &result); err != nil {
			return err
		}
		output.JSON(result)
		return nil
	},
}

// ── create-and-publish / sort ────────────────────────────────────────────────

var contentCreateAndPublishCmd = &cobra.Command{
	Use:   "create-and-publish",
	Short: "Create a document and publish it in one call (pass JSON via --file or stdin with --file=-)",
	Long: `Same body shape as 'content create'. A client-generated ID is added to the
body automatically (if not already present) so this command can always report
back the new document's ID, regardless of what the API response contains.

--cultures controls which variants get published (culturesToPublish); leave
it empty for an invariant document.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		raw, err := readJSONInput(cmd, "file")
		if err != nil {
			return err
		}
		body, ok := raw.(map[string]interface{})
		if !ok {
			return fmt.Errorf("--file body must be a JSON object")
		}
		if id, ok := body["id"].(string); !ok || id == "" {
			body["id"] = newUUID()
		}
		cultures, _ := cmd.Flags().GetStringSlice("cultures")
		body["culturesToPublish"] = cultures

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		if err := client.Post("/document/create-and-publish", body, nil); err != nil {
			return err
		}
		id := body["id"].(string)
		if output.IsJSON() {
			output.JSON(map[string]string{"id": id})
		} else {
			output.Line("Document created and published: %s", id)
		}
		return nil
	},
}

var contentSortChildrenCmd = &cobra.Command{
	Use:   "sort-children [parent-id]",
	Short: "Auto-sort a document's children by name or date",
	Long: `Sorts the children of [parent-id] (or the content root, if omitted) by
--field in --direction order. For arbitrary manual ordering, sort each node
individually with 'content move'.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		field, _ := cmd.Flags().GetString("field")
		direction, _ := cmd.Flags().GetString("direction")
		culture, _ := cmd.Flags().GetString("culture")

		if field != "Name" && field != "CreateDate" && field != "UpdateDate" {
			return fmt.Errorf("--field must be one of: Name, CreateDate, UpdateDate")
		}
		if direction != "Ascending" && direction != "Descending" {
			return fmt.Errorf("--direction must be one of: Ascending, Descending")
		}

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		body := map[string]interface{}{
			"culture":   nullableString(culture),
			"field":     field,
			"direction": direction,
		}
		var path string
		if len(args) == 1 {
			path = "/document/" + args[0] + "/sort-children"
		} else {
			path = "/document/root/sort-children"
		}
		if err := client.Put(path, body, nil); err != nil {
			return err
		}
		output.Line("Children sorted by %s (%s).", field, direction)
		return nil
	},
}

func init() {
	contentCopyCmd.Flags().String("target", "", "New parent document ID (omit to copy to root)")
	contentCopyCmd.Flags().Bool("relate-to-original", false, "Track a relation back to the original document")
	contentCopyCmd.Flags().Bool("include-descendants", true, "Also copy descendant documents")

	contentMoveCmd.Flags().String("target", "", "New parent document ID (omit to move to root)")

	contentRestoreCmd.Flags().String("target", "", "Parent document ID to restore under (omit to restore to the original parent)")

	contentTrashListCmd.Flags().Int("skip", 0, "Number of items to skip")
	contentTrashListCmd.Flags().Int("take", 100, "Number of items to return")
	contentTrashListCmd.Flags().String("parent", "", "Trashed parent ID (list its contents instead of the recycle bin root)")

	contentSearchCmd.Flags().Int("skip", 0, "Number of items to skip")
	contentSearchCmd.Flags().Int("take", 100, "Number of items to return")
	contentSearchCmd.Flags().Bool("trashed", false, "Search within the recycle bin instead of live content")
	contentSearchCmd.Flags().String("culture", "", "Restrict results to a culture variant")
	contentSearchCmd.Flags().String("parent", "", "Restrict results to children of this document")

	contentReferencedByCmd.Flags().Int("skip", 0, "Number of items to skip")
	contentReferencedByCmd.Flags().Int("take", 20, "Number of items to return")

	contentAreReferencedCmd.Flags().StringSlice("id", []string{}, "Document ID to check (repeatable)")
	contentAreReferencedCmd.Flags().Int("skip", 0, "Number of items to skip")
	contentAreReferencedCmd.Flags().Int("take", 100, "Number of items to return")

	contentCreateAndPublishCmd.Flags().String("file", "", "Path to JSON file (use - for stdin)")
	contentCreateAndPublishCmd.Flags().StringSlice("cultures", []string{}, "Cultures to publish (empty = invariant document)")

	contentSortChildrenCmd.Flags().String("field", "Name", "Sort field: Name, CreateDate, or UpdateDate")
	contentSortChildrenCmd.Flags().String("direction", "Ascending", "Sort direction: Ascending or Descending")
	contentSortChildrenCmd.Flags().String("culture", "", "Culture to sort by (for variant document names)")

	contentCmd.AddCommand(contentCopyCmd)
	contentCmd.AddCommand(contentMoveCmd)
	contentCmd.AddCommand(contentTrashCmd)
	contentCmd.AddCommand(contentRestoreCmd)
	contentCmd.AddCommand(contentTrashListCmd)
	contentCmd.AddCommand(contentTrashDeleteCmd)
	contentCmd.AddCommand(contentTrashEmptyCmd)
	contentCmd.AddCommand(contentSearchCmd)
	contentCmd.AddCommand(contentReferencedByCmd)
	contentCmd.AddCommand(contentAreReferencedCmd)
	contentCmd.AddCommand(contentCreateAndPublishCmd)
	contentCmd.AddCommand(contentSortChildrenCmd)
}
