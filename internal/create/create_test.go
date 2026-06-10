package create

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/prismgo/installer/internal/project"
	"github.com/prismgo/installer/internal/run"
	"github.com/prismgo/installer/internal/skeleton"
)

func TestCreateRewritesGeneratedProjectAndPreservesFrameworkImports(t *testing.T) {
	// The fixture models the official skeleton's local module imports and external framework imports.
	target := filepath.Join(t.TempDir(), "myapp")
	service := Service{
		Skeleton: skeleton.LocalSource{Dir: filepath.Join("testdata", "skeleton")},
		Runner:   &recordingRunner{},
	}

	err := service.Create(context.Background(), Options{
		Project: project.Plan{
			Name:      "myapp",
			Directory: target,
			Module:    "github.com/acme/myapp",
		},
		NoInstall: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	assertFileContains(t, filepath.Join(target, "go.mod"), "module github.com/acme/myapp")
	assertFileContains(t, filepath.Join(target, "main.go"), `"github.com/acme/myapp/bootstrap"`)
	assertFileContains(t, filepath.Join(target, "main.go"), `"github.com/prismgo/framework/console"`)
	assertFileContains(t, filepath.Join(target, "bootstrap", "app.go"), `"github.com/prismgo/framework/console"`)
	assertFileContains(t, filepath.Join(target, ".env"), "APP_NAME=PrismGo")
}

func TestCreateRunsDefaultSetupCommands(t *testing.T) {
	// The default flow verifies the generated app after rewrite so users get a ready project.
	target := filepath.Join(t.TempDir(), "myapp")
	runner := &recordingRunner{}
	service := Service{
		Skeleton: skeleton.LocalSource{Dir: filepath.Join("testdata", "skeleton")},
		Runner:   runner,
	}

	err := service.Create(context.Background(), Options{
		Project: project.Plan{
			Name:      "myapp",
			Directory: target,
			Module:    "github.com/acme/myapp",
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	want := []run.Command{
		{Name: "go", Args: []string{"mod", "tidy"}, Dir: target},
		{Name: "go", Args: []string{"test", "./..."}, Dir: target},
	}
	if !reflect.DeepEqual(runner.commands, want) {
		t.Fatalf("recorded commands = %#v, want %#v", runner.commands, want)
	}
}

func TestCreateWithGitRunsRepositoryInitializationAfterSetup(t *testing.T) {
	// Git initialization should run after dependency setup and verification so the first commit captures a ready app.
	target := filepath.Join(t.TempDir(), "myapp")
	runner := &recordingRunner{}
	service := Service{
		Skeleton: skeleton.LocalSource{Dir: filepath.Join("testdata", "skeleton")},
		Runner:   runner,
	}

	err := service.Create(context.Background(), Options{
		Project: project.Plan{
			Name:      "myapp",
			Directory: target,
			Module:    "github.com/acme/myapp",
		},
		Git: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	want := []run.Command{
		{Name: "go", Args: []string{"mod", "tidy"}, Dir: target},
		{Name: "go", Args: []string{"test", "./..."}, Dir: target},
		{Name: "git", Args: []string{"init"}, Dir: target},
		{Name: "git", Args: []string{"add", "."}, Dir: target},
		{Name: "git", Args: []string{"commit", "-m", "Set up a fresh PrismGo app"}, Dir: target},
		{Name: "git", Args: []string{"branch", "-M", "--", "main"}, Dir: target},
	}
	if !reflect.DeepEqual(runner.commands, want) {
		t.Fatalf("recorded commands = %#v, want %#v", runner.commands, want)
	}
}

func TestCreateWithGitUsesRequestedBranch(t *testing.T) {
	// A requested branch name should replace the default initial branch rename.
	target := filepath.Join(t.TempDir(), "myapp")
	runner := &recordingRunner{}
	service := Service{
		Skeleton: skeleton.LocalSource{Dir: filepath.Join("testdata", "skeleton")},
		Runner:   runner,
	}

	err := service.Create(context.Background(), Options{
		Project: project.Plan{
			Name:      "myapp",
			Directory: target,
			Module:    "github.com/acme/myapp",
		},
		Git:    true,
		Branch: "develop",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	last := runner.commands[len(runner.commands)-1]
	want := run.Command{Name: "git", Args: []string{"branch", "-M", "--", "develop"}, Dir: target}
	if !reflect.DeepEqual(last, want) {
		t.Fatalf("last recorded command = %#v, want %#v", last, want)
	}
}

func TestCreateNoInstallSkipsSetupCommands(t *testing.T) {
	// --no-install keeps generation filesystem-only for offline or deferred dependency setup.
	target := filepath.Join(t.TempDir(), "myapp")
	runner := &recordingRunner{}
	service := Service{
		Skeleton: skeleton.LocalSource{Dir: filepath.Join("testdata", "skeleton")},
		Runner:   runner,
	}

	err := service.Create(context.Background(), Options{
		Project: project.Plan{
			Name:      "myapp",
			Directory: target,
			Module:    "github.com/acme/myapp",
		},
		NoInstall: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("recorded commands = %#v, want none", runner.commands)
	}
}

func TestCreateNoInstallWithGitRunsOnlyRepositoryInitialization(t *testing.T) {
	// When install is skipped, git initialization should still commit the rewritten project files.
	target := filepath.Join(t.TempDir(), "myapp")
	runner := &recordingRunner{}
	service := Service{
		Skeleton: skeleton.LocalSource{Dir: filepath.Join("testdata", "skeleton")},
		Runner:   runner,
	}

	err := service.Create(context.Background(), Options{
		Project: project.Plan{
			Name:      "myapp",
			Directory: target,
			Module:    "github.com/acme/myapp",
		},
		NoInstall: true,
		Git:       true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	want := []run.Command{
		{Name: "git", Args: []string{"init"}, Dir: target},
		{Name: "git", Args: []string{"add", "."}, Dir: target},
		{Name: "git", Args: []string{"commit", "-m", "Set up a fresh PrismGo app"}, Dir: target},
		{Name: "git", Args: []string{"branch", "-M", "--", "main"}, Dir: target},
	}
	if !reflect.DeepEqual(runner.commands, want) {
		t.Fatalf("recorded commands = %#v, want %#v", runner.commands, want)
	}
}

func TestCreateRequiresSkeletonSource(t *testing.T) {
	// Create needs an explicit skeleton source because there is no safe default filesystem location.
	err := (Service{Runner: &recordingRunner{}}).Create(context.Background(), Options{
		Project: project.Plan{
			Directory: filepath.Join(t.TempDir(), "myapp"),
			Module:    "github.com/acme/myapp",
		},
	})
	if err == nil {
		t.Fatal("Create() error = nil, want skeleton source error")
	}
	if !strings.Contains(err.Error(), "skeleton source is required") {
		t.Fatalf("Create() error = %q, want skeleton source error", err.Error())
	}
}

func TestCreateRequiresRunnerWhenInstallRuns(t *testing.T) {
	// The setup command path must fail early if no runner is available to execute go commands.
	err := (Service{Skeleton: skeleton.LocalSource{Dir: filepath.Join("testdata", "skeleton")}}).Create(context.Background(), Options{
		Project: project.Plan{
			Directory: filepath.Join(t.TempDir(), "myapp"),
			Module:    "github.com/acme/myapp",
		},
	})
	if err == nil {
		t.Fatal("Create() error = nil, want command runner error")
	}
	if !strings.Contains(err.Error(), "command runner is required") {
		t.Fatalf("Create() error = %q, want command runner error", err.Error())
	}
}

func TestCreateRequiresRunnerWhenGitRuns(t *testing.T) {
	// Git initialization also needs the command runner even when dependency setup is skipped.
	err := (Service{Skeleton: skeleton.LocalSource{Dir: filepath.Join("testdata", "skeleton")}}).Create(context.Background(), Options{
		Project: project.Plan{
			Directory: filepath.Join(t.TempDir(), "myapp"),
			Module:    "github.com/acme/myapp",
		},
		NoInstall: true,
		Git:       true,
	})
	if err == nil {
		t.Fatal("Create() error = nil, want command runner error")
	}
	if !strings.Contains(err.Error(), "command runner is required") {
		t.Fatalf("Create() error = %q, want command runner error", err.Error())
	}
}

func TestCreateRejectsInvalidGitBranchBeforeCopy(t *testing.T) {
	// Branch validation must happen before skeleton copy so invalid git input leaves no partial project.
	target := filepath.Join(t.TempDir(), "myapp")
	runner := &recordingRunner{}
	err := (Service{Skeleton: skeleton.LocalSource{Dir: filepath.Join("testdata", "skeleton")}, Runner: runner}).Create(context.Background(), Options{
		Project: project.Plan{
			Name:      "myapp",
			Directory: target,
			Module:    "github.com/acme/myapp",
		},
		Git:    true,
		Branch: "--help",
	})
	if err == nil {
		t.Fatal("Create() error = nil, want invalid branch error")
	}
	if !strings.Contains(err.Error(), "branch") {
		t.Fatalf("Create() error = %q, want branch error", err.Error())
	}
	if len(runner.commands) != 0 {
		t.Fatalf("recorded commands = %#v, want none", runner.commands)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("target should not exist after invalid branch, stat error: %v", statErr)
	}
}

func TestNormalizeBranchRejectsInvalidNames(t *testing.T) {
	invalid := []string{"-bad", "/bad", "bad/", "bad//name", "bad..name", "bad.lock", "bad.", "@", "bad@{name", "bad name", "bad:name", "bad\\name", "foo/.bar", "foo.lock/bar"}
	for _, branch := range invalid {
		if _, err := normalizeBranch(branch); err == nil {
			t.Fatalf("normalizeBranch(%q) error = nil, want error", branch)
		}
	}
}

func TestNormalizeBranchDefaultsEmptyToMain(t *testing.T) {
	branch, err := normalizeBranch("")
	if err != nil {
		t.Fatalf("normalizeBranch empty error = %v", err)
	}
	if branch != "main" {
		t.Fatalf("normalizeBranch empty = %q, want main", branch)
	}
}

func TestCreateReturnsSkeletonErrors(t *testing.T) {
	// Skeleton acquisition errors should stop before rewrite or command execution.
	wantErr := errors.New("copy failed")
	runner := &recordingRunner{}
	err := (Service{Skeleton: failingSource{err: wantErr}, Runner: runner}).Create(context.Background(), Options{
		Project: project.Plan{
			Directory: filepath.Join(t.TempDir(), "myapp"),
			Module:    "github.com/acme/myapp",
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Create() error = %v, want %v", err, wantErr)
	}
	if len(runner.commands) != 0 {
		t.Fatalf("recorded commands = %#v, want none", runner.commands)
	}
}

func TestCreateReturnsSetupCommandErrors(t *testing.T) {
	// A failing setup command should be returned directly so callers can show the command output.
	target := filepath.Join(t.TempDir(), "myapp")
	wantErr := errors.New("go mod tidy failed")
	runner := &recordingRunner{err: wantErr}
	service := Service{
		Skeleton: skeleton.LocalSource{Dir: filepath.Join("testdata", "skeleton")},
		Runner:   runner,
	}

	err := service.Create(context.Background(), Options{
		Project: project.Plan{
			Name:      "myapp",
			Directory: target,
			Module:    "github.com/acme/myapp",
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Create() error = %v, want %v", err, wantErr)
	}
	want := []run.Command{
		{Name: "go", Args: []string{"mod", "tidy"}, Dir: target},
	}
	if !reflect.DeepEqual(runner.commands, want) {
		t.Fatalf("recorded commands = %#v, want %#v", runner.commands, want)
	}
}

func TestCreateReturnsGitCommandErrors(t *testing.T) {
	// A failing git command should stop the repository setup and return the runner error.
	target := filepath.Join(t.TempDir(), "myapp")
	wantErr := errors.New("git init failed")
	runner := &recordingRunner{err: wantErr}
	service := Service{
		Skeleton: skeleton.LocalSource{Dir: filepath.Join("testdata", "skeleton")},
		Runner:   runner,
	}

	err := service.Create(context.Background(), Options{
		Project: project.Plan{
			Name:      "myapp",
			Directory: target,
			Module:    "github.com/acme/myapp",
		},
		NoInstall: true,
		Git:       true,
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Create() error = %v, want %v", err, wantErr)
	}
	want := []run.Command{
		{Name: "git", Args: []string{"init"}, Dir: target},
	}
	if !reflect.DeepEqual(runner.commands, want) {
		t.Fatalf("recorded commands = %#v, want %#v", runner.commands, want)
	}
}

type failingSource struct {
	err error
}

func (s failingSource) CopyTo(context.Context, string) error {
	return s.err
}

type recordingRunner struct {
	commands []run.Command
	err      error
}

func (r *recordingRunner) Run(_ context.Context, cmd run.Command) error {
	// Copy Args so later caller mutations cannot change the recorded command history.
	cmd.Args = append([]string(nil), cmd.Args...)
	r.commands = append(r.commands, cmd)
	return r.err
}

func assertFileContains(t *testing.T, path string, want string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(content), want) {
		t.Fatalf("%s does not contain %q:\n%s", path, want, content)
	}
}
