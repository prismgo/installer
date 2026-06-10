package rewrite

import (
	"fmt"
	"os"
	"path/filepath"
)

func safeReplaceFile(path string, content []byte) error {
	return safeReplaceFileWithHook(path, content, nil)
}

func safeReplaceFileWithHook(path string, content []byte, beforeFinalCheck func() error) error {
	before, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect %q: %w", path, err)
	}
	if before.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%q is a symlink", path)
	}
	if !before.Mode().IsRegular() {
		return fmt.Errorf("%q is not a regular file", path)
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary file for %q: %w", path, err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary file for %q: %w", path, err)
	}
	if err := tmp.Chmod(before.Mode().Perm()); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temporary file for %q: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary file for %q: %w", path, err)
	}

	if beforeFinalCheck != nil {
		if err := beforeFinalCheck(); err != nil {
			return err
		}
	}

	after, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect %q before replace: %w", path, err)
	}
	if after.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%q is a symlink", path)
	}
	if !after.Mode().IsRegular() {
		return fmt.Errorf("%q is not a regular file", path)
	}
	if !os.SameFile(before, after) {
		return fmt.Errorf("%q changed while preparing rewrite", path)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace %q: %w", path, err)
	}
	return nil
}
