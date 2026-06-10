package cli

import (
	"context"

	"github.com/spf13/cobra"
)

// Execute builds the root command for each invocation so tests and callers get isolated flag state.
func Execute(ctx context.Context, args []string) error {
	cmd := NewRootCommand()
	cmd.SetArgs(args)
	return cmd.ExecuteContext(ctx)
}

// NewRootCommand wires the installer command tree while keeping command behavior in child commands.
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "prismgo",
		Short:         "PrismGo installer",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(NewCommand())
	return cmd
}
