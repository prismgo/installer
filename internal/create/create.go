package create

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/prismgo/installer/internal/project"
	"github.com/prismgo/installer/internal/rewrite"
	"github.com/prismgo/installer/internal/run"
	"github.com/prismgo/installer/internal/skeleton"
)

// Options describes one PrismGo application creation request after CLI project resolution.
type Options struct {
	// Project contains the target directory and final module path for the generated application.
	Project project.Plan
	// NoInstall skips dependency resolution and project test execution.
	NoInstall bool
	// Git initializes a local repository in the generated application.
	Git bool
	// Branch sets the initial git branch name; empty defaults to main.
	Branch string
}

// Service orchestrates skeleton acquisition, metadata rewriting, environment initialization, and setup commands.
type Service struct {
	// Skeleton provides the application skeleton to copy into the project directory.
	Skeleton skeleton.Source
	// Runner executes setup commands in the generated project directory.
	Runner run.Runner
	// Output receives user-facing progress lines for each creation step.
	Output io.Writer
}

// Create generates a PrismGo application from the configured skeleton source.
func (s Service) Create(ctx context.Context, opts Options) error {
	if s.Skeleton == nil {
		return errors.New("create skeleton source is required")
	}
	if (!opts.NoInstall || opts.Git) && s.Runner == nil {
		return errors.New("create command runner is required")
	}
	branch := opts.Branch
	if opts.Git {
		var err error
		branch, err = normalizeBranch(branch)
		if err != nil {
			return err
		}
	}

	target := opts.Project.Directory
	if err := s.progress("Creating project directory..."); err != nil {
		return err
	}
	if err := s.Skeleton.CopyTo(ctx, target); err != nil {
		return err
	}

	goModPath := filepath.Join(target, "go.mod")
	oldModule, err := rewrite.ReadModule(goModPath)
	if err != nil {
		return err
	}
	if err := s.progress("Updating module path..."); err != nil {
		return err
	}
	if err := rewrite.RewriteModule(goModPath, opts.Project.Module); err != nil {
		return err
	}
	if err := s.progress("Updating imports..."); err != nil {
		return err
	}
	if err := rewrite.RewriteImports(target, oldModule, opts.Project.Module); err != nil {
		return err
	}
	if err := s.progress("Creating environment file..."); err != nil {
		return err
	}
	if err := rewrite.InitEnv(target); err != nil {
		return err
	}
	if opts.NoInstall {
		if opts.Git {
			if err := s.initGit(ctx, target, branch); err != nil {
				return err
			}
			return s.complete(target)
		}
		return s.complete(target)
	}

	// Drop the skeleton's placeholder framework requirement before resolving @latest.
	if err := s.progress("Preparing framework dependency..."); err != nil {
		return err
	}
	if err := s.Runner.Run(ctx, run.Command{Name: "go", Args: []string{"mod", "edit", "-droprequire", "github.com/prismgo/framework"}, Dir: target}); err != nil {
		return err
	}
	if err := s.progress("Installing framework dependency..."); err != nil {
		return err
	}
	if err := s.Runner.Run(ctx, run.Command{Name: "go", Args: []string{"get", "github.com/prismgo/framework@latest"}, Dir: target}); err != nil {
		return err
	}
	if err := s.progress("Tidying dependencies..."); err != nil {
		return err
	}
	if err := s.Runner.Run(ctx, run.Command{Name: "go", Args: []string{"mod", "tidy"}, Dir: target}); err != nil {
		return err
	}
	if err := s.progress("Testing generated project..."); err != nil {
		return err
	}
	if err := s.Runner.Run(ctx, run.Command{Name: "go", Args: []string{"test", "./..."}, Dir: target}); err != nil {
		return err
	}
	if !opts.Git {
		return s.complete(target)
	}
	if err := s.initGit(ctx, target, branch); err != nil {
		return err
	}
	return s.complete(target)
}

func (s Service) initGit(ctx context.Context, target string, branch string) error {
	// Keep the commands explicit so callers and tests can observe the same git lifecycle users expect.
	commands := []run.Command{
		{Name: "git", Args: []string{"init"}, Dir: target},
		{Name: "git", Args: []string{"add", "."}, Dir: target},
		{Name: "git", Args: []string{"-c", "user.name=PrismGo Installer", "-c", "user.email=installer@prismgo.dev", "commit", "-m", "Set up a fresh PrismGo app"}, Dir: target},
		{Name: "git", Args: []string{"branch", "-M", "--", branch}, Dir: target},
	}
	for _, command := range commands {
		if err := s.progress("Running " + command.Name + " " + strings.Join(command.Args, " ") + "..."); err != nil {
			return err
		}
		if err := s.Runner.Run(ctx, command); err != nil {
			return err
		}
	}
	return nil
}

func (s Service) progress(message string) error {
	if s.Output == nil {
		return nil
	}
	_, err := fmt.Fprintln(s.Output, message)
	return err
}

func (s Service) complete(target string) error {
	return s.progress("Project created successfully: " + target)
}

func normalizeBranch(branch string) (string, error) {
	if branch == "" {
		return "main", nil
	}
	if strings.HasPrefix(branch, "-") {
		return "", errors.New("git branch name must not start with '-'")
	}
	if strings.HasPrefix(branch, "/") || strings.HasSuffix(branch, "/") || strings.Contains(branch, "//") {
		return "", errors.New("git branch name contains invalid slash placement")
	}
	if branch == "@" || branch == "HEAD" || strings.HasSuffix(branch, ".") || strings.HasSuffix(branch, ".lock") || strings.Contains(branch, "..") || strings.Contains(branch, "@{") {
		return "", errors.New("git branch name contains invalid sequence")
	}
	for _, component := range strings.Split(branch, "/") {
		if strings.HasPrefix(component, ".") || strings.HasSuffix(component, ".lock") {
			return "", errors.New("git branch name contains invalid path component")
		}
	}
	for _, r := range branch {
		if r <= ' ' || strings.ContainsRune(`~^:?*[\`, r) {
			return "", errors.New("git branch name contains invalid character")
		}
	}
	return branch, nil
}
