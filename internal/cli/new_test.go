package cli

import (
	"context"
	"strings"
	"testing"
)

func TestExecuteNewReturnsNotImplementedErrorWithProjectName(t *testing.T) {
	// Execute should parse the project name before returning the Task 1 placeholder error.
	err := Execute(context.Background(), []string{"new", "myapp"})
	if err == nil {
		t.Fatal("expected not implemented error, got nil")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("expected not implemented error, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "myapp") {
		t.Fatalf("expected error to include project name, got %q", err.Error())
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

func TestRootCommandHelpExecutesWithoutError(t *testing.T) {
	// Help should render successfully without invoking any child command behavior.
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"--help"})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("expected root help to execute without error, got %v", err)
	}
}
