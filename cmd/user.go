package cmd

import (
	"fmt"

	"github.com/Eventid3/umbraco-cli/internal/output"
	"github.com/spf13/cobra"
)

type umbracoUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	State string `json:"state"`
}

type userListResponse struct {
	Total int           `json:"total"`
	Items []umbracoUser `json:"items"`
}

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage Umbraco backoffice users",
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List backoffice users",
	RunE: func(cmd *cobra.Command, args []string) error {
		skip, _ := cmd.Flags().GetInt("skip")
		take, _ := cmd.Flags().GetInt("take")

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result userListResponse
		if err := client.Get("/user?skip="+itoa(skip)+"&take="+itoa(take), &result); err != nil {
			return err
		}
		if output.IsJSON() {
			output.JSON(result)
			return nil
		}
		output.Line("Total: %d", result.Total)
		if len(result.Items) == 0 {
			output.Line("No users found.")
			return nil
		}
		headers := []string{"ID", "NAME", "EMAIL", "STATE"}
		rows := make([][]string, 0, len(result.Items))
		for _, u := range result.Items {
			rows = append(rows, []string{u.ID, u.Name, u.Email, u.State})
		}
		output.Table(headers, rows)
		return nil
	},
}

var userGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a user by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result map[string]interface{}
		if err := client.Get("/user/"+args[0], &result); err != nil {
			return err
		}
		output.JSON(result)
		return nil
	},
}

var userGroupListCmd = &cobra.Command{
	Use:   "group-list",
	Short: "List user groups",
	RunE: func(cmd *cobra.Command, args []string) error {
		skip, _ := cmd.Flags().GetInt("skip")
		take, _ := cmd.Flags().GetInt("take")

		client, err := newClientFromFlags()
		if err != nil {
			return err
		}
		var result map[string]interface{}
		if err := client.Get("/user-group?skip="+itoa(skip)+"&take="+itoa(take), &result); err != nil {
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
			output.Line("No user groups found.")
			return nil
		}
		headers := []string{"ID", "NAME", "ALIAS", "DESCRIPTION"}
		rows := make([][]string, 0, len(items))
		for _, raw := range items {
			item, _ := raw.(map[string]interface{})
			id, _ := item["id"].(string)
			name, _ := item["name"].(string)
			alias, _ := item["alias"].(string)
			desc, _ := item["description"].(string)
			rows = append(rows, []string{id, name, alias, desc})
		}
		output.Table(headers, rows)
		return nil
	},
}

func init() {
	userListCmd.Flags().Int("skip", 0, "Number of items to skip")
	userListCmd.Flags().Int("take", 100, "Number of items to return")
	userGroupListCmd.Flags().Int("skip", 0, "Number of items to skip")
	userGroupListCmd.Flags().Int("take", 100, "Number of items to return")

	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userGetCmd)
	userCmd.AddCommand(userGroupListCmd)
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
