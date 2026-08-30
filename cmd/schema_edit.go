package cmd

import (
	"fmt"

	"github.com/Eventid3/umbraco-cli/internal/api"
	"github.com/Eventid3/umbraco-cli/internal/output"
	"github.com/spf13/cobra"
)

// fetchDocTypeMap fetches a document type as a raw map, for patch-style edits
// (add/update/remove a single property, container, or composition) without
// requiring the caller to round-trip the entire document type body by hand.
func fetchDocTypeMap(client *api.Client, id string) (map[string]interface{}, error) {
	var doc map[string]interface{}
	if err := client.Get("/document-type/"+id, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// putDocTypeMap writes a modified document type map back via PUT.
func putDocTypeMap(client *api.Client, id string, doc map[string]interface{}) error {
	return client.Put("/document-type/"+id, doc, nil)
}

// ── property subcommands ─────────────────────────────────────────────────────

var schemaDocTypePropertyCmd = &cobra.Command{
	Use:   "property",
	Short: "Add, update, or remove a single property on a document type",
	Long: `The Management API only exposes a whole-document-type PUT, so these
commands fetch the document type, patch a single property in place, and PUT
the whole thing back — sparing you from hand-editing the full properties
array for one-property changes.`,
}

var schemaDocTypePropertyAddCmd = &cobra.Command{
	Use:   "add <doctype-id>",
	Short: "Add a property to a document type",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias, _ := cmd.Flags().GetString("alias")
		name, _ := cmd.Flags().GetString("name")
		dataTypeID, _ := cmd.Flags().GetString("data-type")
		description, _ := cmd.Flags().GetString("description")
		container, _ := cmd.Flags().GetString("container")
		newContainerName, _ := cmd.Flags().GetString("new-container-name")
		newContainerType, _ := cmd.Flags().GetString("new-container-type")
		newContainerParent, _ := cmd.Flags().GetString("new-container-parent")
		sortOrder, _ := cmd.Flags().GetInt("sort-order")
		mandatory, _ := cmd.Flags().GetBool("mandatory")
		mandatoryMessage, _ := cmd.Flags().GetString("mandatory-message")
		regEx, _ := cmd.Flags().GetString("reg-ex")
		regExMessage, _ := cmd.Flags().GetString("reg-ex-message")
		variesByCulture, _ := cmd.Flags().GetBool("varies-by-culture")
		variesBySegment, _ := cmd.Flags().GetBool("varies-by-segment")
		labelOnTop, _ := cmd.Flags().GetBool("label-on-top")

		if alias == "" || name == "" || dataTypeID == "" {
			return fmt.Errorf("--alias, --name, and --data-type are required")
		}
		if container != "" && newContainerName != "" {
			return fmt.Errorf("--container and --new-container-name are mutually exclusive")
		}
		if newContainerName != "" && newContainerType == "" {
			return fmt.Errorf("--new-container-type (Tab or Group) is required when --new-container-name is set")
		}

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		doc, err := fetchDocTypeMap(client, args[0])
		if err != nil {
			return err
		}

		properties, _ := doc["properties"].([]interface{})
		for _, raw := range properties {
			if p, ok := raw.(map[string]interface{}); ok && p["alias"] == alias {
				return fmt.Errorf("property with alias %q already exists on this document type", alias)
			}
		}
		if !cmd.Flags().Changed("sort-order") {
			sortOrder = len(properties)
		}

		// A container with no properties is silently discarded by Umbraco on save
		// (confirmed empirically), so a brand-new container can only be created in
		// the very same PUT as the first property that uses it — never as a
		// standalone 'schema doctype container add' followed by a separate
		// 'property add --container <id>'.
		var newContainerID string
		if newContainerName != "" {
			containers, _ := doc["containers"].([]interface{})
			// Umbraco matches/merges containers by (name, type), not id: creating a
			// "new" container that collides with an existing one's name+type gets
			// merged into a single container under a fresh id, silently orphaning
			// (and dropping) any existing property still pointing at the old id.
			// Confirmed by direct testing — refuse rather than risk that data loss.
			for _, raw := range containers {
				if c, ok := raw.(map[string]interface{}); ok {
					if c["name"] == newContainerName && c["type"] == newContainerType {
						return fmt.Errorf("a container named %q of type %q already exists (id: %v) — use --container %v to attach to it instead of --new-container-name, which would silently merge and can drop existing properties", newContainerName, newContainerType, c["id"], c["id"])
					}
				}
			}
			newContainerID = newUUID()
			newContainer := map[string]interface{}{
				"id":        newContainerID,
				"name":      newContainerName,
				"type":      newContainerType,
				"sortOrder": len(containers),
			}
			if newContainerParent != "" {
				newContainer["parent"] = map[string]interface{}{"id": newContainerParent}
			} else {
				newContainer["parent"] = nil
			}
			doc["containers"] = append(containers, newContainer)
			container = newContainerID
		}

		id := newUUID()
		prop := map[string]interface{}{
			"id":              id,
			"alias":           alias,
			"name":            name,
			"description":     nullableString(description),
			"dataType":        map[string]interface{}{"id": dataTypeID},
			"sortOrder":       sortOrder,
			"variesByCulture": variesByCulture,
			"variesBySegment": variesBySegment,
			"validation": map[string]interface{}{
				"mandatory":        mandatory,
				"mandatoryMessage": nullableString(mandatoryMessage),
				"regEx":            nullableString(regEx),
				"regExMessage":     nullableString(regExMessage),
			},
			"appearance": map[string]interface{}{
				"labelOnTop": labelOnTop,
			},
		}
		if container != "" {
			prop["container"] = map[string]interface{}{"id": container}
		} else {
			prop["container"] = nil
		}

		doc["properties"] = append(properties, prop)

		if err := putDocTypeMap(client, args[0], doc); err != nil {
			return err
		}
		if output.IsJSON() {
			result := map[string]string{"id": id, "alias": alias, "status": "added"}
			if newContainerID != "" {
				result["containerId"] = newContainerID
			}
			output.JSON(result)
		} else if newContainerID != "" {
			output.Line("Property %q added to document type %s (id: %s), in new container %q (id: %s).", alias, args[0], id, newContainerName, newContainerID)
		} else {
			output.Line("Property %q added to document type %s (id: %s).", alias, args[0], id)
		}
		return nil
	},
}

var schemaDocTypePropertyUpdateCmd = &cobra.Command{
	Use:   "update <doctype-id>",
	Short: "Update an existing property on a document type (identified by --alias)",
	Long:  `Only flags explicitly passed are changed; everything else on the property is left as-is.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias, _ := cmd.Flags().GetString("alias")
		if alias == "" {
			return fmt.Errorf("--alias is required to identify the property to update")
		}

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		doc, err := fetchDocTypeMap(client, args[0])
		if err != nil {
			return err
		}

		properties, _ := doc["properties"].([]interface{})
		found := false
		for _, raw := range properties {
			p, ok := raw.(map[string]interface{})
			if !ok || p["alias"] != alias {
				continue
			}
			found = true

			if cmd.Flags().Changed("name") {
				v, _ := cmd.Flags().GetString("name")
				p["name"] = v
			}
			if cmd.Flags().Changed("description") {
				v, _ := cmd.Flags().GetString("description")
				p["description"] = nullableString(v)
			}
			if cmd.Flags().Changed("data-type") {
				v, _ := cmd.Flags().GetString("data-type")
				p["dataType"] = map[string]interface{}{"id": v}
			}
			if cmd.Flags().Changed("container") {
				v, _ := cmd.Flags().GetString("container")
				if v == "" {
					p["container"] = nil
				} else {
					p["container"] = map[string]interface{}{"id": v}
				}
			}
			if cmd.Flags().Changed("sort-order") {
				v, _ := cmd.Flags().GetInt("sort-order")
				p["sortOrder"] = v
			}
			if cmd.Flags().Changed("varies-by-culture") {
				v, _ := cmd.Flags().GetBool("varies-by-culture")
				p["variesByCulture"] = v
			}
			if cmd.Flags().Changed("varies-by-segment") {
				v, _ := cmd.Flags().GetBool("varies-by-segment")
				p["variesBySegment"] = v
			}
			if cmd.Flags().Changed("label-on-top") {
				v, _ := cmd.Flags().GetBool("label-on-top")
				appearance, _ := p["appearance"].(map[string]interface{})
				if appearance == nil {
					appearance = map[string]interface{}{}
				}
				appearance["labelOnTop"] = v
				p["appearance"] = appearance
			}

			validation, _ := p["validation"].(map[string]interface{})
			if validation == nil {
				validation = map[string]interface{}{}
			}
			if cmd.Flags().Changed("mandatory") {
				v, _ := cmd.Flags().GetBool("mandatory")
				validation["mandatory"] = v
			}
			if cmd.Flags().Changed("mandatory-message") {
				v, _ := cmd.Flags().GetString("mandatory-message")
				validation["mandatoryMessage"] = nullableString(v)
			}
			if cmd.Flags().Changed("reg-ex") {
				v, _ := cmd.Flags().GetString("reg-ex")
				validation["regEx"] = nullableString(v)
			}
			if cmd.Flags().Changed("reg-ex-message") {
				v, _ := cmd.Flags().GetString("reg-ex-message")
				validation["regExMessage"] = nullableString(v)
			}
			p["validation"] = validation
			break
		}
		if !found {
			return fmt.Errorf("no property with alias %q found on document type %s", alias, args[0])
		}

		if err := putDocTypeMap(client, args[0], doc); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"alias": alias, "status": "updated"})
		} else {
			output.Line("Property %q updated on document type %s.", alias, args[0])
		}
		return nil
	},
}

var schemaDocTypePropertyRemoveCmd = &cobra.Command{
	Use:   "remove <doctype-id>",
	Short: "Remove a property from a document type (identified by --alias)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias, _ := cmd.Flags().GetString("alias")
		if alias == "" {
			return fmt.Errorf("--alias is required")
		}

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		doc, err := fetchDocTypeMap(client, args[0])
		if err != nil {
			return err
		}

		properties, _ := doc["properties"].([]interface{})
		kept := make([]interface{}, 0, len(properties))
		found := false
		for _, raw := range properties {
			if p, ok := raw.(map[string]interface{}); ok && p["alias"] == alias {
				found = true
				continue
			}
			kept = append(kept, raw)
		}
		if !found {
			return fmt.Errorf("no property with alias %q found on document type %s", alias, args[0])
		}
		doc["properties"] = kept

		if err := putDocTypeMap(client, args[0], doc); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"alias": alias, "status": "removed"})
		} else {
			output.Line("Property %q removed from document type %s.", alias, args[0])
		}
		return nil
	},
}

// ── container (tab/group) subcommands ───────────────────────────────────────

var schemaDocTypeContainerCmd = &cobra.Command{
	Use:   "container",
	Short: "Add, update, or remove a tab/group container on a document type",
}

var schemaDocTypeContainerAddCmd = &cobra.Command{
	Use:   "add <doctype-id>",
	Short: "Add a tab or group container to a document type",
	Long: `WARNING: a container with zero properties is silently discarded by Umbraco
when it saves the document type (confirmed against a live instance) — this
command alone is a no-op for a brand-new container, and a follow-up
'property add --container <id>' referencing it will fail because the
container is already gone. To create a new container and its first property
together in one atomic operation, use:

  schema doctype property add <doctype-id> --alias ... --name ... --data-type ... \
    --new-container-name "<name>" --new-container-type Tab|Group

Only use this command to add an additional container once the document type
already has at least one property (so the save won't prune it), or to
restructure containers that already hold properties.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		containerType, _ := cmd.Flags().GetString("type")
		parent, _ := cmd.Flags().GetString("parent")
		sortOrder, _ := cmd.Flags().GetInt("sort-order")

		if name == "" || containerType == "" {
			return fmt.Errorf("--name and --type (Tab or Group) are required")
		}

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		doc, err := fetchDocTypeMap(client, args[0])
		if err != nil {
			return err
		}

		containers, _ := doc["containers"].([]interface{})
		if !cmd.Flags().Changed("sort-order") {
			sortOrder = len(containers)
		}

		id := newUUID()
		container := map[string]interface{}{
			"id":        id,
			"name":      name,
			"type":      containerType,
			"sortOrder": sortOrder,
		}
		if parent != "" {
			container["parent"] = map[string]interface{}{"id": parent}
		} else {
			container["parent"] = nil
		}
		doc["containers"] = append(containers, container)

		if err := putDocTypeMap(client, args[0], doc); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": id, "status": "added"})
		} else {
			output.Line("Container %q added to document type %s (id: %s).", name, args[0], id)
		}
		return nil
	},
}

var schemaDocTypeContainerUpdateCmd = &cobra.Command{
	Use:   "update <doctype-id>",
	Short: "Update a tab/group container on a document type (identified by --id)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required to identify the container to update")
		}

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		doc, err := fetchDocTypeMap(client, args[0])
		if err != nil {
			return err
		}

		containers, _ := doc["containers"].([]interface{})
		found := false
		for _, raw := range containers {
			c, ok := raw.(map[string]interface{})
			if !ok || c["id"] != id {
				continue
			}
			found = true
			if cmd.Flags().Changed("name") {
				v, _ := cmd.Flags().GetString("name")
				c["name"] = v
			}
			if cmd.Flags().Changed("type") {
				v, _ := cmd.Flags().GetString("type")
				c["type"] = v
			}
			if cmd.Flags().Changed("sort-order") {
				v, _ := cmd.Flags().GetInt("sort-order")
				c["sortOrder"] = v
			}
			if cmd.Flags().Changed("parent") {
				v, _ := cmd.Flags().GetString("parent")
				if v == "" {
					c["parent"] = nil
				} else {
					c["parent"] = map[string]interface{}{"id": v}
				}
			}
			break
		}
		if !found {
			return fmt.Errorf("no container with id %q found on document type %s", id, args[0])
		}

		if err := putDocTypeMap(client, args[0], doc); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": id, "status": "updated"})
		} else {
			output.Line("Container %s updated on document type %s.", id, args[0])
		}
		return nil
	},
}

var schemaDocTypeContainerRemoveCmd = &cobra.Command{
	Use:   "remove <doctype-id>",
	Short: "Remove a tab/group container from a document type (identified by --id)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		if id == "" {
			return fmt.Errorf("--id is required")
		}

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		doc, err := fetchDocTypeMap(client, args[0])
		if err != nil {
			return err
		}

		containers, _ := doc["containers"].([]interface{})
		kept := make([]interface{}, 0, len(containers))
		found := false
		for _, raw := range containers {
			if c, ok := raw.(map[string]interface{}); ok && c["id"] == id {
				found = true
				continue
			}
			kept = append(kept, raw)
		}
		if !found {
			return fmt.Errorf("no container with id %q found on document type %s", id, args[0])
		}
		doc["containers"] = kept

		if err := putDocTypeMap(client, args[0], doc); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"id": id, "status": "removed"})
		} else {
			output.Line("Container %s removed from document type %s.", id, args[0])
		}
		return nil
	},
}

// ── composition subcommands ──────────────────────────────────────────────────

var schemaDocTypeCompositionCmd = &cobra.Command{
	Use:   "composition",
	Short: "Add or remove a composition/inheritance link on a document type",
}

var schemaDocTypeCompositionAddCmd = &cobra.Command{
	Use:   "add <doctype-id>",
	Short: "Compose another document type into a document type",
	Long: `Use 'schema doctype available-compositions <id>' first to check which
document types are compatible (no clashing property aliases or cycles).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compositionID, _ := cmd.Flags().GetString("composition-id")
		compositionType, _ := cmd.Flags().GetString("type")
		if compositionID == "" {
			return fmt.Errorf("--composition-id is required")
		}
		if compositionType != "Composition" && compositionType != "Inheritance" {
			return fmt.Errorf("--type must be \"Composition\" or \"Inheritance\"")
		}

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		doc, err := fetchDocTypeMap(client, args[0])
		if err != nil {
			return err
		}

		compositions, _ := doc["compositions"].([]interface{})
		for _, raw := range compositions {
			if c, ok := raw.(map[string]interface{}); ok {
				if dt, ok := c["documentType"].(map[string]interface{}); ok && dt["id"] == compositionID {
					return fmt.Errorf("document type %s is already composed into this document type", compositionID)
				}
			}
		}
		doc["compositions"] = append(compositions, map[string]interface{}{
			"documentType":    map[string]interface{}{"id": compositionID},
			"compositionType": compositionType,
		})

		if err := putDocTypeMap(client, args[0], doc); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"compositionId": compositionID, "status": "added"})
		} else {
			output.Line("Composition %s added to document type %s.", compositionID, args[0])
		}
		return nil
	},
}

var schemaDocTypeCompositionRemoveCmd = &cobra.Command{
	Use:   "remove <doctype-id>",
	Short: "Remove a composition from a document type",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compositionID, _ := cmd.Flags().GetString("composition-id")
		if compositionID == "" {
			return fmt.Errorf("--composition-id is required")
		}

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		doc, err := fetchDocTypeMap(client, args[0])
		if err != nil {
			return err
		}

		compositions, _ := doc["compositions"].([]interface{})
		kept := make([]interface{}, 0, len(compositions))
		found := false
		for _, raw := range compositions {
			if c, ok := raw.(map[string]interface{}); ok {
				if dt, ok := c["documentType"].(map[string]interface{}); ok && dt["id"] == compositionID {
					found = true
					continue
				}
			}
			kept = append(kept, raw)
		}
		if !found {
			return fmt.Errorf("document type %s is not composed into this document type", compositionID)
		}
		doc["compositions"] = kept

		if err := putDocTypeMap(client, args[0], doc); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(map[string]string{"compositionId": compositionID, "status": "removed"})
		} else {
			output.Line("Composition %s removed from document type %s.", compositionID, args[0])
		}
		return nil
	},
}

func init() {
	schemaDocTypePropertyAddCmd.Flags().String("alias", "", "Property alias (required)")
	schemaDocTypePropertyAddCmd.Flags().String("name", "", "Property display name (required)")
	schemaDocTypePropertyAddCmd.Flags().String("data-type", "", "Data type ID (required)")
	schemaDocTypePropertyAddCmd.Flags().String("description", "", "Property description")
	schemaDocTypePropertyAddCmd.Flags().String("container", "", "Existing parent container (tab/group) ID")
	schemaDocTypePropertyAddCmd.Flags().String("new-container-name", "", "Create a new container for this property instead of using --container (required together with --new-container-type)")
	schemaDocTypePropertyAddCmd.Flags().String("new-container-type", "", "Type of the new container: Tab or Group")
	schemaDocTypePropertyAddCmd.Flags().String("new-container-parent", "", "Parent container ID for the new container (for a Group nested under a Tab)")
	schemaDocTypePropertyAddCmd.Flags().Int("sort-order", 0, "Sort order (default: appended at the end)")
	schemaDocTypePropertyAddCmd.Flags().Bool("mandatory", false, "Mark the property as mandatory")
	schemaDocTypePropertyAddCmd.Flags().String("mandatory-message", "", "Custom message when mandatory validation fails")
	schemaDocTypePropertyAddCmd.Flags().String("reg-ex", "", "Regex validation pattern")
	schemaDocTypePropertyAddCmd.Flags().String("reg-ex-message", "", "Custom message when regex validation fails")
	schemaDocTypePropertyAddCmd.Flags().Bool("varies-by-culture", false, "Property value varies by culture")
	schemaDocTypePropertyAddCmd.Flags().Bool("varies-by-segment", false, "Property value varies by segment")
	schemaDocTypePropertyAddCmd.Flags().Bool("label-on-top", false, "Display the property label above the editor")

	schemaDocTypePropertyUpdateCmd.Flags().String("alias", "", "Alias of the property to update (required)")
	schemaDocTypePropertyUpdateCmd.Flags().String("name", "", "New display name")
	schemaDocTypePropertyUpdateCmd.Flags().String("data-type", "", "New data type ID")
	schemaDocTypePropertyUpdateCmd.Flags().String("description", "", "New description")
	schemaDocTypePropertyUpdateCmd.Flags().String("container", "", "New parent container ID (empty string clears it)")
	schemaDocTypePropertyUpdateCmd.Flags().Int("sort-order", 0, "New sort order")
	schemaDocTypePropertyUpdateCmd.Flags().Bool("mandatory", false, "Mark the property as mandatory")
	schemaDocTypePropertyUpdateCmd.Flags().String("mandatory-message", "", "Custom message when mandatory validation fails")
	schemaDocTypePropertyUpdateCmd.Flags().String("reg-ex", "", "Regex validation pattern")
	schemaDocTypePropertyUpdateCmd.Flags().String("reg-ex-message", "", "Custom message when regex validation fails")
	schemaDocTypePropertyUpdateCmd.Flags().Bool("varies-by-culture", false, "Property value varies by culture")
	schemaDocTypePropertyUpdateCmd.Flags().Bool("varies-by-segment", false, "Property value varies by segment")
	schemaDocTypePropertyUpdateCmd.Flags().Bool("label-on-top", false, "Display the property label above the editor")

	schemaDocTypePropertyRemoveCmd.Flags().String("alias", "", "Alias of the property to remove (required)")

	schemaDocTypePropertyCmd.AddCommand(schemaDocTypePropertyAddCmd)
	schemaDocTypePropertyCmd.AddCommand(schemaDocTypePropertyUpdateCmd)
	schemaDocTypePropertyCmd.AddCommand(schemaDocTypePropertyRemoveCmd)

	schemaDocTypeContainerAddCmd.Flags().String("name", "", "Container name (required)")
	schemaDocTypeContainerAddCmd.Flags().String("type", "", "Container type: Tab or Group (required)")
	schemaDocTypeContainerAddCmd.Flags().String("parent", "", "Parent container ID (for a Group nested under a Tab)")
	schemaDocTypeContainerAddCmd.Flags().Int("sort-order", 0, "Sort order (default: appended at the end)")

	schemaDocTypeContainerUpdateCmd.Flags().String("id", "", "ID of the container to update (required)")
	schemaDocTypeContainerUpdateCmd.Flags().String("name", "", "New container name")
	schemaDocTypeContainerUpdateCmd.Flags().String("type", "", "New container type: Tab or Group")
	schemaDocTypeContainerUpdateCmd.Flags().String("parent", "", "New parent container ID (empty string clears it)")
	schemaDocTypeContainerUpdateCmd.Flags().Int("sort-order", 0, "New sort order")

	schemaDocTypeContainerRemoveCmd.Flags().String("id", "", "ID of the container to remove (required)")

	schemaDocTypeContainerCmd.AddCommand(schemaDocTypeContainerAddCmd)
	schemaDocTypeContainerCmd.AddCommand(schemaDocTypeContainerUpdateCmd)
	schemaDocTypeContainerCmd.AddCommand(schemaDocTypeContainerRemoveCmd)

	schemaDocTypeCompositionAddCmd.Flags().String("composition-id", "", "ID of the document type to compose in (required)")
	schemaDocTypeCompositionAddCmd.Flags().String("type", "Composition", "Composition type: Composition or Inheritance")
	schemaDocTypeCompositionRemoveCmd.Flags().String("composition-id", "", "ID of the composed document type to remove (required)")

	schemaDocTypeCompositionCmd.AddCommand(schemaDocTypeCompositionAddCmd)
	schemaDocTypeCompositionCmd.AddCommand(schemaDocTypeCompositionRemoveCmd)

	schemaDocTypeCmd.AddCommand(schemaDocTypePropertyCmd)
	schemaDocTypeCmd.AddCommand(schemaDocTypeContainerCmd)
	schemaDocTypeCmd.AddCommand(schemaDocTypeCompositionCmd)
}
