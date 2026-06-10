package rewrite

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// InitEnv copies .env.example to .env when an example exists and no .env file is present.
func InitEnv(root string) error {
	examplePath := filepath.Join(root, ".env.example")
	envPath := filepath.Join(root, ".env")

	info, err := os.Lstat(examplePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("inspect env example %q: %w", examplePath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("env example %q is a symlink", examplePath)
	}
	if info.IsDir() {
		return fmt.Errorf("env example %q is a directory", examplePath)
	}

	if envInfo, err := os.Lstat(envPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect env file %q: %w", envPath, err)
		}
	} else {
		if envInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("env file %q is a symlink", envPath)
		}
		// Existing .env content belongs to the user or skeleton and must not be overwritten.
		return nil
	}

	content, openedInfo, err := safeReadRegularFile(examplePath)
	if err != nil {
		return fmt.Errorf("read env example %q: %w", examplePath, err)
	}
	if !os.SameFile(info, openedInfo) {
		return fmt.Errorf("env example %q changed while preparing read", examplePath)
	}
	if err := writeNewFile(envPath, content, openedInfo.Mode().Perm()); err != nil {
		return fmt.Errorf("write env file %q: %w", envPath, err)
	}
	return nil
}

func writeNewFile(path string, content []byte, perm os.FileMode) error {
	// O_EXCL avoids replacing a path that appears after the Lstat check.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}

	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
