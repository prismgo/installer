package project

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Options contains the user-supplied project arguments and directory safety policy.
type Options struct {
	// CWD is the base directory used to resolve relative project targets; empty means os.Getwd.
	CWD string
	// Name is the required project argument from `prismgo new <name>`.
	Name string
	// Module is an optional go.mod module override from --module.
	Module string
	// Force permits using an existing target directory only when that directory is empty.
	Force bool
}

// Plan describes the safe target directory and module selected before project creation starts.
type Plan struct {
	// Name is the local application directory name.
	Name string
	// Directory is the absolute filesystem path where the project will be created.
	Directory string
	// Module is the go.mod module path to write into the generated application.
	Module string
}

// Resolve parses the project argument, resolves the target directory, and validates that it is safe to use.
func Resolve(opts Options) (Plan, error) {
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		return Plan{}, errors.New("project name is required")
	}

	cwd, err := resolveCWD(opts.CWD)
	if err != nil {
		return Plan{}, err
	}

	directoryName, module, err := resolveNameAndModule(name, strings.TrimSpace(opts.Module))
	if err != nil {
		return Plan{}, err
	}

	target, err := resolveTarget(cwd, directoryName)
	if err != nil {
		return Plan{}, err
	}
	if err := validateTarget(cwd, target, opts.Force); err != nil {
		return Plan{}, err
	}

	return Plan{
		Name:      filepath.Base(directoryName),
		Directory: target,
		Module:    module,
	}, nil
}

func resolveCWD(cwd string) (string, error) {
	// Resolve relative caller-supplied CWD values once so all target checks compare absolute paths.
	if strings.TrimSpace(cwd) == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get current working directory: %w", err)
		}
		cwd = wd
	}
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("resolve current working directory %q: %w", cwd, err)
	}
	return filepath.Clean(abs), nil
}

func resolveNameAndModule(name string, explicitModule string) (string, string, error) {
	// Module-path input keeps the full module path but uses its final segment as the local directory.
	if hasUnsafePathSegment(name) {
		return "", "", fmt.Errorf("project name %q contains unsafe path segment", name)
	}

	directoryName := name
	module := name
	if isModulePath(name) {
		directoryName = pathBase(name)
	}
	if explicitModule != "" {
		module = explicitModule
	}
	if directoryName == "" || directoryName == "." {
		return "", "", fmt.Errorf("project name %q does not include a directory name", name)
	}
	return directoryName, module, nil
}

func isModulePath(name string) bool {
	// A dotted host before the final slash is the project plan's signal for module-path input.
	lastSlash := strings.LastIndex(name, "/")
	if lastSlash < 0 {
		return false
	}
	return strings.Contains(name[:lastSlash], ".")
}

func pathBase(name string) string {
	// filepath.Base is OS-specific; module paths always use slash separators.
	parts := strings.Split(name, "/")
	return parts[len(parts)-1]
}

func hasUnsafePathSegment(name string) bool {
	// Reject absolute paths, empty path components, current-directory, and parent traversal segments.
	if filepath.IsAbs(name) || strings.HasPrefix(name, "/") {
		return true
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." {
			return true
		}
	}
	return false
}

func resolveTarget(cwd string, directoryName string) (string, error) {
	target := filepath.Clean(filepath.Join(cwd, filepath.FromSlash(directoryName)))
	rel, err := filepath.Rel(cwd, target)
	if err != nil {
		return "", fmt.Errorf("resolve target directory %q: %w", directoryName, err)
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("target directory %q escapes current working directory", directoryName)
	}
	return target, nil
}

func validateTarget(cwd string, target string, force bool) error {
	// Missing targets are safe for later creation; existing targets must pass the --force rules below.
	if err := rejectSymlinkedParentComponents(cwd, target); err != nil {
		return err
	}

	info, err := os.Lstat(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("inspect target directory %q: %w", target, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("target path %q is a symlink", target)
	}
	if !info.IsDir() {
		return fmt.Errorf("target path %q already exists and is not a directory", target)
	}
	if !force {
		return fmt.Errorf("target directory %q already exists; use --force to reuse an empty directory", target)
	}

	entries, err := os.ReadDir(target)
	if err != nil {
		return fmt.Errorf("read target directory %q: %w", target, err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("target directory %q is not empty", target)
	}
	return nil
}

func rejectSymlinkedParentComponents(cwd string, target string) error {
	rel, err := filepath.Rel(cwd, target)
	if err != nil {
		return fmt.Errorf("resolve target directory %q: %w", target, err)
	}
	if rel == "." {
		return nil
	}

	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) == 1 {
		return nil
	}

	current := cwd
	for _, part := range parts[:len(parts)-1] {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return fmt.Errorf("inspect target path component %q: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("target path component %q is a symlink", current)
		}
	}
	return nil
}
