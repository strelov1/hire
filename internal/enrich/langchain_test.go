package enrich

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tmc/langchaingo/llms"
)

// blockingLLM hangs until its context is cancelled, modelling a stalled gateway.
type blockingLLM struct{}

func (blockingLLM) GenerateContent(ctx context.Context, _ []llms.MessageContent, _ ...llms.CallOption) (*llms.ContentResponse, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func (blockingLLM) Call(context.Context, string, ...llms.CallOption) (string, error) { return "", nil }

// A stalled gateway must not hang the worker: the provider's own timeout cancels
// the call so Enrich returns an error instead of blocking forever.
func TestEnrichTimesOutOnStalledModel(t *testing.T) {
	p := &LangChainProvider{llm: blockingLLM{}, timeout: 20 * time.Millisecond}

	done := make(chan error, 1)
	go func() {
		_, err := p.Enrich(context.Background(), JobInput{Description: "x"})
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Enrich returned nil error, want a timeout error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Enrich did not return; the per-call timeout did not fire")
	}
}

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
