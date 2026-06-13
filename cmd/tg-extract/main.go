// Command tg-extract is the standalone Telegram extraction worker. It drains the
// telegram_posts queue: for each claimed post it asks the LLM to classify the post
// and extract its vacancies, validates the payload, and writes the jobs through
// the canonical upsert — enqueuing them for enrichment in the same transaction as
// marking the post extracted. Run it on a schedule (e.g. cron); it processes a
// bounded batch and exits.
package main

import (
	"context"
	"log"
	"os"

	"github.com/strelov1/freehire/internal/config"
	"github.com/strelov1/freehire/internal/database"
	"github.com/strelov1/freehire/internal/telegram"
)

func main() {
	cfg := config.Load()
	ecfg, err := config.LoadEnrich()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// sources/telegram.yml supplies each channel's kind, steering the extraction prompt.
	path := os.Getenv("CHANNELS_FILE")
	if path == "" {
		path = "sources/telegram.yml"
	}
	chanCfg, err := telegram.LoadConfig(path)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if err := chanCfg.Validate(); err != nil {
		log.Fatalf("config: %v", err)
	}
	kinds := make(map[string]telegram.Kind, len(chanCfg.Channels))
	for _, e := range chanCfg.Channels {
		kinds[e.Channel] = e.Kind
	}

	ctx := context.Background()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	extractor, err := telegram.NewLangChainExtractor(ecfg.LLMBaseURL, ecfg.LLMAPIKey, ecfg.LLMModel)
	if err != nil {
		log.Fatalf("extractor: %v", err)
	}

	runner := telegram.ExtractRunner{
		Extractor: extractor,
		Store:     newExtractStore(pool),
		Kinds:     kinds,
	}

	stats, err := runner.Run(ctx)
	if err != nil {
		log.Fatalf("extract: %v", err)
	}
	log.Printf("tg-extract done: processed=%d jobs=%d failed=%d",
		stats.Processed, stats.Jobs, stats.Failed)
}
