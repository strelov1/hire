package telegram

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeExtractor struct {
	result  Extraction
	err     error
	prompts []Kind
}

func (f *fakeExtractor) Extract(_ context.Context, _ string, kind Kind) (Extraction, error) {
	f.prompts = append(f.prompts, kind)
	if f.err != nil {
		return Extraction{}, f.err
	}
	return f.result, nil
}

type completion struct {
	post PendingPost
	jobs []ExtractedJob
}

type fakeExtractStore struct {
	pending   []PendingPost
	completed []completion
	failures  []string
}

func (s *fakeExtractStore) Claim(_ context.Context, _ int32, batch int32) ([]PendingPost, error) {
	n := int(batch)
	if n > len(s.pending) {
		n = len(s.pending)
	}
	out := s.pending[:n]
	s.pending = s.pending[n:]
	return out, nil
}

func (s *fakeExtractStore) Complete(_ context.Context, post PendingPost, jobs []ExtractedJob) error {
	s.completed = append(s.completed, completion{post: post, jobs: jobs})
	return nil
}

func (s *fakeExtractStore) Fail(_ context.Context, post PendingPost, msg string) error {
	s.failures = append(s.failures, post.Channel+": "+msg)
	return nil
}

func pendingPost() PendingPost {
	return PendingPost{
		Channel:  "hrlunapark",
		MsgID:    392,
		Text:     "tl;dr: ML & full-stack engineers, $110k-220k, London ...",
		PostedAt: time.Date(2026, 5, 28, 12, 3, 7, 0, time.UTC),
	}
}

func kinds() map[string]Kind {
	return map[string]Kind{"hrlunapark": KindAuthored}
}

func TestExtractCompletesWithExtractedJobs(t *testing.T) {
	ex := &fakeExtractor{result: Extraction{Jobs: []ExtractedJob{
		{Title: "ML Engineer", Company: "Claimsorted", Description: "AI claims workflows, $120k-220k"},
		{Title: "Full-stack Engineer", Company: "Claimsorted", Description: "Next.js/Node, $110k-200k"},
	}}}
	store := &fakeExtractStore{pending: []PendingPost{pendingPost()}}
	r := ExtractRunner{Extractor: ex, Store: store, Kinds: kinds()}

	stats, err := r.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if stats.Processed != 1 || stats.Jobs != 2 || stats.Failed != 0 {
		t.Errorf("stats = %+v, want Processed=1 Jobs=2 Failed=0", stats)
	}
	if len(store.completed) != 1 || len(store.completed[0].jobs) != 2 {
		t.Fatalf("completed = %+v, want one completion with 2 jobs", store.completed)
	}
	if ex.prompts[0] != KindAuthored {
		t.Errorf("extractor got kind %q, want authored (from config)", ex.prompts[0])
	}
}

func TestExtractZeroJobsIsANormalCompletion(t *testing.T) {
	ex := &fakeExtractor{result: Extraction{}}
	store := &fakeExtractStore{pending: []PendingPost{pendingPost()}}
	r := ExtractRunner{Extractor: ex, Store: store, Kinds: kinds()}

	stats, err := r.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if stats.Processed != 1 || stats.Jobs != 0 || stats.Failed != 0 {
		t.Errorf("stats = %+v, want Processed=1 Jobs=0 Failed=0", stats)
	}
	if len(store.completed) != 1 || len(store.completed[0].jobs) != 0 {
		t.Errorf("want a zero-job completion, got %+v", store.completed)
	}
}

func TestExtractInvalidPayloadIsFailedNotPersisted(t *testing.T) {
	ex := &fakeExtractor{result: Extraction{Jobs: []ExtractedJob{{Title: ""}}}}
	store := &fakeExtractStore{pending: []PendingPost{pendingPost()}}
	r := ExtractRunner{Extractor: ex, Store: store, Kinds: kinds()}

	stats, err := r.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if stats.Failed != 1 || stats.Processed != 0 {
		t.Errorf("stats = %+v, want Failed=1 Processed=0", stats)
	}
	if len(store.completed) != 0 {
		t.Errorf("invalid payload was persisted: %+v", store.completed)
	}
	if len(store.failures) != 1 {
		t.Errorf("failures = %v, want 1", store.failures)
	}
}

func TestExtractLLMErrorIsFailed(t *testing.T) {
	ex := &fakeExtractor{err: errors.New("llm down")}
	store := &fakeExtractStore{pending: []PendingPost{pendingPost()}}
	r := ExtractRunner{Extractor: ex, Store: store, Kinds: kinds()}

	stats, err := r.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if stats.Failed != 1 {
		t.Errorf("stats = %+v, want Failed=1", stats)
	}
}

func TestExtractUnknownChannelKindDefaultsToBoard(t *testing.T) {
	ex := &fakeExtractor{result: Extraction{}}
	store := &fakeExtractStore{pending: []PendingPost{{Channel: "unlisted", MsgID: 1, Text: "x", PostedAt: time.Now()}}}
	r := ExtractRunner{Extractor: ex, Store: store, Kinds: kinds()}

	if _, err := r.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(ex.prompts) != 1 || ex.prompts[0] != KindBoard {
		t.Errorf("kind = %v, want board fallback for a channel no longer configured", ex.prompts)
	}
}
