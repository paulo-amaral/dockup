package steps

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMergeDaemonJSONCreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.json")
	changed, backup, err := mergeDaemonJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if backup != "" {
		t.Errorf("no backup expected for a new file, got %q", backup)
	}
	if len(changed) != len(hardenSettings) {
		t.Errorf("expected %d changes, got %d: %v", len(hardenSettings), len(changed), changed)
	}
	assertKeys(t, path, "log-driver", "log-opts", "live-restore", "no-new-privileges")
}

func TestMergeDaemonJSONPreservesExistingKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.json")
	orig := `{"data-root": "/data/docker", "live-restore": false}`
	if err := os.WriteFile(path, []byte(orig), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, backup, err := mergeDaemonJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if backup == "" {
		t.Error("expected a backup path for an existing file")
	} else if b, err := os.ReadFile(backup); err != nil || string(b) != orig {
		t.Errorf("backup should hold the original content, got %q (err %v)", b, err)
	}
	if len(changed) == 0 {
		t.Error("live-restore=false should have been changed to true")
	}

	cfg := readJSON(t, path)
	if cfg["data-root"] != "/data/docker" {
		t.Errorf("unrelated key data-root was lost: %v", cfg)
	}
	if cfg["live-restore"] != true {
		t.Errorf("live-restore not enforced: %v", cfg["live-restore"])
	}
}

func TestMergeDaemonJSONIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.json")
	if _, _, err := mergeDaemonJSON(path); err != nil {
		t.Fatal(err)
	}
	changed, backup, err := mergeDaemonJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 0 || backup != "" {
		t.Errorf("second run must be a no-op, got changed=%v backup=%q", changed, backup)
	}
}

func TestMergeDaemonJSONRejectsInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := mergeDaemonJSON(path); err == nil {
		t.Fatal("invalid existing JSON must abort, not be overwritten")
	}
	if b, _ := os.ReadFile(path); string(b) != "{broken" {
		t.Error("original file must remain untouched on abort")
	}
}

func assertKeys(t *testing.T, path string, keys ...string) {
	t.Helper()
	cfg := readJSON(t, path)
	for _, k := range keys {
		if _, ok := cfg[k]; !ok {
			t.Errorf("missing key %q in %v", k, cfg)
		}
	}
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := map[string]any{}
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}
