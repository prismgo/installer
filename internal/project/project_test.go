package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePlainNameUsesNameForDirectoryAndModule(t *testing.T) {
	// A simple project name should create a directory with the same name and keep the module local.
	cwd := t.TempDir()

	plan, err := Resolve(Options{CWD: cwd, Name: "myapp"})
	if err != nil {
		t.Fatalf("expected project plan, got error: %v", err)
	}

	if plan.Directory != filepath.Join(cwd, "myapp") {
		t.Fatalf("expected directory %q, got %q", filepath.Join(cwd, "myapp"), plan.Directory)
	}
	if plan.Module != "myapp" {
		t.Fatalf("expected module %q, got %q", "myapp", plan.Module)
	}
	if plan.Name != "myapp" {
		t.Fatalf("expected name %q, got %q", "myapp", plan.Name)
	}
}

func TestResolveModulePathUsesFinalSegmentForDirectory(t *testing.T) {
	// Module-like paths keep the full module path while placing files in the final path segment.
	cwd := t.TempDir()

	plan, err := Resolve(Options{CWD: cwd, Name: "github.com/acme/myapp"})
	if err != nil {
		t.Fatalf("expected project plan, got error: %v", err)
	}

	if plan.Directory != filepath.Join(cwd, "myapp") {
		t.Fatalf("expected directory %q, got %q", filepath.Join(cwd, "myapp"), plan.Directory)
	}
	if plan.Module != "github.com/acme/myapp" {
		t.Fatalf("expected module %q, got %q", "github.com/acme/myapp", plan.Module)
	}
	if plan.Name != "myapp" {
		t.Fatalf("expected name %q, got %q", "myapp", plan.Name)
	}
}

func TestResolveExplicitModuleOverridesInferredModule(t *testing.T) {
	// The --module value should control the generated go.mod module without changing the directory name.
	cwd := t.TempDir()

	plan, err := Resolve(Options{
		CWD:    cwd,
		Name:   "myapp",
		Module: "github.com/acme/service",
	})
	if err != nil {
		t.Fatalf("expected project plan, got error: %v", err)
	}

	if plan.Directory != filepath.Join(cwd, "myapp") {
		t.Fatalf("expected directory %q, got %q", filepath.Join(cwd, "myapp"), plan.Directory)
	}
	if plan.Module != "github.com/acme/service" {
		t.Fatalf("expected module %q, got %q", "github.com/acme/service", plan.Module)
	}
}

func TestResolveRejectsEmptyName(t *testing.T) {
	// Empty project names cannot produce a safe target directory or module name.
	_, err := Resolve(Options{CWD: t.TempDir(), Name: ""})
	if err == nil {
		t.Fatal("expected empty name to fail")
	}
}

func TestResolveRejectsPathTraversal(t *testing.T) {
	// The project argument must not be able to escape the current working directory.
	_, err := Resolve(Options{CWD: t.TempDir(), Name: "../app"})
	if err == nil {
		t.Fatal("expected path traversal to fail")
	}
}

func TestResolveTargetRejectsEscapingDirectory(t *testing.T) {
	// The lower-level boundary check protects callers even when a directory name was not prevalidated.
	_, err := resolveTarget(t.TempDir(), "../app")
	if err == nil {
		t.Fatal("expected escaping target directory to fail")
	}
}

func TestResolveRejectsAbsolutePath(t *testing.T) {
	// Absolute paths would bypass the current working directory boundary and must be rejected.
	_, err := Resolve(Options{CWD: t.TempDir(), Name: filepath.Join(string(filepath.Separator), "tmp", "app")})
	if err == nil {
		t.Fatal("expected absolute path to fail")
	}
}

func TestResolveRejectsEmptyPathSegment(t *testing.T) {
	// Empty path segments make target resolution ambiguous and should not be accepted.
	_, err := Resolve(Options{CWD: t.TempDir(), Name: "github.com/acme/"})
	if err == nil {
		t.Fatal("expected empty path segment to fail")
	}
}

func TestResolveUsesProcessWorkingDirectoryWhenCWDIsEmpty(t *testing.T) {
	// Empty CWD lets CLI callers rely on the process working directory without passing it explicitly.
	cwd := t.TempDir()
	t.Chdir(cwd)

	plan, err := Resolve(Options{Name: "myapp"})
	if err != nil {
		t.Fatalf("expected project plan, got error: %v", err)
	}
	if plan.Directory != filepath.Join(cwd, "myapp") {
		t.Fatalf("expected directory %q, got %q", filepath.Join(cwd, "myapp"), plan.Directory)
	}
}

func TestResolveExistingDirectoryWithoutForceFails(t *testing.T) {
	// Existing targets are refused unless --force is set and the directory is empty.
	cwd := t.TempDir()
	if err := os.Mkdir(filepath.Join(cwd, "myapp"), 0o755); err != nil {
		t.Fatalf("create existing directory: %v", err)
	}

	_, err := Resolve(Options{CWD: cwd, Name: "myapp"})
	if err == nil {
		t.Fatal("expected existing directory without force to fail")
	}
}

func TestResolveExistingFileFails(t *testing.T) {
	// File collisions are explicit failures because later creation steps expect a directory target.
	cwd := t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, "myapp"), []byte("occupied"), 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	_, err := Resolve(Options{CWD: cwd, Name: "myapp", Force: true})
	if err == nil {
		t.Fatal("expected existing file to fail")
	}
}

func TestResolveExistingEmptyDirectoryWithForcePasses(t *testing.T) {
	// --force permits reusing an empty directory without deleting or creating paths.
	cwd := t.TempDir()
	target := filepath.Join(cwd, "myapp")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create existing directory: %v", err)
	}

	plan, err := Resolve(Options{CWD: cwd, Name: "myapp", Force: true})
	if err != nil {
		t.Fatalf("expected existing empty directory with force to pass, got error: %v", err)
	}
	if plan.Directory != target {
		t.Fatalf("expected directory %q, got %q", target, plan.Directory)
	}
}

func TestResolveExistingNonEmptyDirectoryWithForceFails(t *testing.T) {
	// --force must not overwrite non-empty directories because later tasks may copy project files there.
	cwd := t.TempDir()
	target := filepath.Join(cwd, "myapp")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create existing directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "README.md"), []byte("occupied"), 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	_, err := Resolve(Options{CWD: cwd, Name: "myapp", Force: true})
	if err == nil {
		t.Fatal("expected existing non-empty directory with force to fail")
	}
}
