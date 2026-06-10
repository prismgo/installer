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

	info, err := os.Stat(examplePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("inspect env example %q: %w", examplePath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("env example %q is a directory", examplePath)
	}

	if _, err := os.Stat(envPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect env file %q: %w", envPath, err)
		}
	} else {
		// Existing .env content belongs to the user or skeleton and must not be overwritten.
		return nil
	}

	content, err := os.ReadFile(examplePath)
	if err != nil {
		return fmt.Errorf("read env example %q: %w", examplePath, err)
	}
	if err := os.WriteFile(envPath, content, info.Mode().Perm()); err != nil {
		return fmt.Errorf("write env file %q: %w", envPath, err)
	}
	return nil
}
