package skeleton

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/prismgo/installer/internal/run"
)

const prismgoRepository = "https://github.com/prismgo/prismgo"

const prismgoModule = "github.com/prismgo/prismgo"

var stableVersionPattern = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(\+incompatible)?$`)

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
	Runner run.OutputRunner
	// Prompter asks users before falling back to the repository default branch after failures.
	Prompter FallbackPrompter
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

	version, err := s.resolveStableVersion(ctx)
	if err != nil {
		if ok, promptErr := s.confirmFallback(ctx, "latest stable PrismGo skeleton version could not be resolved", err); promptErr != nil || !ok {
			return err
		}
		if fallbackErr := s.cloneDefaultBranch(ctx, tmp); fallbackErr != nil {
			return fallbackWithPreviousError(fallbackErr, err)
		}
		return copyTree(ctx, tmp, target)
	}
	if version == "" {
		if err := s.cloneDefaultBranch(ctx, tmp); err != nil {
			return err
		}
		return copyTree(ctx, tmp, target)
	}
	if err := s.cloneStableVersion(ctx, tmp, version); err != nil {
		if ok, promptErr := s.confirmFallback(ctx, fmt.Sprintf("stable PrismGo skeleton version %s could not be cloned", version), err); promptErr != nil || !ok {
			return err
		}
		if resetErr := resetTemporaryDirectory(tmp); resetErr != nil {
			return fallbackWithPreviousError(resetErr, err)
		}
		if fallbackErr := s.cloneDefaultBranch(ctx, tmp); fallbackErr != nil {
			return fallbackWithPreviousError(fallbackErr, err)
		}
	}

	return copyTree(ctx, tmp, target)
}

func (s GitHubSource) resolveStableVersion(ctx context.Context) (string, error) {
	output, err := s.Runner.Output(ctx, run.Command{
		Name: "go",
		Args: []string{"list", "-m", "-f", "{{.Version}}", prismgoModule + "@latest"},
	})
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(string(output))
	if !stableVersionPattern.MatchString(version) {
		return "", nil
	}
	return version, nil
}

func (s GitHubSource) cloneStableVersion(ctx context.Context, target string, version string) error {
	return s.Runner.Run(ctx, run.Command{
		Name: "git",
		Args: []string{"clone", "--branch", version, "--depth=1", prismgoRepository, target},
	})
}

func (s GitHubSource) cloneDefaultBranch(ctx context.Context, target string) error {
	return s.Runner.Run(ctx, run.Command{
		Name: "git",
		Args: []string{"clone", "--depth=1", prismgoRepository, target},
	})
}

func (s GitHubSource) confirmFallback(ctx context.Context, reason string, cause error) (bool, error) {
	if s.Prompter == nil {
		return false, nil
	}
	return s.Prompter.ConfirmFallback(ctx, reason, cause)
}

func fallbackWithPreviousError(fallbackErr error, previousErr error) error {
	return errors.Join(fallbackErr, fmt.Errorf("previous skeleton acquisition failure: %w", previousErr))
}

func resetTemporaryDirectory(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("remove failed skeleton checkout %q: %w", path, err)
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		return fmt.Errorf("recreate skeleton checkout directory %q: %w", path, err)
	}
	return nil
}

// FallbackPrompter asks whether skeleton acquisition may use the repository default branch.
type FallbackPrompter interface {
	ConfirmFallback(ctx context.Context, reason string, cause error) (bool, error)
}

// TerminalPrompter reads fallback confirmation from an interactive terminal stream.
type TerminalPrompter struct {
	Input  io.Reader
	Output io.Writer
}

// ConfirmFallback asks for explicit user consent before default-branch fallback.
func (p TerminalPrompter) ConfirmFallback(ctx context.Context, reason string, cause error) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if p.Input == nil {
		return false, errors.New("fallback prompt input is required")
	}
	if p.Output == nil {
		return false, errors.New("fallback prompt output is required")
	}
	if _, err := fmt.Fprintf(p.Output, "%s: %v\nFallback to the default branch? [y/N] ", reason, cause); err != nil {
		return false, err
	}
	scanner := bufio.NewScanner(p.Input)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, err
		}
		return false, io.EOF
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes", nil
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
	if err := ensureTargetDirectory(target, mode); err != nil {
		return err
	}

	entries, err := os.ReadDir(source)
	if err != nil {
		return fmt.Errorf("read skeleton directory %q: %w", source, err)
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if shouldSkipSkeletonDirectory(entry.Name(), entry.IsDir()) {
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
		if err := copyFile(ctx, sourcePath, targetPath, info.Mode().Perm()); err != nil {
			return err
		}
	}
	if err := os.Chmod(target, mode); err != nil {
		return fmt.Errorf("chmod target directory %q: %w", target, err)
	}
	return nil
}

func shouldSkipSkeletonDirectory(name string, isDir bool) bool {
	// Repository metadata belongs to the template source and should not become part of generated apps.
	return isDir && (name == ".git" || name == ".github")
}

func ensureTargetDirectory(target string, mode os.FileMode) error {
	info, err := os.Lstat(target)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("target directory already exists as symlink %q", target)
		}
		if !info.IsDir() {
			return fmt.Errorf("target directory already exists as non-directory %q", target)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("inspect target directory %q: %w", target, err)
	}
	if err := os.MkdirAll(target, mode); err != nil {
		return fmt.Errorf("create target directory %q: %w", target, err)
	}
	return nil
}

func copyFile(ctx context.Context, source string, target string, mode os.FileMode) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := os.Lstat(target); err == nil {
		return fmt.Errorf("target file already exists %q", target)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect target file %q: %w", target, err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create parent directory for %q: %w", target, err)
	}

	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open skeleton file %q: %w", source, err)
	}

	// O_EXCL keeps destination creation atomic after the Lstat preflight rejects existing entries and symlinks.
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		closeErr := closeFile(input, fmt.Sprintf("close skeleton file %q", source))
		return errors.Join(fmt.Errorf("create target file %q: %w", target, err), closeErr)
	}

	if _, err := io.Copy(output, input); err != nil {
		copyErr := fmt.Errorf("copy skeleton file %q to %q: %w", source, target, err)
		closeInputErr := closeFile(input, fmt.Sprintf("close skeleton file %q", source))
		closeOutputErr := closeFile(output, fmt.Sprintf("close target file %q", target))
		return errors.Join(copyErr, closeInputErr, closeOutputErr)
	}
	if err := closeFile(input, fmt.Sprintf("close skeleton file %q", source)); err != nil {
		closeOutputErr := closeFile(output, fmt.Sprintf("close target file %q", target))
		return errors.Join(err, closeOutputErr)
	}
	if err := output.Chmod(mode); err != nil {
		closeErr := closeFile(output, fmt.Sprintf("close target file %q", target))
		return errors.Join(fmt.Errorf("chmod target file %q: %w", target, err), closeErr)
	}
	return closeFile(output, fmt.Sprintf("close target file %q", target))
}

func closeFile(file *os.File, context string) error {
	// Close can surface delayed write errors, so callers preserve it instead of relying on deferred cleanup.
	if err := file.Close(); err != nil {
		return fmt.Errorf("%s: %w", context, err)
	}
	return nil
}
