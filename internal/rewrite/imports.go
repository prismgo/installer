package rewrite

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// RewriteImports rewrites Go import paths that belong to oldModule under root to newModule.
func RewriteImports(root string, oldModule string, newModule string) error {
	if strings.TrimSpace(oldModule) == "" {
		return fmt.Errorf("rewrite imports under %q: old module cannot be empty", root)
	}
	if strings.TrimSpace(newModule) == "" {
		return fmt.Errorf("rewrite imports under %q: new module cannot be empty", root)
	}

	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk %q: %w", path, walkErr)
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("rewrite imports %q: symlinked paths are not supported", path)
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		if err := rewriteGoFileImports(path, oldModule, newModule); err != nil {
			return err
		}
		return nil
	})
}

func rewriteGoFileImports(path string, oldModule string, newModule string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect Go file %q: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("rewrite imports %q: symlinked Go files are not supported", path)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse Go file %q: %w", path, err)
	}

	changed := false
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return fmt.Errorf("unquote import path in %q: %w", path, err)
		}
		rewritten, ok := rewriteImportPath(importPath, oldModule, newModule)
		if !ok {
			continue
		}
		// Only the string literal changes; aliases, dot imports, blanks, and comments remain on the ImportSpec.
		spec.Path.Value = strconv.Quote(rewritten)
		changed = true
	}
	if !changed {
		return nil
	}

	var formatted bytes.Buffer
	if err := format.Node(&formatted, fset, file); err != nil {
		return fmt.Errorf("format Go file %q: %w", path, err)
	}
	info, err = os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect Go file %q before write: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("rewrite imports %q: symlinked Go files are not supported", path)
	}
	if err := safeReplaceFile(path, formatted.Bytes()); err != nil {
		return fmt.Errorf("write Go file %q: %w", path, err)
	}
	return nil
}

func rewriteImportPath(importPath string, oldModule string, newModule string) (string, bool) {
	// Match only the exact old module or its descendants; external github.com/prismgo/... paths are unrelated.
	if importPath == oldModule {
		return newModule, true
	}
	prefix := oldModule + "/"
	if strings.HasPrefix(importPath, prefix) {
		return newModule + strings.TrimPrefix(importPath, oldModule), true
	}
	return importPath, false
}
