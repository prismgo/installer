package rewrite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadModuleReadsPrismgoModule(t *testing.T) {
	// ReadModule should return the module path from the go.mod module directive.
	path := writeFile(t, t.TempDir(), "go.mod", "module prismgo\n\ngo 1.26.2\n")

	module, err := ReadModule(path)
	if err != nil {
		t.Fatalf("expected module to be read, got error: %v", err)
	}
	if module != "prismgo" {
		t.Fatalf("expected module %q, got %q", "prismgo", module)
	}
}

func TestReadModuleFailsWhenDirectiveIsMissing(t *testing.T) {
	// A go.mod without a module directive cannot be rewritten safely by later creation steps.
	path := writeFile(t, t.TempDir(), "go.mod", "go 1.26.2\n")

	_, err := ReadModule(path)
	if err == nil {
		t.Fatal("expected missing module directive to fail")
	}
	if !strings.Contains(err.Error(), "module directive") {
		t.Fatalf("expected module directive error, got %q", err.Error())
	}
}

func TestReadModuleFailsWhenFileIsMissing(t *testing.T) {
	// Missing go.mod files should return the underlying read error with path context.
	_, err := ReadModule(filepath.Join(t.TempDir(), "go.mod"))
	if err == nil {
		t.Fatal("expected missing go.mod to fail")
	}
	if !strings.Contains(err.Error(), "read go.mod") {
		t.Fatalf("expected read error, got %q", err.Error())
	}
}

func TestRewriteModuleOnlyChangesModuleDirective(t *testing.T) {
	// RewriteModule should preserve non-module content byte-for-byte while changing only the module line.
	dir := t.TempDir()
	path := writeFile(t, dir, "go.mod", "module prismgo\n\nrequire github.com/prismgo/framework v0.0.0\n")

	if err := RewriteModule(path, "github.com/acme/myapp"); err != nil {
		t.Fatalf("expected module rewrite to pass, got error: %v", err)
	}

	got := readFile(t, path)
	want := "module github.com/acme/myapp\n\nrequire github.com/prismgo/framework v0.0.0\n"
	if got != want {
		t.Fatalf("expected go.mod content:\n%s\ngot:\n%s", want, got)
	}
}

func TestRewriteModuleRejectsEmptyModule(t *testing.T) {
	// Empty module paths would create an invalid go.mod and should fail before modifying the file.
	path := writeFile(t, t.TempDir(), "go.mod", "module prismgo\n")

	err := RewriteModule(path, " ")
	if err == nil {
		t.Fatal("expected empty module rewrite to fail")
	}
	if got := readFile(t, path); got != "module prismgo\n" {
		t.Fatalf("expected original go.mod to remain unchanged, got %q", got)
	}
}

func TestRewriteModuleFailsWhenDirectiveIsMissing(t *testing.T) {
	// RewriteModule should report malformed go.mod files instead of appending a second module line.
	path := writeFile(t, t.TempDir(), "go.mod", "go 1.26.2\n")

	err := RewriteModule(path, "github.com/acme/myapp")
	if err == nil {
		t.Fatal("expected missing module directive to fail")
	}
	if !strings.Contains(err.Error(), "module directive") {
		t.Fatalf("expected module directive error, got %q", err.Error())
	}
}

func TestRewriteModuleFailsWhenFileIsMissing(t *testing.T) {
	// Missing go.mod files should fail before any rewrite is attempted.
	err := RewriteModule(filepath.Join(t.TempDir(), "go.mod"), "github.com/acme/myapp")
	if err == nil {
		t.Fatal("expected missing go.mod to fail")
	}
	if !strings.Contains(err.Error(), "inspect go.mod") {
		t.Fatalf("expected inspect error, got %q", err.Error())
	}
}

func TestRewriteModuleFailsWhenPathIsDirectory(t *testing.T) {
	// A directory named go.mod passes Lstat but must fail when read as a module file.
	path := filepath.Join(t.TempDir(), "go.mod")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("create go.mod directory: %v", err)
	}

	err := RewriteModule(path, "github.com/acme/myapp")
	if err == nil {
		t.Fatal("expected go.mod directory rewrite to fail")
	}
	if !strings.Contains(err.Error(), "read go.mod") {
		t.Fatalf("expected read error, got %q", err.Error())
	}
}

func TestRewriteModuleReturnsWriteError(t *testing.T) {
	// Read-only go.mod files should surface write errors instead of ignoring failed rewrites.
	path := writeFile(t, t.TempDir(), "go.mod", "module prismgo\n")
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatalf("make go.mod read-only: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(path, 0o600); err != nil {
			t.Fatalf("restore go.mod permissions: %v", err)
		}
	})

	err := RewriteModule(path, "github.com/acme/myapp")
	if err == nil {
		t.Skip("filesystem permits writing to read-only owner files")
	}
	if !strings.Contains(err.Error(), "write go.mod") {
		t.Fatalf("expected write error, got %q", err.Error())
	}
}

func TestRewriteModuleRejectsSymlinkedGoMod(t *testing.T) {
	// Symlinked go.mod files must not be followed because rewriting through them can modify files outside the project.
	dir := t.TempDir()
	outside := writeFile(t, t.TempDir(), "go.mod", "module outside\n")
	linkPath := filepath.Join(dir, "go.mod")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Fatalf("create go.mod symlink: %v", err)
	}

	err := RewriteModule(linkPath, "github.com/acme/myapp")
	if err == nil {
		t.Fatal("expected symlinked go.mod to fail")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink error, got %q", err.Error())
	}
	if got := readFile(t, outside); got != "module outside\n" {
		t.Fatalf("expected outside go.mod to remain unchanged, got %q", got)
	}
}

func TestRewriteImportsUpdatesOnlyInternalPrismgoImports(t *testing.T) {
	// RewriteImports should rewrite project-local imports while preserving external framework imports.
	dir := t.TempDir()
	path := writeFile(t, dir, "main.go", `package main

import (
	"prismgo/bootstrap"
	"prismgo/app/http/controllers"
	"github.com/prismgo/framework/foundation"
)

func main() {}
`)

	if err := RewriteImports(dir, "prismgo", "github.com/acme/myapp"); err != nil {
		t.Fatalf("expected import rewrite to pass, got error: %v", err)
	}

	got := readFile(t, path)
	for _, want := range []string{
		`"github.com/acme/myapp/bootstrap"`,
		`"github.com/acme/myapp/app/http/controllers"`,
		`"github.com/prismgo/framework/foundation"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected rewritten file to contain %s, got:\n%s", want, got)
		}
	}
	if strings.Contains(got, `"prismgo/bootstrap"`) || strings.Contains(got, `"prismgo/app/http/controllers"`) {
		t.Fatalf("expected old internal imports to be removed, got:\n%s", got)
	}
}

func TestRewriteImportsFailsWhenRootIsMissing(t *testing.T) {
	// A missing root directory is a caller error and should surface from the filesystem walk.
	err := RewriteImports(filepath.Join(t.TempDir(), "missing"), "prismgo", "github.com/acme/myapp")
	if err == nil {
		t.Fatal("expected missing rewrite root to fail")
	}
	if !strings.Contains(err.Error(), "walk") {
		t.Fatalf("expected walk error, got %q", err.Error())
	}
}

func TestRewriteImportsRejectsSymlinkedGoFile(t *testing.T) {
	// Symlinked .go files inside root must not be parsed and rewritten through to files outside root.
	dir := t.TempDir()
	outside := writeFile(t, t.TempDir(), "outside.go", `package main

import "prismgo/bootstrap"

func main() {}
`)
	linkPath := filepath.Join(dir, "main.go")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Fatalf("create Go file symlink: %v", err)
	}

	err := RewriteImports(dir, "prismgo", "github.com/acme/myapp")
	if err == nil {
		t.Fatal("expected symlinked Go file to fail")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink error, got %q", err.Error())
	}
	if got := readFile(t, outside); !strings.Contains(got, `"prismgo/bootstrap"`) {
		t.Fatalf("expected outside Go file to remain unchanged, got:\n%s", got)
	}
}

func TestRewriteImportsUpdatesExactOldModuleImport(t *testing.T) {
	// Exact imports of the old module root should become exact imports of the new module root.
	dir := t.TempDir()
	path := writeFile(t, dir, "main.go", `package main

import "prismgo"

func main() {}
`)

	if err := RewriteImports(dir, "prismgo", "github.com/acme/myapp"); err != nil {
		t.Fatalf("expected import rewrite to pass, got error: %v", err)
	}

	got := readFile(t, path)
	if !strings.Contains(got, `"github.com/acme/myapp"`) {
		t.Fatalf("expected exact old module import to be rewritten, got:\n%s", got)
	}
}

func TestRewriteImportsPreservesImportNamesAndGroupedImports(t *testing.T) {
	// Aliases, dot imports, blank imports, and grouped imports should keep their import names after path rewriting.
	dir := t.TempDir()
	path := writeFile(t, dir, "main.go", `package main

import (
	bootstrapAlias "prismgo/bootstrap"
	. "prismgo/app/http/controllers"
	_ "prismgo/app/providers"

	"fmt"
)

func main() {
	_ = fmt.Sprintf
}
`)

	if err := RewriteImports(dir, "prismgo", "github.com/acme/myapp"); err != nil {
		t.Fatalf("expected import rewrite to pass, got error: %v", err)
	}

	got := readFile(t, path)
	for _, want := range []string{
		`bootstrapAlias "github.com/acme/myapp/bootstrap"`,
		`. "github.com/acme/myapp/app/http/controllers"`,
		`_ "github.com/acme/myapp/app/providers"`,
		`"fmt"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected rewritten file to contain %s, got:\n%s", want, got)
		}
	}
}

func TestRewriteImportsLeavesFilesWithoutMatchingImportsUnchanged(t *testing.T) {
	// Files with no old-module imports should not be formatted or rewritten unnecessarily.
	dir := t.TempDir()
	original := "package main\nimport \"fmt\"\nfunc main(){fmt.Println(\"ok\")}\n"
	path := writeFile(t, dir, "main.go", original)

	if err := RewriteImports(dir, "prismgo", "github.com/acme/myapp"); err != nil {
		t.Fatalf("expected import rewrite to pass, got error: %v", err)
	}
	if got := readFile(t, path); got != original {
		t.Fatalf("expected unmatched file to remain unchanged, got:\n%s", got)
	}
}

func TestRewriteImportsRejectsEmptyModules(t *testing.T) {
	// Empty old or new module values would make import path matching ambiguous and should fail early.
	dir := t.TempDir()

	if err := RewriteImports(dir, "", "github.com/acme/myapp"); err == nil {
		t.Fatal("expected empty old module to fail")
	}
	if err := RewriteImports(dir, "prismgo", " "); err == nil {
		t.Fatal("expected empty new module to fail")
	}
}

func TestRewriteImportsReturnsParseErrorForInvalidGoFile(t *testing.T) {
	// Invalid Go files should return a clear parse error instead of being skipped silently.
	dir := t.TempDir()
	writeFile(t, dir, "broken.go", "package main\n\nfunc broken(\n")

	err := RewriteImports(dir, "prismgo", "github.com/acme/myapp")
	if err == nil {
		t.Fatal("expected invalid Go file to fail")
	}
	if !strings.Contains(err.Error(), "parse") || !strings.Contains(err.Error(), "broken.go") {
		t.Fatalf("expected parse error to name the file, got %q", err.Error())
	}
}

func TestRewriteGoFileImportsRejectsDirectSymlink(t *testing.T) {
	// The file-level helper rechecks symlinks defensively in case callers bypass the directory walk.
	dir := t.TempDir()
	outside := writeFile(t, t.TempDir(), "outside.go", `package main

import "prismgo/bootstrap"

func main() {}
`)
	linkPath := filepath.Join(dir, "main.go")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Fatalf("create Go file symlink: %v", err)
	}

	err := rewriteGoFileImports(linkPath, "prismgo", "github.com/acme/myapp")
	if err == nil {
		t.Fatal("expected direct symlinked Go file rewrite to fail")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink error, got %q", err.Error())
	}
}

func TestRewriteGoFileImportsFailsWhenFileIsMissing(t *testing.T) {
	// The file-level helper should return inspect errors before parsing missing paths.
	err := rewriteGoFileImports(filepath.Join(t.TempDir(), "missing.go"), "prismgo", "github.com/acme/myapp")
	if err == nil {
		t.Fatal("expected missing Go file rewrite to fail")
	}
	if !strings.Contains(err.Error(), "inspect Go file") {
		t.Fatalf("expected inspect error, got %q", err.Error())
	}
}

func TestInitEnvSkipsWhenExampleIsMissing(t *testing.T) {
	// Missing .env.example means there is nothing to initialize, so InitEnv should be a no-op.
	dir := t.TempDir()

	if err := InitEnv(dir); err != nil {
		t.Fatalf("expected missing env example to be skipped, got error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".env")); !os.IsNotExist(err) {
		t.Fatalf("expected .env to remain absent, got stat error: %v", err)
	}
}

func TestWriteNewFileFailsWhenTargetExists(t *testing.T) {
	// Exclusive create semantics should refuse existing paths created between checks.
	path := writeFile(t, t.TempDir(), ".env", "APP_NAME=Existing\n")

	err := writeNewFile(path, []byte("APP_NAME=PrismGo\n"), 0o644)
	if err == nil {
		t.Fatal("expected exclusive write to existing file to fail")
	}
	if got := readFile(t, path); got != "APP_NAME=Existing\n" {
		t.Fatalf("expected existing file to remain unchanged, got %q", got)
	}
}

func TestInitEnvFailsWhenExampleCannotBeInspected(t *testing.T) {
	// A symlink loop at .env.example should return a stat error instead of being treated as absent.
	dir := t.TempDir()
	examplePath := filepath.Join(dir, ".env.example")
	if err := os.Symlink(".env.example", examplePath); err != nil {
		t.Fatalf("create .env.example symlink loop: %v", err)
	}

	err := InitEnv(dir)
	if err == nil {
		t.Fatal("expected symlink loop to fail")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected env example symlink error, got %q", err.Error())
	}
}

func TestInitEnvRejectsSymlinkedEnvExample(t *testing.T) {
	// Symlinked .env.example files must not be followed because they can copy outside content into .env.
	dir := t.TempDir()
	outside := writeFile(t, t.TempDir(), "secret.env", "SECRET=outside\n")
	if err := os.Symlink(outside, filepath.Join(dir, ".env.example")); err != nil {
		t.Fatalf("create .env.example symlink: %v", err)
	}

	err := InitEnv(dir)
	if err == nil {
		t.Fatal("expected symlinked .env.example to fail")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink error, got %q", err.Error())
	}
	if _, err := os.Stat(filepath.Join(dir, ".env")); !os.IsNotExist(err) {
		t.Fatalf("expected .env to remain absent, got stat error: %v", err)
	}
}

func TestInitEnvCopiesExampleOnlyWhenEnvMissing(t *testing.T) {
	// InitEnv should create .env from .env.example once and preserve an existing .env on later calls.
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "APP_NAME=PrismGo\n")

	if err := InitEnv(dir); err != nil {
		t.Fatalf("expected env initialization to pass, got error: %v", err)
	}
	if got := readFile(t, filepath.Join(dir, ".env")); got != "APP_NAME=PrismGo\n" {
		t.Fatalf("expected .env to copy example content, got %q", got)
	}

	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("APP_NAME=Existing\n"), 0o644); err != nil {
		t.Fatalf("write existing .env: %v", err)
	}
	if err := InitEnv(dir); err != nil {
		t.Fatalf("expected env initialization with existing .env to pass, got error: %v", err)
	}
	if got := readFile(t, filepath.Join(dir, ".env")); got != "APP_NAME=Existing\n" {
		t.Fatalf("expected existing .env to be preserved, got %q", got)
	}
}

func TestInitEnvRejectsSymlinkedEnv(t *testing.T) {
	// Existing .env symlinks must not be followed or overwritten by initialization.
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "APP_NAME=PrismGo\n")
	outside := writeFile(t, t.TempDir(), "outside.env", "APP_NAME=Outside\n")
	if err := os.Symlink(outside, filepath.Join(dir, ".env")); err != nil {
		t.Fatalf("create .env symlink: %v", err)
	}

	err := InitEnv(dir)
	if err == nil {
		t.Fatal("expected symlinked .env to fail")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink error, got %q", err.Error())
	}
	if got := readFile(t, outside); got != "APP_NAME=Outside\n" {
		t.Fatalf("expected outside .env target to remain unchanged, got %q", got)
	}
}

func TestInitEnvRejectsDanglingSymlinkedEnv(t *testing.T) {
	// Dangling .env symlinks must not be treated as missing because writes would create the outside target.
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "APP_NAME=PrismGo\n")
	outside := filepath.Join(t.TempDir(), "outside.env")
	if err := os.Symlink(outside, filepath.Join(dir, ".env")); err != nil {
		t.Fatalf("create dangling .env symlink: %v", err)
	}

	err := InitEnv(dir)
	if err == nil {
		t.Fatal("expected dangling symlinked .env to fail")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink error, got %q", err.Error())
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatalf("expected outside target to remain absent, got stat error: %v", err)
	}
}

func TestInitEnvFailsWhenEnvCannotBeInspected(t *testing.T) {
	// A symlink loop at .env should return a stat error so callers know initialization is unsafe.
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "APP_NAME=PrismGo\n")
	envPath := filepath.Join(dir, ".env")
	if err := os.Symlink(".env", envPath); err != nil {
		t.Fatalf("create .env symlink loop: %v", err)
	}

	err := InitEnv(dir)
	if err == nil {
		t.Fatal("expected .env symlink loop to fail")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected env symlink error, got %q", err.Error())
	}
}

func TestInitEnvFailsWhenExampleIsDirectory(t *testing.T) {
	// A directory named .env.example is malformed input and should not create .env.
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".env.example"), 0o755); err != nil {
		t.Fatalf("create .env.example directory: %v", err)
	}

	err := InitEnv(dir)
	if err == nil {
		t.Fatal("expected .env.example directory to fail")
	}
	if !strings.Contains(err.Error(), "is a directory") {
		t.Fatalf("expected directory error, got %q", err.Error())
	}
}

func writeFile(t *testing.T, root string, name string, content string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}
