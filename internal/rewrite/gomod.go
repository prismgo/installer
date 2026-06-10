package rewrite

import (
	"fmt"
	"strings"
)

// ReadModule reads the module path from the module directive in a go.mod file.
func ReadModule(path string) (string, error) {
	content, _, err := safeReadRegularFile(path)
	if err != nil {
		return "", fmt.Errorf("read go.mod %q: %w", path, err)
	}

	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("go.mod %q does not contain a module directive", path)
}

// RewriteModule changes only the module directive line and leaves the rest of go.mod unchanged.
func RewriteModule(path string, module string) error {
	content, info, err := safeReadRegularFile(path)
	if err != nil {
		return fmt.Errorf("read go.mod %q: %w", path, err)
	}
	if strings.TrimSpace(module) == "" {
		return fmt.Errorf("rewrite go.mod %q: module cannot be empty", path)
	}

	lines := strings.SplitAfter(string(content), "\n")
	for i, line := range lines {
		text := strings.TrimRight(line, "\r\n")
		fields := strings.Fields(text)
		if len(fields) >= 2 && fields[0] == "module" {
			// Preserve the original line ending while replacing only the directive payload.
			lineEnding := strings.TrimPrefix(line, text)
			lines[i] = "module " + module + lineEnding
			if err := safeReplaceFile(path, []byte(strings.Join(lines, "")), info); err != nil {
				return fmt.Errorf("write go.mod %q: %w", path, err)
			}
			return nil
		}
	}
	return fmt.Errorf("go.mod %q does not contain a module directive", path)
}
