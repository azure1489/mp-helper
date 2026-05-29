package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTmp(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadYAML(t *testing.T) {
	p := writeTmp(t, "listen: \":9000\"\nadmin_token: \"secret\"\n")
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != ":9000" || cfg.AdminToken != "secret" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
	if cfg.DBPath != "./mp-helper.db" {
		t.Fatalf("default db_path not applied: %q", cfg.DBPath)
	}
}

func TestEnvOverride(t *testing.T) {
	p := writeTmp(t, "admin_token: \"fromfile\"\n")
	t.Setenv("MP_HELPER_ADMIN_TOKEN", "fromenv")
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AdminToken != "fromenv" {
		t.Fatalf("env did not override: %q", cfg.AdminToken)
	}
}

func TestValidateAdminTokenRequired(t *testing.T) {
	p := writeTmp(t, "listen: \":8080\"\n")
	if _, err := Load(p); err == nil {
		t.Fatal("expected error for missing admin_token")
	}
}

func TestValidateCacheType(t *testing.T) {
	p := writeTmp(t, "admin_token: \"x\"\ncache:\n  type: redis\n")
	if _, err := Load(p); err == nil {
		t.Fatal("expected error for unsupported cache type")
	}
}
