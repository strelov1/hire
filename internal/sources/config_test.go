package sources

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfig(t *testing.T) {
	data := []byte(`
- company: Cohere
  board: cohere
- company: Stripe
  board: stripe
`)

	cfg, err := ParseConfig("greenhouse", data)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if cfg.Provider != "greenhouse" {
		t.Errorf("Provider = %q, want greenhouse", cfg.Provider)
	}
	if len(cfg.Sources) != 2 {
		t.Fatalf("len(Sources) = %d, want 2", len(cfg.Sources))
	}
	want := CompanyEntry{Company: "Cohere", Provider: "greenhouse", Board: "cohere"}
	if cfg.Sources[0] != want {
		t.Errorf("Sources[0] = %+v, want %+v", cfg.Sources[0], want)
	}
}

// LoadConfig takes the provider from the file name, so the board file never repeats
// it per entry.
func TestLoadConfigInfersProviderFromFilename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ashby.yml")
	if err := os.WriteFile(path, []byte("- company: Vercel\n  board: vercel\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Provider != "ashby" {
		t.Errorf("Provider = %q, want ashby (from file name)", cfg.Provider)
	}
	if len(cfg.Sources) != 1 || cfg.Sources[0].Provider != "ashby" {
		t.Errorf("Sources = %+v, want one ashby entry", cfg.Sources)
	}
}

func TestConfigValidateRejectsUnknownProvider(t *testing.T) {
	cfg := Config{Provider: "myspace", Sources: []CompanyEntry{{Company: "Acme", Board: "acme"}}}

	err := cfg.Validate(reg(fakeSource{"greenhouse"}))
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
	if !strings.Contains(err.Error(), "myspace") {
		t.Errorf("error %q should name the unknown provider", err.Error())
	}
}

func TestConfigValidateRejectsEmptyBoard(t *testing.T) {
	cfg := Config{Provider: "greenhouse", Sources: []CompanyEntry{{Company: "Cohere", Board: ""}}}

	err := cfg.Validate(reg(fakeSource{"greenhouse"}))
	if err == nil {
		t.Fatal("expected error for empty board, got nil")
	}
	if !strings.Contains(err.Error(), "Cohere") {
		t.Errorf("error %q should name the offending company", err.Error())
	}
}

func TestConfigValidateRejectsEmptyCompany(t *testing.T) {
	cfg := Config{Provider: "greenhouse", Sources: []CompanyEntry{{Company: "", Board: "cohere"}}}

	if err := cfg.Validate(reg(fakeSource{"greenhouse"})); err == nil {
		t.Fatal("expected error for empty company, got nil")
	}
}

func TestConfigValidateAcceptsKnownProviders(t *testing.T) {
	cfg := Config{Provider: "greenhouse", Sources: []CompanyEntry{{Company: "Cohere", Board: "cohere"}}}

	if err := cfg.Validate(reg(fakeSource{"greenhouse"})); err != nil {
		t.Errorf("Validate: unexpected error %v", err)
	}
}
