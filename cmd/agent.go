package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// skillContent holds the embedded SKILL.md content, set by main() via SetSkillContent.
var skillContent string

// SetSkillContent is called from main() to inject the embedded skill file content.
func SetSkillContent(s string) { skillContent = s }

// skillTarget describes where a particular tool expects the skill file.
type skillTarget struct {
	tool string
	dir  func() string // returns the absolute target directory
}

var skillTargets = []skillTarget{
	{
		tool: "opencode",
		dir: func() string {
			home, _ := os.UserHomeDir()
			return filepath.Join(home, ".opencode", "skill", "umbraco-cli")
		},
	},
	{
		tool: "claudecode",
		dir: func() string {
			home, _ := os.UserHomeDir()
			return filepath.Join(home, ".claude", "skills", "umbraco-cli")
		},
	},
}

// resolveTargets returns the targets matching the --tool flag value.
func resolveTargets(tool string) ([]skillTarget, error) {
	if tool == "all" || tool == "" {
		return skillTargets, nil
	}
	for _, t := range skillTargets {
		if t.tool == tool {
			return []skillTarget{t}, nil
		}
	}
	return nil, fmt.Errorf("unknown tool %q — valid values: opencode, claudecode, all", tool)
}

// ── commands ─────────────────────────────────────────────────────────────────

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage AI agent skill file installation",
}

var agentInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the umbraco-cli skill into AI agent tool directories",
	Long: `Copies the embedded skill/SKILL.md into the global skill directory of
the specified AI tool(s) so they automatically load Umbraco CLI knowledge.

Supported tools: opencode, claudecode (default: all)

The command aborts if the skill is already installed. Run 'umbraco agent status'
to check current state, or 'umbraco agent uninstall' to remove first.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tool, _ := cmd.Flags().GetString("tool")
		targets, err := resolveTargets(tool)
		if err != nil {
			return err
		}

		anyFailed := false
		for _, t := range targets {
			dir := t.dir()
			dest := filepath.Join(dir, "SKILL.md")

			if _, err := os.Stat(dest); err == nil {
				fmt.Fprintf(os.Stderr, "error [%s]: skill already installed at %s\n  Run 'umbraco agent uninstall --tool %s' to remove it first.\n", t.tool, dest, t.tool)
				anyFailed = true
				continue
			}

			if err := os.MkdirAll(dir, 0o755); err != nil {
				fmt.Fprintf(os.Stderr, "error [%s]: cannot create directory %s: %v\n", t.tool, dir, err)
				anyFailed = true
				continue
			}

			if err := os.WriteFile(dest, []byte(skillContent), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "error [%s]: cannot write %s: %v\n", t.tool, dest, err)
				anyFailed = true
				continue
			}

			fmt.Printf("installed [%s]: %s\n", t.tool, dest)
		}

		if anyFailed {
			return fmt.Errorf("one or more installations failed")
		}
		return nil
	},
}

var agentUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the umbraco-cli skill from AI agent tool directories",
	RunE: func(cmd *cobra.Command, args []string) error {
		tool, _ := cmd.Flags().GetString("tool")
		targets, err := resolveTargets(tool)
		if err != nil {
			return err
		}

		for _, t := range targets {
			dir := t.dir()
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				fmt.Printf("not installed [%s]\n", t.tool)
				continue
			}
			if err := os.RemoveAll(dir); err != nil {
				return fmt.Errorf("[%s]: cannot remove %s: %w", t.tool, dir, err)
			}
			fmt.Printf("removed [%s]: %s\n", t.tool, dir)
		}
		return nil
	},
}

var agentStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show whether the umbraco-cli skill is installed for each AI tool",
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, t := range skillTargets {
			dest := filepath.Join(t.dir(), "SKILL.md")
			if _, err := os.Stat(dest); err == nil {
				fmt.Printf("%-12s installed    %s\n", t.tool, dest)
			} else {
				fmt.Printf("%-12s not installed\n", t.tool)
			}
		}
		return nil
	},
}

func init() {
	agentInstallCmd.Flags().String("tool", "all", "Tool to install for: opencode, claudecode, or all")
	agentUninstallCmd.Flags().String("tool", "all", "Tool to uninstall for: opencode, claudecode, or all")

	agentCmd.AddCommand(agentInstallCmd)
	agentCmd.AddCommand(agentUninstallCmd)
	agentCmd.AddCommand(agentStatusCmd)
}
