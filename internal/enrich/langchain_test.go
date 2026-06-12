package enrich

import (
	"strings"
	"testing"
)

// The system prompt must pin the region (reach) vocabulary it asks the model to
// use, drawn from the same list Validate enforces, so prompt and validator
// cannot drift.
func TestSystemPromptIncludesRegionVocabulary(t *testing.T) {
	p := buildSystemPrompt()

	if !strings.Contains(p, "regions") {
		t.Errorf("prompt must mention regions, got:\n%s", p)
	}
	for _, v := range RegionValues {
		if !strings.Contains(p, v) {
			t.Errorf("prompt must list region value %q", v)
		}
	}
}
