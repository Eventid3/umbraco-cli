package cmd

import (
	"fmt"
	"os"

	"github.com/Eventid3/umbraco-cli/internal/api"
	"github.com/Eventid3/umbraco-cli/internal/config"
	"github.com/Eventid3/umbraco-cli/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgViper    *viper.Viper
	profileName string
	jsonFlag    bool
	baseURLFlag string
)

var rootCmd = &cobra.Command{
	Use:   "umbraco",
	Short: "CLI for the Umbraco Management API",
	Long: `umbraco is a command-line tool for managing Umbraco CMS instances
via the Management API. It is designed for both human operators and
AI agents.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		output.SetJSON(jsonFlag)

		// Warn whenever a command will use an insecure profile.
		// Skip this check for auth/agent subcommands (they don't call the API).
		if cmd.Parent() != nil && cmd.Parent().Use != "auth" && cmd.Parent().Use != "agent" {
			if name := activeProfile(); name != "" && cfgViper != nil {
				p, err := config.GetProfile(cfgViper, name)
				if err == nil && p.Insecure {
					fmt.Fprintf(os.Stderr, "Warning: TLS verification is disabled for profile %q.\n", name)
				}
			}
		}
		return nil
	},
}

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		output.Error(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&profileName, "profile", "p", "", "Named credential profile to use (default: value of default_profile in config)")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output results as JSON")
	rootCmd.PersistentFlags().StringVar(&baseURLFlag, "base-url", "", "Override the base URL for this invocation")

	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(contentCmd)
	rootCmd.AddCommand(mediaCmd)
	rootCmd.AddCommand(userCmd)
	rootCmd.AddCommand(schemaCmd)
	rootCmd.AddCommand(systemCmd)
	rootCmd.AddCommand(agentCmd)
}

func initConfig() {
	v, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
		cfgViper = viper.New()
		return
	}
	cfgViper = v
}

// activeProfile resolves the profile name to use, in priority order:
// --profile flag > UMBRACO_PROFILE env > default_profile in config.
func activeProfile() string {
	if profileName != "" {
		return profileName
	}
	if env := os.Getenv("UMBRACO_PROFILE"); env != "" {
		return env
	}
	if cfgViper != nil {
		return cfgViper.GetString("default_profile")
	}
	return ""
}

// newClientWithProfile creates an API client directly from a Profile struct.
// Used during auth add to validate credentials before saving.
func newClientWithProfile(p config.Profile) (*api.Client, error) {
	return api.NewClient(&p)
}

// newClientFromFlags creates an API client from the active profile,
// optionally overriding the base URL with --base-url.
func newClientFromFlags() (*api.Client, error) {
	name := activeProfile()
	if name == "" {
		return nil, fmt.Errorf("no profile selected — run `umbraco auth add <name>` first, or use --profile")
	}
	p, err := config.GetProfile(cfgViper, name)
	if err != nil {
		return nil, err
	}
	if baseURLFlag != "" {
		p.BaseURL = baseURLFlag
	}
	return api.NewClient(p)
}
