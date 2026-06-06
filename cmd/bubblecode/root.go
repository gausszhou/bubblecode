package bubblecode

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "bubblecode",
	Short: "TUI for AI agents via the ACP protocol",
	Long: `A terminal user interface for interacting with AI agents via the ACP protocol.
Built with Bubble Tea v2 and Lip Gloss v2.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChat(cmd)
	},
}

func chatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "chat",
		Short: "Start the chat TUI (default)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChat(cmd)
		},
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(chatCmd())
	rootCmd.AddCommand(acpCmd())
	rootCmd.AddCommand(modelsCmd())
	rootCmd.AddCommand(providersCmd())
}
