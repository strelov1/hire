package sources

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is one provider's parsed board file: the boards to crawl, all sharing the
// provider taken from the file name.
type Config struct {
	Provider string
	Sources  []CompanyEntry
}

// LoadConfig reads a per-provider board file (e.g. sources/greenhouse.yml). The
// provider is the file's base name without extension; the file itself is a flat list
// of company + board entries, so the provider is never repeated per line.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("sources: read config %s: %w", path, err)
	}
	provider := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return ParseConfig(provider, data)
}

// ParseConfig parses a provider's board-list bytes, stamping each entry with the
// provider so the rest of the pipeline still sees a fully-populated CompanyEntry.
func ParseConfig(provider string, data []byte) (Config, error) {
	var entries []CompanyEntry
	if err := yaml.Unmarshal(data, &entries); err != nil {
		return Config{}, fmt.Errorf("sources: parse config: %w", err)
	}
	for i := range entries {
		entries[i].Provider = provider
	}
	return Config{Provider: provider, Sources: entries}, nil
}

// Validate checks the file's provider is registered and every entry is complete, so
// the ingest command fails fast instead of silently skipping a misconfigured board.
func (c Config) Validate(registry map[string]Source) error {
	if _, ok := registry[c.Provider]; !ok {
		return fmt.Errorf("sources: unknown provider %q (from the file name)", c.Provider)
	}
	for _, e := range c.Sources {
		if e.Company == "" {
			return fmt.Errorf("sources: %s entry has empty company", c.Provider)
		}
		if e.Board == "" {
			return fmt.Errorf("sources: %s entry for company %q has empty board", c.Provider, e.Company)
		}
	}
	return nil
}
