package run

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestOSRunnerRunsCommandSuccessfully(t *testing.T) {
	// The test process helper avoids depending on shell availability or platform-specific commands.
	err := (OSRunner{}).Run(context.Background(), Command{
		Name: os.Args[0],
		Args: []string{"-test.run=TestHelperProcess", "--", "success"},
	})
	if err != nil {
		t.Fatalf("expected command to pass, got error: %v", err)
	}
}

func TestOSRunnerIncludesCombinedOutputInErrors(t *testing.T) {
	// Failed setup commands need stderr/stdout in the error so users can act on the real tool failure.
	err := (OSRunner{}).Run(context.Background(), Command{
		Name: os.Args[0],
		Args: []string{"-test.run=TestHelperProcess", "--", "fail"},
	})
	if err == nil {
		t.Fatal("expected failing command to return an error")
	}
	if !strings.Contains(err.Error(), "command output") {
		t.Fatalf("expected command output in error, got: %v", err)
	}
}

func TestOSRunnerHandlesFailureWithoutOutput(t *testing.T) {
	// Some tools fail before writing output; those errors should still include the command identity.
	err := (OSRunner{}).Run(context.Background(), Command{
		Name: os.Args[0],
		Args: []string{"-test.run=TestHelperProcess", "--", "fail-silent"},
	})
	if err == nil {
		t.Fatal("expected failing command to return an error")
	}
	if !strings.Contains(err.Error(), os.Args[0]) {
		t.Fatalf("expected command name in error, got: %v", err)
	}
}

func TestOSRunnerOutputReturnsCommandOutput(t *testing.T) {
	// Version resolution needs command output without callers knowing about os/exec details.
	output, err := (OSRunner{}).Output(context.Background(), Command{
		Name: os.Args[0],
		Args: []string{"-test.run=TestHelperProcess", "--", "success"},
	})
	if err != nil {
		t.Fatalf("expected command to pass, got error: %v", err)
	}
	if string(output) != "command output\n" {
		t.Fatalf("output = %q, want command output", string(output))
	}
}

func TestOSRunnerOutputIncludesCombinedOutputInErrors(t *testing.T) {
	// Failed output commands should include stdout/stderr so version resolution errors stay actionable.
	_, err := (OSRunner{}).Output(context.Background(), Command{
		Name: os.Args[0],
		Args: []string{"-test.run=TestHelperProcess", "--", "fail"},
	})
	if err == nil {
		t.Fatal("expected failing command to return an error")
	}
	if !strings.Contains(err.Error(), "command output") {
		t.Fatalf("expected command output in error, got: %v", err)
	}
}

func TestOSRunnerOutputHandlesFailureWithoutOutput(t *testing.T) {
	// Silent version-resolution failures should still include the command identity.
	_, err := (OSRunner{}).Output(context.Background(), Command{
		Name: os.Args[0],
		Args: []string{"-test.run=TestHelperProcess", "--", "fail-silent"},
	})
	if err == nil {
		t.Fatal("expected failing command to return an error")
	}
	if !strings.Contains(err.Error(), os.Args[0]) {
		t.Fatalf("expected command name in error, got: %v", err)
	}
}

func TestHelperProcess(t *testing.T) {
	if len(os.Args) < 4 || os.Args[len(os.Args)-2] != "--" {
		return
	}

	switch os.Args[len(os.Args)-1] {
	case "success":
		if _, err := os.Stdout.WriteString("command output\n"); err != nil {
			os.Exit(9)
		}
		os.Exit(0)
	case "fail":
		if _, err := os.Stdout.WriteString("command output\n"); err != nil {
			os.Exit(9)
		}
		os.Exit(7)
	case "fail-silent":
		os.Exit(8)
	}
}
