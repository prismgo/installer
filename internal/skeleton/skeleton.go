package skeleton

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/prismgo/installer/internal/run"
)

const prismgoRepository = "https://github.com/prismgo/prismgo"

// Source acquires a PrismGo skeleton into target.
type Source interface {
	CopyTo(ctx context.Context, target string) error
}

// LocalSource copies a skeleton from a local directory, mainly for tests and offline fixtures.
type LocalSource struct {
	// Dir is the root directory that contains the skeleton files.
	Dir string
}

// CopyTo recursively copies the local skeleton into target while excluding VCS metadata.
func (s LocalSource) CopyTo(ctx context.Context, target string) error {
	if s.Dir == "" {
		return errors.New("skeleton source directory is required")
	}
	return copyTree(ctx, s.Dir, target)
}

// GitHubSource clones the official PrismGo repository and copies its contents into target.
type GitHubSource struct {
	// Runner executes the git clone command.
	Runner run.Runner
}

// CopyTo clones the official PrismGo repository into a temporary directory, then copies it to target.
func (s GitHubSource) CopyTo(ctx context.Context, target string) error {
	if s.Runner == nil {
		return errors.New("skeleton command runner is required")
	}

	tmp, err := os.MkdirTemp("", "prismgo-skeleton-*")
	if err != nil {
		return fmt.Errorf("create temporary skeleton directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(tmp)
	}()

	if err := s.Runner.Run(ctx, run.Command{
		Name: "git",
		Args: []string{"clone", "--depth=1", prismgoRepository, tmp},
	}); err != nil {
		return err
	}
	return copyTree(ctx, tmp, target)
}

func copyTree(ctx context.Context, source string, target string) error {
	info, err := os.Lstat(source)
	if err != nil {
		return fmt.Errorf("inspect skeleton source %q: %w", source, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("skeleton source %q is a symlink", source)
	}
	if !info.IsDir() {
		return fmt.Errorf("skeleton source %q is not a directory", source)
	}
	if target == "" {
		return errors.New("skeleton target directory is required")
	}
	return copyDirectory(ctx, source, target, info.Mode().Perm())
}

func copyDirectory(ctx context.Context, source string, target string, mode os.FileMode) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(target, mode); err != nil {
		return fmt.Errorf("create target directory %q: %w", target, err)
	}

	entries, err := os.ReadDir(source)
	if err != nil {
		return fmt.Errorf("read skeleton directory %q: %w", source, err)
	}
	for _, entry := range entries {
		if entry.Name() == ".git" && entry.IsDir() {
			continue
		}

		sourcePath := filepath.Join(source, entry.Name())
		targetPath := filepath.Join(target, entry.Name())
		info, err := os.Lstat(sourcePath)
		if err != nil {
			return fmt.Errorf("inspect skeleton entry %q: %w", sourcePath, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsupported symlink in skeleton %q", sourcePath)
		}
		if info.IsDir() {
			if err := copyDirectory(ctx, sourcePath, targetPath, info.Mode().Perm()); err != nil {
				return err
			}
			continue
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported non-regular skeleton file %q", sourcePath)
		}
		if err := copyFile(sourcePath, targetPath, info.Mode().Perm()); err != nil {
			return err
		}
	}
	if err := os.Chmod(target, mode); err != nil {
		return fmt.Errorf("chmod target directory %q: %w", target, err)
	}
	return nil
}

func copyFile(source string, target string, mode os.FileMode) error {
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open skeleton file %q: %w", source, err)
	}
	defer func() {
		_ = input.Close()
	}()

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create parent directory for %q: %w", target, err)
	}
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("create target file %q: %w", target, err)
	}
	defer func() {
		_ = output.Close()
	}()

	if _, err := io.Copy(output, input); err != nil {
		return fmt.Errorf("copy skeleton file %q to %q: %w", source, target, err)
	}
	if err := output.Chmod(mode); err != nil {
		return fmt.Errorf("chmod target file %q: %w", target, err)
	}
	return nil
}
