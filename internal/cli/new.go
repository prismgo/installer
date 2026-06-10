package cli

import (
	"context"
	"errors"

	"github.com/prismgo/installer/internal/create"
	"github.com/prismgo/installer/internal/project"
	"github.com/prismgo/installer/internal/run"
	"github.com/prismgo/installer/internal/skeleton"
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

type creator interface {
	Create(context.Context, create.Options) error
}

// NewCommand defines the `prismgo new` surface and delegates creation to the create service.
func NewCommand() *cobra.Command {
	runner := run.OSRunner{}
	return newCommandWithCreator(create.Service{
		Skeleton: skeleton.GitHubSource{Runner: runner},
		Runner:   runner,
	})
}

func newCommandWithCreator(creator creator) *cobra.Command {
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
			// Resolve validates the target before the create service writes skeleton files.
			plan, err := project.Resolve(project.Options{
				Name:   args[0],
				Module: opts.module,
				Force:  opts.force,
			})
			if err != nil {
				return err
			}
			return creator.Create(cmd.Context(), create.Options{
				Project:   plan,
				NoInstall: opts.noInstall,
				Git:       opts.git,
				Branch:    opts.branch,
			})
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
