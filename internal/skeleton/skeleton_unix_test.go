//go:build unix

package skeleton

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestCopyDirectoryRejectsNonRegularEntries(t *testing.T) {
	// Unix FIFOs exercise non-regular skeleton entries without relying on regular-file behavior.
	source := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(source, "pipe"), 0o644); err != nil {
		t.Fatalf("create fifo: %v", err)
	}

	err := copyDirectory(context.Background(), source, filepath.Join(t.TempDir(), "app"), 0o755)
	if err == nil {
		t.Fatal("expected non-regular entry to fail")
	}
	if !strings.Contains(err.Error(), "non-regular") {
		t.Fatalf("expected non-regular error, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(source, "pipe")); err != nil {
		t.Fatalf("expected fifo fixture to remain stattable: %v", err)
	}
}
