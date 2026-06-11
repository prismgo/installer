package skeleton

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/prismgo/installer/internal/run"
)

func TestLocalSourceCopiesSkeletonWithoutGitDirectory(t *testing.T) {
	// A local fixture exercises the same copy path as cloned skeletons without requiring network access.
	source := t.TempDir()
	target := filepath.Join(t.TempDir(), "app")
	writeFile(t, filepath.Join(source, "go.mod"), []byte("module prismgo\n"), 0o644)
	writeFile(t, filepath.Join(source, "app", "http", "controller.go"), []byte("package http\n"), 0o600)
	writeFile(t, filepath.Join(source, ".git", "config"), []byte("[core]\n"), 0o644)
	writeFile(t, filepath.Join(source, ".github", "workflows", "ci.yml"), []byte("name: ci\n"), 0o644)

	if err := (LocalSource{Dir: source}).CopyTo(context.Background(), target); err != nil {
		t.Fatalf("copy skeleton: %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, ".git")); !os.IsNotExist(err) {
		t.Fatalf("expected .git to be excluded, stat error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, ".github")); !os.IsNotExist(err) {
		t.Fatalf("expected .github to be excluded, stat error: %v", err)
	}
	assertFileContent(t, filepath.Join(target, "go.mod"), "module prismgo\n")
	assertFileContent(t, filepath.Join(target, "app", "http", "controller.go"), "package http\n")
	assertFileMode(t, filepath.Join(target, "app", "http", "controller.go"), 0o600)
}

func TestLocalSourceRejectsSymlinkEntries(t *testing.T) {
	// Symlinks are rejected instead of followed so fixture copy cannot escape the source tree.
	source := t.TempDir()
	target := filepath.Join(t.TempDir(), "app")
	writeFile(t, filepath.Join(source, "go.mod"), []byte("module prismgo\n"), 0o644)
	createSymlinkOrSkip(t, "/tmp/outside", filepath.Join(source, "outside"))

	err := (LocalSource{Dir: source}).CopyTo(context.Background(), target)
	if err == nil {
		t.Fatal("expected symlink entry to fail")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink error, got: %v", err)
	}
}

func TestLocalSourceRequiresDirectory(t *testing.T) {
	// A missing local source is a caller error and should fail before any target writes.
	err := (LocalSource{}).CopyTo(context.Background(), filepath.Join(t.TempDir(), "app"))
	if err == nil {
		t.Fatal("expected missing source directory to fail")
	}
}

func TestCopyTreeRejectsFileSource(t *testing.T) {
	// Skeleton sources must be directories so the copy boundary is unambiguous.
	source := filepath.Join(t.TempDir(), "go.mod")
	writeFile(t, source, []byte("module prismgo\n"), 0o644)

	err := copyTree(context.Background(), source, filepath.Join(t.TempDir(), "app"))
	if err == nil {
		t.Fatal("expected file source to fail")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected not-a-directory error, got: %v", err)
	}
}

func TestCopyTreeReportsMissingSource(t *testing.T) {
	// Missing skeleton roots should fail with source inspection context.
	err := copyTree(context.Background(), filepath.Join(t.TempDir(), "missing"), filepath.Join(t.TempDir(), "app"))
	if err == nil {
		t.Fatal("expected missing source to fail")
	}
	if !strings.Contains(err.Error(), "inspect skeleton source") {
		t.Fatalf("expected source inspection error, got: %v", err)
	}
}

func TestCopyTreeRejectsSymlinkSource(t *testing.T) {
	// The root source is checked with Lstat so callers cannot provide a symlinked skeleton root.
	realSource := t.TempDir()
	link := filepath.Join(t.TempDir(), "source-link")
	createSymlinkOrSkip(t, realSource, link)

	err := copyTree(context.Background(), link, filepath.Join(t.TempDir(), "app"))
	if err == nil {
		t.Fatal("expected symlink source to fail")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink error, got: %v", err)
	}
}

func TestCopyTreeRequiresTarget(t *testing.T) {
	// Empty targets are rejected before recursive copying to avoid accidental writes to CWD.
	err := copyTree(context.Background(), t.TempDir(), "")
	if err == nil {
		t.Fatal("expected empty target to fail")
	}
}

func TestCopyDirectoryReturnsCanceledContext(t *testing.T) {
	// Cancellation is checked before filesystem writes so orchestration can abort promptly.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := copyDirectory(ctx, t.TempDir(), filepath.Join(t.TempDir(), "app"), 0o755)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}

func TestCopyDirectoryReportsTargetCreateFailure(t *testing.T) {
	// If the target path cannot be inspected as a directory, callers should see that filesystem error.
	blocker := filepath.Join(t.TempDir(), "blocker")
	writeFile(t, blocker, []byte("occupied"), 0o644)

	err := copyDirectory(context.Background(), t.TempDir(), filepath.Join(blocker, "app"), 0o755)
	if err == nil {
		t.Fatal("expected target directory failure")
	}
	if !strings.Contains(err.Error(), "target directory") {
		t.Fatalf("expected target directory error, got: %v", err)
	}
}

func TestCopyDirectoryStopsBeforeLaterEntriesWhenContextIsCanceled(t *testing.T) {
	// Entry-loop cancellation prevents later files from being copied after orchestration aborts.
	source := t.TempDir()
	target := filepath.Join(t.TempDir(), "app")
	writeFile(t, filepath.Join(source, "a.go"), []byte("package a\n"), 0o644)
	writeFile(t, filepath.Join(source, "b.go"), []byte("package b\n"), 0o644)
	ctx := &errAfterContext{remainingNil: 3}

	err := copyDirectory(ctx, source, target, 0o755)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
	assertFileContent(t, filepath.Join(target, "a.go"), "package a\n")
	if _, err := os.Stat(filepath.Join(target, "b.go")); !os.IsNotExist(err) {
		t.Fatalf("expected later entry to be skipped, stat error: %v", err)
	}
}

func TestCopyDirectoryReturnsNestedCopyFailure(t *testing.T) {
	// Recursive directory copies surface nested file conflicts without overwriting target contents.
	source := t.TempDir()
	target := filepath.Join(t.TempDir(), "app")
	writeFile(t, filepath.Join(source, "nested", "go.mod"), []byte("module prismgo\n"), 0o644)
	writeFile(t, filepath.Join(target, "nested", "go.mod"), []byte("existing\n"), 0o644)

	err := copyDirectory(context.Background(), source, target, 0o755)
	if err == nil {
		t.Fatal("expected nested copy failure")
	}
	if !strings.Contains(err.Error(), "target file already exists") {
		t.Fatalf("expected existing target error, got: %v", err)
	}
	assertFileContent(t, filepath.Join(target, "nested", "go.mod"), "existing\n")
}

func TestCopyDirectoryRejectsNestedTargetDirectorySymlink(t *testing.T) {
	// Existing target directory symlinks are rejected so nested copies cannot escape the target tree.
	source := t.TempDir()
	target := filepath.Join(t.TempDir(), "app")
	outside := t.TempDir()
	writeFile(t, filepath.Join(source, "nested", "go.mod"), []byte("module prismgo\n"), 0o644)
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("create target root: %v", err)
	}
	createSymlinkOrSkip(t, outside, filepath.Join(target, "nested"))

	err := copyDirectory(context.Background(), source, target, 0o755)
	if err == nil {
		t.Fatal("expected nested target symlink failure")
	}
	if !strings.Contains(err.Error(), "target directory already exists as symlink") {
		t.Fatalf("expected target symlink error, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "go.mod")); !os.IsNotExist(err) {
		t.Fatalf("expected outside file to remain absent, stat error: %v", err)
	}
}

func TestCopyDirectoryReportsSourceReadFailure(t *testing.T) {
	// A non-directory source passed to the lower-level helper should report the read failure directly.
	source := filepath.Join(t.TempDir(), "go.mod")
	writeFile(t, source, []byte("module prismgo\n"), 0o644)

	err := copyDirectory(context.Background(), source, filepath.Join(t.TempDir(), "app"), 0o755)
	if err == nil {
		t.Fatal("expected source read failure")
	}
	if !strings.Contains(err.Error(), "read skeleton directory") {
		t.Fatalf("expected source read error, got: %v", err)
	}
}

func TestCopyFileReportsSourceOpenFailure(t *testing.T) {
	// Missing source files should preserve the open failure context.
	err := copyFile(context.Background(), filepath.Join(t.TempDir(), "missing.go"), filepath.Join(t.TempDir(), "main.go"), 0o644)
	if err == nil {
		t.Fatal("expected source open failure")
	}
	if !strings.Contains(err.Error(), "open skeleton file") {
		t.Fatalf("expected source open error, got: %v", err)
	}
}

func TestCopyFileReturnsCanceledContextBeforeTargetWrite(t *testing.T) {
	// File-level cancellation is checked before target inspection or creation.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	source := filepath.Join(t.TempDir(), "go.mod")
	target := filepath.Join(t.TempDir(), "go.mod")
	writeFile(t, source, []byte("module prismgo\n"), 0o644)

	err := copyFile(ctx, source, target, 0o644)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected no target file, stat error: %v", err)
	}
}

func TestCopyFileRejectsExistingTargetFileWithoutOverwrite(t *testing.T) {
	// Existing files are caller-owned output and must not be overwritten by skeleton copy.
	source := filepath.Join(t.TempDir(), "go.mod")
	target := filepath.Join(t.TempDir(), "go.mod")
	writeFile(t, source, []byte("module prismgo\n"), 0o644)
	writeFile(t, target, []byte("existing\n"), 0o644)

	err := copyFile(context.Background(), source, target, 0o644)
	if err == nil {
		t.Fatal("expected existing target to fail")
	}
	if !strings.Contains(err.Error(), "target file already exists") {
		t.Fatalf("expected existing target error, got: %v", err)
	}
	assertFileContent(t, target, "existing\n")
}

func TestCopyFileRejectsExistingTargetSymlinkWithoutOverwrite(t *testing.T) {
	// Lstat rejects destination symlinks before create so copy cannot write outside the target tree.
	dir := t.TempDir()
	source := filepath.Join(dir, "go.mod")
	outside := filepath.Join(dir, "outside.txt")
	target := filepath.Join(dir, "target-link")
	writeFile(t, source, []byte("module prismgo\n"), 0o644)
	writeFile(t, outside, []byte("outside\n"), 0o644)
	createSymlinkOrSkip(t, outside, target)

	err := copyFile(context.Background(), source, target, 0o644)
	if err == nil {
		t.Fatal("expected existing symlink target to fail")
	}
	if !strings.Contains(err.Error(), "target file already exists") {
		t.Fatalf("expected existing target error, got: %v", err)
	}
	assertFileContent(t, outside, "outside\n")
}

func TestCopyFileReportsCopyFailure(t *testing.T) {
	// A source that opens but cannot be streamed should return copy context and close opened files.
	source := t.TempDir()
	target := filepath.Join(t.TempDir(), "go.mod")

	err := copyFile(context.Background(), source, target, 0o644)
	if err == nil {
		t.Fatal("expected copy failure")
	}
	if !strings.Contains(err.Error(), "copy skeleton file") {
		t.Fatalf("expected copy error, got: %v", err)
	}
}

func TestCopyFileReportsTargetCreateFailure(t *testing.T) {
	// Directory targets are existing entries and should be rejected before file creation.
	source := filepath.Join(t.TempDir(), "go.mod")
	writeFile(t, source, []byte("module prismgo\n"), 0o644)

	err := copyFile(context.Background(), source, t.TempDir(), 0o644)
	if err == nil {
		t.Fatal("expected existing target failure")
	}
	if !strings.Contains(err.Error(), "target file already exists") {
		t.Fatalf("expected existing target error, got: %v", err)
	}
}

func TestCopyFileReportsTargetCreateOpenFailure(t *testing.T) {
	// A non-writable parent can fail atomic target creation after the source has opened.
	if runtime.GOOS == "windows" {
		t.Skip("Windows permission semantics do not reliably reject create in a chmod-only fixture")
	}
	source := filepath.Join(t.TempDir(), "go.mod")
	writeFile(t, source, []byte("module prismgo\n"), 0o644)
	targetDir := filepath.Join(t.TempDir(), "locked")
	if err := os.Mkdir(targetDir, 0o555); err != nil {
		t.Fatalf("create locked directory: %v", err)
	}
	defer func() {
		if err := os.Chmod(targetDir, 0o755); err != nil {
			t.Fatalf("restore locked directory permissions: %v", err)
		}
	}()

	err := copyFile(context.Background(), source, filepath.Join(targetDir, "go.mod"), 0o644)
	if err == nil {
		t.Fatal("expected target create failure")
	}
	if !strings.Contains(err.Error(), "create target file") {
		t.Fatalf("expected target create error, got: %v", err)
	}
}

func TestCopyFileReportsParentCreateFailure(t *testing.T) {
	// A file occupying the target parent path should surface during target inspection before writes.
	source := filepath.Join(t.TempDir(), "go.mod")
	writeFile(t, source, []byte("module prismgo\n"), 0o644)
	blocker := filepath.Join(t.TempDir(), "blocker")
	writeFile(t, blocker, []byte("occupied"), 0o644)

	err := copyFile(context.Background(), source, filepath.Join(blocker, "go.mod"), 0o644)
	if err == nil {
		t.Fatal("expected target inspection failure")
	}
	if !strings.Contains(err.Error(), "inspect target file") {
		t.Fatalf("expected target inspection error, got: %v", err)
	}
}

func TestCloseFileReportsCloseError(t *testing.T) {
	// Close failures are wrapped with operation context for callers to preserve delayed write errors.
	file, err := os.CreateTemp(t.TempDir(), "closed-*")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}

	err = closeFile(file, "close fixture")
	if err == nil {
		t.Fatal("expected close error")
	}
	if !strings.Contains(err.Error(), "close fixture") {
		t.Fatalf("expected close context, got: %v", err)
	}
}

func TestGitHubSourceClonesOfficialRepositoryThenCopiesSkeleton(t *testing.T) {
	// The GitHub source delegates network access to Runner so tests can verify the command offline.
	runner := &recordingRunner{t: t}
	target := filepath.Join(t.TempDir(), "app")

	if err := (GitHubSource{Runner: runner}).CopyTo(context.Background(), target); err != nil {
		t.Fatalf("copy GitHub skeleton: %v", err)
	}

	if len(runner.commands) != 1 {
		t.Fatalf("expected one command, got %d", len(runner.commands))
	}
	command := runner.commands[0]
	if command.Name != "git" {
		t.Fatalf("expected git command, got %q", command.Name)
	}
	wantArgs := []string{"clone", "--depth=1", prismgoRepository}
	for i, want := range wantArgs {
		if command.Args[i] != want {
			t.Fatalf("expected arg %d to be %q, got %q", i, want, command.Args[i])
		}
	}
	if command.Args[3] == "" {
		t.Fatal("expected clone target temporary directory")
	}
	assertFileContent(t, filepath.Join(target, "go.mod"), "module prismgo\n")
	if _, err := os.Stat(filepath.Join(target, ".git")); !os.IsNotExist(err) {
		t.Fatalf("expected cloned .git to be excluded, stat error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, ".github")); !os.IsNotExist(err) {
		t.Fatalf("expected cloned .github to be excluded, stat error: %v", err)
	}
}

func TestGitHubSourceReturnsRunnerError(t *testing.T) {
	// Clone failures are returned unchanged enough for callers to report the underlying git problem.
	want := errors.New("clone failed")
	err := (GitHubSource{Runner: failingRunner{err: want}}).CopyTo(context.Background(), filepath.Join(t.TempDir(), "app"))
	if !errors.Is(err, want) {
		t.Fatalf("expected runner error, got: %v", err)
	}
}

func TestGitHubSourceReportsTempDirFailure(t *testing.T) {
	// Temporary directory failures should stop before invoking the clone runner.
	blocker := filepath.Join(t.TempDir(), "tmp-blocker")
	writeFile(t, blocker, []byte("not a directory"), 0o644)
	t.Setenv("TMPDIR", blocker)
	t.Setenv("TEMP", blocker)
	t.Setenv("TMP", blocker)

	err := (GitHubSource{Runner: failingRunner{err: errors.New("runner should not be called")}}).
		CopyTo(context.Background(), filepath.Join(t.TempDir(), "app"))
	if err == nil {
		t.Fatal("expected temporary directory failure")
	}
	if !strings.Contains(err.Error(), "create temporary skeleton directory") {
		t.Fatalf("expected temporary directory error, got: %v", err)
	}
}

func TestGitHubSourceRequiresRunner(t *testing.T) {
	// GitHub acquisition must receive its command runner explicitly from orchestration.
	err := (GitHubSource{}).CopyTo(context.Background(), filepath.Join(t.TempDir(), "app"))
	if err == nil {
		t.Fatal("expected missing runner to fail")
	}
	if !strings.Contains(err.Error(), "runner") {
		t.Fatalf("expected runner error, got: %v", err)
	}
}

type recordingRunner struct {
	t        *testing.T
	commands []run.Command
}

func (r *recordingRunner) Run(ctx context.Context, cmd run.Command) error {
	r.t.Helper()
	r.commands = append(r.commands, cmd)
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(cmd.Args) != 4 {
		r.t.Fatalf("expected git clone destination arg, got args: %#v", cmd.Args)
	}
	cloneTarget := cmd.Args[3]
	writeFile(r.t, filepath.Join(cloneTarget, "go.mod"), []byte("module prismgo\n"), 0o644)
	writeFile(r.t, filepath.Join(cloneTarget, ".git", "config"), []byte("[core]\n"), 0o644)
	writeFile(r.t, filepath.Join(cloneTarget, ".github", "workflows", "ci.yml"), []byte("name: ci\n"), 0o644)
	return nil
}

type failingRunner struct {
	err error
}

func (r failingRunner) Run(context.Context, run.Command) error {
	return r.err
}

type errAfterContext struct {
	context.Context
	remainingNil int
}

func (c *errAfterContext) Err() error {
	if c.remainingNil > 0 {
		c.remainingNil--
		return nil
	}
	return context.Canceled
}

func writeFile(t *testing.T, path string, content []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent directory for %q: %v", path, err)
	}
	if err := os.WriteFile(path, content, mode); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func createSymlinkOrSkip(t *testing.T, oldname string, newname string) {
	t.Helper()
	if err := os.Symlink(oldname, newname); err != nil {
		t.Skipf("symlink creation is not available in this test environment: %v", err)
	}
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}
	if string(content) != want {
		t.Fatalf("expected %q content %q, got %q", path, want, string(content))
	}
}

func assertFileMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %q: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("expected %q mode %v, got %v", path, want, got)
	}
}
