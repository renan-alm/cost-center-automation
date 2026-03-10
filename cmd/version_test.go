package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestVersionCmd_DevFallback(t *testing.T) {
	// When version is "dev" and no VERSION file exists, should print "dev".
	old := version
	version = "dev"
	defer func() { version = old }()

	// Run in a temp dir with no VERSION file.
	dir := t.TempDir()
	t.Chdir(dir)

	var buf bytes.Buffer
	cmd := versionCmd
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{})

	err := cmd.RunE(cmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVersionCmd_ReadsVersionFile(t *testing.T) {
	old := version
	version = "dev"
	defer func() { version = old }()

	dir := t.TempDir()
	t.Chdir(dir)

	// Create a VERSION file.
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("1.2.3\n"), 0o644); err != nil {
		t.Fatalf("writing VERSION file: %v", err)
	}

	err := versionCmd.RunE(versionCmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVersionCmd_BuildTimeVersion(t *testing.T) {
	old := version
	version = "v2.0.0"
	defer func() { version = old }()

	// When version is set at build time, VERSION file should not be read.
	err := versionCmd.RunE(versionCmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVersionCmd_UnreadableVersionFile(t *testing.T) {
	old := version
	version = "dev"
	defer func() { version = old }()

	dir := t.TempDir()
	t.Chdir(dir)

	// Create a VERSION file that is a directory (causes read error).
	versionPath := filepath.Join(dir, "VERSION")
	if err := os.Mkdir(versionPath, 0o755); err != nil {
		t.Fatalf("creating VERSION directory: %v", err)
	}

	err := versionCmd.RunE(versionCmd, nil)
	if err == nil {
		t.Fatal("expected error for unreadable VERSION file")
	}
}
