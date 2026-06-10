package rewrite

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func safeReadRegularFile(path string) ([]byte, os.FileInfo, error) {
	return safeReadRegularFileWithHook(path, nil)
}

func safeReadRegularFileWithHook(path string, beforeOpen func() error) ([]byte, os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, nil, fmt.Errorf("inspect %q: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, nil, fmt.Errorf("%q is a symlink", path)
	}
	if !info.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("%q is not a regular file", path)
	}

	if beforeOpen != nil {
		if err := beforeOpen(); err != nil {
			return nil, nil, err
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open %q: %w", path, err)
	}
	defer func() {
		_ = file.Close()
	}()

	openedInfo, err := file.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("stat %q: %w", path, err)
	}
	if !os.SameFile(info, openedInfo) {
		return nil, nil, fmt.Errorf("%q changed while preparing read", path)
	}

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, nil, fmt.Errorf("read %q: %w", path, err)
	}
	return content, info, nil
}

func safeReplaceFile(path string, content []byte, expected os.FileInfo) error {
	return safeReplaceFileWithHook(path, content, expected, nil)
}

func safeReplaceFileWithHook(path string, content []byte, expected os.FileInfo, beforeFinalCheck func() error) error {
	if expected == nil {
		return fmt.Errorf("%q has no expected file identity", path)
	}
	if expected.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%q is a symlink", path)
	}
	if !expected.Mode().IsRegular() {
		return fmt.Errorf("%q is not a regular file", path)
	}
	if expected.Mode().Perm()&0o222 == 0 {
		return fmt.Errorf("%q is not writable", path)
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
	if err := tmp.Chmod(expected.Mode().Perm()); err != nil {
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
	if !os.SameFile(expected, after) {
		return fmt.Errorf("%q changed while preparing rewrite", path)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace %q: %w", path, err)
	}
	return nil
}
