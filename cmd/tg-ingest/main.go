// Command tg-ingest is the standalone Telegram crawl worker. It loads the
// configured channels from sources/telegram.yml, fetches each channel's latest posts from
// the public t.me web preview, prefilters obvious non-vacancies, and stores new
// posts in the telegram_posts queue for the extraction worker (cmd/tg-extract).
// Run it on a schedule (e.g. cron); it crawls every channel once and exits.
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/strelov1/freehire/internal/config"
	"github.com/strelov1/freehire/internal/database"
	"github.com/strelov1/freehire/internal/db"
	"github.com/strelov1/freehire/internal/telegram"
)

func main() {
	cfg := config.Load()

	path := os.Getenv("CHANNELS_FILE")
	if path == "" {
		path = "sources/telegram.yml"
	}
	chanCfg, err := telegram.LoadConfig(path)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	// Fail fast before touching the DB: a misconfigured channel should not start a run.
	if err := chanCfg.Validate(); err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	runner := telegram.CrawlRunner{
		Fetcher: telegram.NewFetcher(),
		Store:   &postStore{q: db.New(pool)},
		Delay:   2 * time.Second, // polite pacing toward t.me
	}

	stats, err := runner.Run(ctx, chanCfg.Channels)
	if err != nil {
		log.Fatalf("crawl: %v", err)
	}
	log.Printf("tg-ingest done: stored=%d filtered=%d failed=%d",
		stats.Stored, stats.Filtered, stats.Failed)
}

// postStore adapts the generated queries to telegram.PostStore.
type postStore struct {
	q *db.Queries
}

func (s *postStore) Insert(ctx context.Context, channel string, p telegram.Post, done bool) (bool, error) {
	var extractedAt pgtype.Timestamptz
	if done {
		extractedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	}
	rows, err := s.q.InsertTelegramPost(ctx, db.InsertTelegramPostParams{
		Channel:     channel,
		MsgID:       p.MsgID,
		Text:        p.Text,
		PostedAt:    pgtype.Timestamptz{Time: p.PostedAt, Valid: true},
		ExtractedAt: extractedAt,
	})
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

var _ telegram.PostStore = (*postStore)(nil)
