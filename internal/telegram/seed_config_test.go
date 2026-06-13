package telegram

import (
	"path/filepath"
	"testing"
)

// The committed sources/telegram.yml must always load and validate — a broken seed
// file would otherwise only surface at the next scheduled crawl.
func TestSeedChannelsFileIsValid(t *testing.T) {
	path, err := filepath.Abs(filepath.Join("..", "..", "sources", "telegram.yml"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(cfg.Channels) < 30 {
		t.Errorf("channels = %d, want the curated tier-1 list (>=30)", len(cfg.Channels))
	}
}
