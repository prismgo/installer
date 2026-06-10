package cli

import (
	"errors"
	"fmt"

	"github.com/prismgo/installer/internal/project"
	"github.com/spf13/cobra"
)

// newOptions stores parsed CLI flags until later tasks connect project creation behavior.
type newOptions struct {
	module    string
	noInstall bool
	git       bool
	branch    string
	force     bool
	github    bool
}

// NewCommand defines the initial `prismgo new` surface before project creation is implemented.
func NewCommand() *cobra.Command {
	opts := newOptions{}

	cmd := &cobra.Command{
		Use:   "new [name]",
		Short: "Create a new PrismGo application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// --github is parsed now so users get a deliberate unsupported error instead of a generic placeholder.
			if opts.github {
				return errors.New("github repository creation is not supported yet")
			}
			// Resolve validates the target before later tasks perform any filesystem creation.
			plan, err := project.Resolve(project.Options{
				Name:   args[0],
				Module: opts.module,
				Force:  opts.force,
			})
			if err != nil {
				return err
			}
			return fmt.Errorf("prismgo new %q is not implemented yet", plan.Name)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.module, "module", "", "module path for the generated application")
	flags.BoolVar(&opts.noInstall, "no-install", false, "skip dependency installation and verification")
	flags.BoolVar(&opts.git, "git", false, "initialize a git repository")
	flags.StringVar(&opts.branch, "branch", "main", "initial git branch name")
	flags.BoolVar(&opts.force, "force", false, "allow using an existing empty target directory")
	flags.BoolVar(&opts.github, "github", false, "create a GitHub repository")

	return cmd
}
