package run

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Command describes one external command invocation and its working directory.
type Command struct {
	Name string
	Args []string
	Dir  string
}

// Runner executes external commands for installer steps that must call system tools.
type Runner interface {
	Run(ctx context.Context, cmd Command) error
}

// OutputRunner executes external commands whose stdout is needed by the caller.
type OutputRunner interface {
	Runner
	Output(ctx context.Context, cmd Command) ([]byte, error)
}

// OSRunner executes commands on the host operating system.
type OSRunner struct{}

// Run executes cmd and includes combined stdout/stderr output in any returned error.
func (OSRunner) Run(ctx context.Context, cmd Command) error {
	execCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)
	execCmd.Dir = cmd.Dir

	output, err := execCmd.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			return fmt.Errorf("run %s: %w: %s", commandString(cmd), err, strings.TrimSpace(string(output)))
		}
		return fmt.Errorf("run %s: %w", commandString(cmd), err)
	}
	return nil
}

// Output executes cmd and returns its output, preserving command output in returned errors.
func (OSRunner) Output(ctx context.Context, cmd Command) ([]byte, error) {
	execCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)
	execCmd.Dir = cmd.Dir

	output, err := execCmd.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			return nil, fmt.Errorf("run %s: %w: %s", commandString(cmd), err, strings.TrimSpace(string(output)))
		}
		return nil, fmt.Errorf("run %s: %w", commandString(cmd), err)
	}
	return output, nil
}

func commandString(cmd Command) string {
	// The command string is for diagnostics only; execution uses structured name and args above.
	parts := append([]string{cmd.Name}, cmd.Args...)
	return strings.Join(parts, " ")
}
