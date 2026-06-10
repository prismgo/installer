package rewrite

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ReadModule reads the module path from the module directive in a go.mod file.
func ReadModule(path string) (string, error) {
	content, _, err := safeReadRegularFile(path)
	if err != nil {
		return "", fmt.Errorf("read go.mod %q: %w", path, err)
	}

	for _, line := range strings.Split(string(content), "\n") {
		directive, ok, err := parseModuleDirectiveLine(line)
		if err != nil {
			return "", fmt.Errorf("read module directive in go.mod %q: %w", path, err)
		}
		if ok {
			return directive.module, nil
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
		directive, ok, err := parseModuleDirectiveLine(line)
		if err != nil {
			return fmt.Errorf("rewrite module directive in go.mod %q: %w", path, err)
		}
		if ok {
			lines[i] = directive.prefix + module + directive.suffix
			if err := safeReplaceFile(path, []byte(strings.Join(lines, "")), info); err != nil {
				return fmt.Errorf("write go.mod %q: %w", path, err)
			}
			return nil
		}
	}
	return fmt.Errorf("go.mod %q does not contain a module directive", path)
}

type moduleDirective struct {
	module string
	prefix string
	suffix string
}

func parseModuleDirectiveLine(line string) (moduleDirective, bool, error) {
	text, lineEnding := splitLineEnding(line)
	offset := 0
	for offset < len(text) {
		r, size := utf8.DecodeRuneInString(text[offset:])
		if !unicode.IsSpace(r) {
			break
		}
		offset += size
	}
	if !strings.HasPrefix(text[offset:], "module") {
		return moduleDirective{}, false, nil
	}
	afterKeyword := offset + len("module")
	if afterKeyword >= len(text) || !isSpaceByte(text[afterKeyword]) {
		return moduleDirective{}, false, nil
	}
	tokenStart := afterKeyword
	for tokenStart < len(text) && isSpaceByte(text[tokenStart]) {
		tokenStart++
	}
	if tokenStart >= len(text) {
		return moduleDirective{}, true, fmt.Errorf("module path is missing")
	}

	tokenEnd, module, err := parseModuleToken(text, tokenStart)
	if err != nil {
		return moduleDirective{}, true, err
	}
	return moduleDirective{
		module: module,
		prefix: text[:tokenStart],
		suffix: text[tokenEnd:] + lineEnding,
	}, true, nil
}

func parseModuleToken(text string, start int) (int, string, error) {
	switch text[start] {
	case '"':
		for end := start + 1; end < len(text); end++ {
			if text[end] == '\\' {
				end++
				continue
			}
			if text[end] == '"' {
				token := text[start : end+1]
				module, err := strconv.Unquote(token)
				if err != nil {
					return 0, "", err
				}
				return end + 1, module, nil
			}
		}
		return 0, "", fmt.Errorf("quoted module path is not terminated")
	case '`':
		for end := start + 1; end < len(text); end++ {
			if text[end] == '`' {
				token := text[start : end+1]
				module, err := strconv.Unquote(token)
				if err != nil {
					return 0, "", err
				}
				return end + 1, module, nil
			}
		}
		return 0, "", fmt.Errorf("quoted module path is not terminated")
	default:
		end := start
		for end < len(text) && !isSpaceByte(text[end]) && !startsLineComment(text, end) {
			end++
		}
		if end == start {
			return 0, "", fmt.Errorf("module path is missing")
		}
		return end, text[start:end], nil
	}
}

func splitLineEnding(line string) (string, string) {
	if strings.HasSuffix(line, "\r\n") {
		return strings.TrimSuffix(line, "\r\n"), "\r\n"
	}
	if strings.HasSuffix(line, "\n") {
		return strings.TrimSuffix(line, "\n"), "\n"
	}
	if strings.HasSuffix(line, "\r") {
		return strings.TrimSuffix(line, "\r"), "\r"
	}
	return line, ""
}

func isSpaceByte(b byte) bool {
	return b == ' ' || b == '\t'
}

func startsLineComment(text string, offset int) bool {
	return offset+1 < len(text) && text[offset] == '/' && text[offset+1] == '/'
}
