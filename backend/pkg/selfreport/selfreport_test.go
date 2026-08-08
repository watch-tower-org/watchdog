package selfreport

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/watch-tower-org/watchdog/backend/internal/config"
)

func TestResolveKeyPrefersExplicitOverride(t *testing.T) {
	key, fromFile, err := resolveKey("wt_override", "/nonexistent", os.ReadFile, func() (string, error) {
		t.Fatal("provision should not be called when override is set")
		return "", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if key != "wt_override" || fromFile {
		t.Fatalf("key=%q fromFile=%v", key, fromFile)
	}
}

func TestResolveKeyReadsExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key")
	if err := os.WriteFile(path, []byte("wt_file_key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	key, fromFile, err := resolveKey("", path, os.ReadFile, func() (string, error) {
		t.Fatal("provision should not be called when file exists")
		return "", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if key != "wt_file_key" || !fromFile {
		t.Fatalf("key=%q fromFile=%v", key, fromFile)
	}
}

func TestResolveKeyEmptyFileFallsBackToProvision(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key")
	if err := os.WriteFile(path, []byte("   \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	key, fromFile, err := resolveKey("", path, os.ReadFile, func() (string, error) {
		return "wt_provisioned", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if key != "wt_provisioned" || fromFile {
		t.Fatalf("key=%q fromFile=%v", key, fromFile)
	}
}

func TestResolveKeyProvisionErrorPropagates(t *testing.T) {
	_, _, err := resolveKey("", "", os.ReadFile, func() (string, error) {
		return "", errors.New("db down")
	})
	if err == nil {
		t.Fatal("expected error from provision to propagate")
	}
}

func TestWriteKeyFileCreatesDirsAndPerms(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", keyFileName)
	if err := writeKeyFile(path, "wt_secret"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "wt_secret\n" {
		t.Fatalf("content = %q", string(data))
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("permissions = %v, want 0600", info.Mode().Perm())
	}
}

func TestReporterNilSafe(t *testing.T) {
	var r *Reporter
	r.Report(errors.New("boom"))
	r.ReportPanic("boom")
	r.Recover("boom", map[string]any{"url": "/x"})
	r.Close()
}

func TestReporterDisabledReturnsNil(t *testing.T) {
	cfg := config.SelfReportConfig{Enabled: false}
	r, err := New(cfg, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if r != nil {
		t.Fatalf("expected nil reporter, got %+v", r)
	}
}
