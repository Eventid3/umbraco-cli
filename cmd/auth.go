package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"syscall"

	"github.com/Eventid3/umbraco-cli/internal/config"
	"github.com/Eventid3/umbraco-cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Umbraco credential profiles",
}

var authAddCmd = &cobra.Command{
	Use:   "add <profile-name>",
	Short: "Add or update a credential profile",
	Long: `Add or update a named credential profile.

Credentials can be supplied as flags (non-interactive) or entered via prompts:

  umbraco auth add local \
    --base-url https://localhost:44382 \
    --client-id umbraco-back-office-umbraco-cli \
    --client-secret my-secret \
    --insecure`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Read from flags if provided, otherwise prompt interactively.
		baseURL, _ := cmd.Flags().GetString("base-url")
		clientID, _ := cmd.Flags().GetString("client-id")
		clientSecret, _ := cmd.Flags().GetString("client-secret")
		insecure, _ := cmd.Flags().GetBool("insecure")

		if baseURL == "" {
			baseURL = prompt("Base URL (e.g. https://mysite.com): ")
		}
		if clientID == "" {
			clientID = prompt("Client ID: ")
		}
		if clientSecret == "" {
			clientSecret = promptSecret("Client Secret: ")
		}

		p := config.Profile{
			BaseURL:      strings.TrimRight(baseURL, "/"),
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Insecure:     insecure,
		}

		if insecure {
			fmt.Fprintln(os.Stderr, "Warning: TLS verification is disabled for this profile. Do not use in production.")
		}

		// Validate by attempting to fetch a token.
		fmt.Print("Validating credentials... ")
		client, err := newClientWithProfile(p)
		if err != nil {
			fmt.Println("failed")
			return fmt.Errorf("credential validation failed: %w", err)
		}
		_ = client
		fmt.Println("OK")

		if err := config.SaveProfile(name, p); err != nil {
			return fmt.Errorf("cannot save profile: %w", err)
		}

		output.Line("Profile %q saved.", name)
		return nil
	},
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		names := config.ListProfiles(cfgViper)
		if len(names) == 0 {
			output.Line("No profiles configured. Run `umbraco auth add <name>` to add one.")
			return nil
		}
		sort.Strings(names)
		defaultProfile := cfgViper.GetString("default_profile")

		if output.IsJSON() {
			type profileEntry struct {
				Name      string `json:"name"`
				IsDefault bool   `json:"default"`
				BaseURL   string `json:"base_url"`
				Insecure  bool   `json:"insecure"`
			}
			entries := make([]profileEntry, 0, len(names))
			for _, n := range names {
				p, _ := config.GetProfile(cfgViper, n)
				entry := profileEntry{Name: n, IsDefault: n == defaultProfile}
				if p != nil {
					entry.BaseURL = p.BaseURL
					entry.Insecure = p.Insecure
				}
				entries = append(entries, entry)
			}
			output.JSON(entries)
			return nil
		}

		headers := []string{"NAME", "BASE URL", "DEFAULT", "INSECURE"}
		rows := make([][]string, 0, len(names))
		for _, n := range names {
			p, _ := config.GetProfile(cfgViper, n)
			baseURL := "(invalid)"
			insecure := ""
			if p != nil {
				baseURL = p.BaseURL
				if p.Insecure {
					insecure = "yes"
				}
			}
			isDefault := ""
			if n == defaultProfile {
				isDefault = "*"
			}
			rows = append(rows, []string{n, baseURL, isDefault, insecure})
		}
		output.Table(headers, rows)
		return nil
	},
}

var authRemoveCmd = &cobra.Command{
	Use:   "remove <profile-name>",
	Short: "Remove a credential profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := config.RemoveProfile(name); err != nil {
			return err
		}
		output.Line("Profile %q removed.", name)
		return nil
	},
}

var authSetDefaultCmd = &cobra.Command{
	Use:   "set-default <profile-name>",
	Short: "Set the default profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		names := config.ListProfiles(cfgViper)
		found := false
		for _, n := range names {
			if n == name {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("profile %q not found", name)
		}

		dir, err := config.ConfigDir()
		if err != nil {
			return err
		}
		cfgViper.Set("default_profile", name)
		return cfgViper.WriteConfigAs(dir + "/config.toml")
	},
}

func init() {
	authAddCmd.Flags().String("base-url", "", "Base URL of the Umbraco instance")
	authAddCmd.Flags().String("client-id", "", "OAuth2 client ID")
	authAddCmd.Flags().String("client-secret", "", "OAuth2 client secret")
	authAddCmd.Flags().Bool("insecure", false, "Disable TLS certificate verification (for self-signed certs in local dev)")

	authCmd.AddCommand(authAddCmd)
	authCmd.AddCommand(authListCmd)
	authCmd.AddCommand(authRemoveCmd)
	authCmd.AddCommand(authSetDefaultCmd)
}

// prompt reads a line from stdin with a visible prompt.
func prompt(label string) string {
	fmt.Print(label)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// promptSecret reads a password without echoing to the terminal.
func promptSecret(label string) string {
	fmt.Print(label)
	b, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		// Fallback to plain prompt if not a tty (e.g. piped input).
		return prompt("")
	}
	return string(b)
}
