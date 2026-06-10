package cli

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/prismgo/installer/internal/create"
	"github.com/prismgo/installer/internal/project"
)

func TestNewCommandCallsCreateService(t *testing.T) {
	// The CLI should stay thin: parse flags, resolve the project, then pass creation options through.
	cwd := t.TempDir()
	creator := &recordingCreator{}
	cmd := newCommandWithCreator(creator)
	cmd.SetArgs([]string{"myapp", "--module", "github.com/acme/myapp", "--no-install", "--git", "--branch", "develop"})

	err := runCommandInDir(t, cwd, cmd)
	if err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}

	want := create.Options{
		Project: project.Plan{
			Name:      "myapp",
			Directory: filepath.Join(cwd, "myapp"),
			Module:    "github.com/acme/myapp",
		},
		NoInstall: true,
		Git:       true,
		Branch:    "develop",
	}
	if !reflect.DeepEqual(creator.options, want) {
		t.Fatalf("Create() options = %#v, want %#v", creator.options, want)
	}
	if creator.calls != 1 {
		t.Fatalf("Create() calls = %d, want 1", creator.calls)
	}
}

func TestNewCommandModulePathResolvesDirectoryName(t *testing.T) {
	// Module-path input keeps the full module while using the final path segment as the local directory.
	cwd := t.TempDir()
	creator := &recordingCreator{}
	cmd := newCommandWithCreator(creator)
	cmd.SetArgs([]string{"github.com/acme/myapp", "--no-install"})

	err := runCommandInDir(t, cwd, cmd)
	if err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if creator.options.Project.Name != "myapp" {
		t.Fatalf("project name = %q, want %q", creator.options.Project.Name, "myapp")
	}
	if creator.options.Project.Module != "github.com/acme/myapp" {
		t.Fatalf("project module = %q, want %q", creator.options.Project.Module, "github.com/acme/myapp")
	}
}

func TestExecuteNewRejectsUnsafeProjectPath(t *testing.T) {
	// CLI callers should see project validation errors before any later create operation can run.
	err := Execute(context.Background(), []string{"new", "../app"})
	if err == nil {
		t.Fatal("expected unsafe path error, got nil")
	}
	if strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("expected validation error before placeholder, got %q", err.Error())
	}
}

func TestExecuteNewWithGitHubReturnsUnsupportedError(t *testing.T) {
	// --github is explicitly unsupported in the MVP and should not fall through to the generic placeholder.
	err := Execute(context.Background(), []string{"new", "myapp", "--github"})
	if err == nil {
		t.Fatal("expected unsupported GitHub error, got nil")
	}
	if !strings.Contains(err.Error(), "github repository creation is not supported yet") {
		t.Fatalf("expected unsupported GitHub error, got %q", err.Error())
	}
}

func TestNewCommandWithGitPropagatesToCreateService(t *testing.T) {
	// --git is supported by the create service, so the CLI should pass it through without special handling.
	creator := &recordingCreator{}
	cmd := newCommandWithCreator(creator)
	cmd.SetArgs([]string{"myapp", "--git"})

	err := runCommandInDir(t, t.TempDir(), cmd)
	if err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if !creator.options.Git {
		t.Fatal("Create() Git = false, want true")
	}
	if creator.options.Branch != "main" {
		t.Fatalf("Create() Branch = %q, want %q", creator.options.Branch, "main")
	}
	if creator.calls != 1 {
		t.Fatalf("Create() calls = %d, want 1", creator.calls)
	}
}

func TestExecuteNewRequiresName(t *testing.T) {
	// The command requires exactly one positional project name before later creation logic runs.
	err := Execute(context.Background(), []string{"new"})
	if err == nil {
		t.Fatal("expected missing name error, got nil")
	}
}

func TestNewCommandRegistersExpectedFlags(t *testing.T) {
	// The initial command contract defines all flags Task 1 exposes to later implementation steps.
	cmd := NewCommand()

	expectedFlags := []string{
		"module",
		"no-install",
		"git",
		"branch",
		"force",
		"github",
	}
	for _, name := range expectedFlags {
		if flag := cmd.Flags().Lookup(name); flag == nil {
			t.Fatalf("expected flag %q to be registered", name)
		}
	}

	branch := cmd.Flags().Lookup("branch")
	if branch == nil {
		t.Fatal("expected branch flag to be registered")
	}
	if branch.DefValue != "main" {
		t.Fatalf("expected branch default %q, got %q", "main", branch.DefValue)
	}
}

type recordingCreator struct {
	calls   int
	options create.Options
}

func (r *recordingCreator) Create(_ context.Context, opts create.Options) error {
	// Store the exact options passed by the CLI so tests cover flag propagation and project resolution together.
	r.calls++
	r.options = opts
	return nil
}

func runCommandInDir(t *testing.T, dir string, cmd interface {
	ExecuteContext(context.Context) error
}) error {
	t.Helper()

	t.Chdir(dir)
	return cmd.ExecuteContext(context.Background())
}

func TestRootCommandHelpExecutesWithoutError(t *testing.T) {
	// Help should render successfully without invoking any child command behavior.
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"--help"})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("expected root help to execute without error, got %v", err)
	}
}
