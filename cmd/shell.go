package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const shellIntegrationMarker = "# Kubconfig shell integration"
const shellIntegrationScript = `
kubconfig() {
	if [ "$1" = "activate" ]; then
		eval "$(command kubconfig activate "$2" --session "$3" --shell-eval)"
	elif [ "$1" = "deactivate" ]; then
		eval "$(command kubconfig deactivate --shell-eval)"
	else
		command kubconfig "$@"
	fi
}
`

var ShellCmd = &cobra.Command{
	Use:   "shell [install|uninstall]",
	Short: "Manage shell integration",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		action := args[0]
		shells := []string{
			filepath.Join(os.Getenv("HOME"), ".bashrc"),
			filepath.Join(os.Getenv("HOME"), ".zshrc"),
		}

		switch action {
		case "install":
			for _, shellRc := range shells {
				if err := installShellIntegration(shellRc); err != nil {
					fmt.Printf("Error installing to %s: %v\n", shellRc, err)
				} else {
					fmt.Printf("Installed shell integration to %s\n", shellRc)
				}
			}
			fmt.Println("\nPlease restart your shell or run:")
			fmt.Println("source ~/.bashrc  # for bash")
			fmt.Println("source ~/.zshrc   # for zsh")

		case "uninstall":
			for _, shellRc := range shells {
				if err := removeShellIntegration(shellRc); err != nil {
					fmt.Printf("Error removing from %s: %v\n", shellRc, err)
				} else {
					fmt.Printf("Removed shell integration from %s\n", shellRc)
				}
			}
		default:
			fmt.Println("Invalid action. Use 'install' or 'uninstall'")
		}
	},
}

func installShellIntegration(rcFile string) error {
	// Check if already installed
	if isIntegrationInstalled(rcFile) {
		return nil
	}

	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Add integration script
	_, err = f.WriteString(fmt.Sprintf("\n%s\n%s\n", shellIntegrationMarker, shellIntegrationScript))
	return err
}

func removeShellIntegration(rcFile string) error {
	if !isIntegrationInstalled(rcFile) {
		return nil
	}

	// Read current content
	content, err := os.ReadFile(rcFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	removing := false

	// Remove integration block
	for _, line := range lines {
		if strings.TrimSpace(line) == shellIntegrationMarker {
			removing = true
			continue
		}
		if removing && strings.TrimSpace(line) == "" {
			removing = false
			continue
		}
		if !removing {
			newLines = append(newLines, line)
		}
	}

	// Write back
	return os.WriteFile(rcFile, []byte(strings.Join(newLines, "\n")), 0644)
}

func isIntegrationInstalled(rcFile string) bool {
	content, err := os.ReadFile(rcFile)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), shellIntegrationMarker)
}
