package create

import (
	"context"
	"errors"
	"path/filepath"

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
	// Git is preserved for Task 6, where repository initialization is implemented.
	Git bool
	// Branch is preserved for Task 6, where repository initialization is implemented.
	Branch string
}

// Service orchestrates skeleton acquisition, metadata rewriting, environment initialization, and setup commands.
type Service struct {
	// Skeleton provides the application skeleton to copy into the project directory.
	Skeleton skeleton.Source
	// Runner executes setup commands in the generated project directory.
	Runner run.Runner
}

// Create generates a PrismGo application from the configured skeleton source.
func (s Service) Create(ctx context.Context, opts Options) error {
	if s.Skeleton == nil {
		return errors.New("create skeleton source is required")
	}
	if !opts.NoInstall && s.Runner == nil {
		return errors.New("create command runner is required")
	}

	target := opts.Project.Directory
	if err := s.Skeleton.CopyTo(ctx, target); err != nil {
		return err
	}

	goModPath := filepath.Join(target, "go.mod")
	oldModule, err := rewrite.ReadModule(goModPath)
	if err != nil {
		return err
	}
	if err := rewrite.RewriteModule(goModPath, opts.Project.Module); err != nil {
		return err
	}
	if err := rewrite.RewriteImports(target, oldModule, opts.Project.Module); err != nil {
		return err
	}
	if err := rewrite.InitEnv(target); err != nil {
		return err
	}
	if opts.NoInstall {
		return nil
	}

	if err := s.Runner.Run(ctx, run.Command{Name: "go", Args: []string{"mod", "tidy"}, Dir: target}); err != nil {
		return err
	}
	return s.Runner.Run(ctx, run.Command{Name: "go", Args: []string{"test", "./..."}, Dir: target})
}
