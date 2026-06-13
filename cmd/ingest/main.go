// Command ingest is the standalone source-ingest worker. It loads ONE provider's
// board file (sources/<provider>.yml — passed as the first argument or via
// SOURCES_FILE), fetches each board through that platform's adapter, normalizes the
// postings, and upserts them — enqueuing new ones for enrichment in the same write.
// Run one invocation per provider on a schedule (e.g. cron); each processes its
// boards once and exits, so a slow or throttled provider never blocks the others.
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
	"github.com/strelov1/freehire/internal/enrich"
	"github.com/strelov1/freehire/internal/pipeline"
	"github.com/strelov1/freehire/internal/sources"
)

func main() {
	cfg := config.Load()

	// The board file is one provider's list (sources/<provider>.yml); the provider is
	// its file name. Accept it as the first argument (cron passes it per provider) or
	// via SOURCES_FILE.
	path := os.Getenv("SOURCES_FILE")
	if len(os.Args) > 1 && os.Args[1] != "" {
		path = os.Args[1]
	}
	if path == "" {
		log.Fatal("config: no board file given (pass sources/<provider>.yml as an argument or set SOURCES_FILE)")
	}
	sourceCfg, err := sources.LoadConfig(path)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	registry := sources.All(sources.NewClient())
	// Fail fast before touching the DB: a misconfigured board should not start a run.
	if err := sourceCfg.Validate(registry); err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	runner := pipeline.Runner{
		Registry: registry,
		Store:    newDBStore(pool, enrich.Version),
	}

	stats, err := runner.Run(ctx, sourceCfg.Sources)
	if err != nil {
		log.Fatalf("ingest: %v", err)
	}

	log.Printf("ingest done: provider=%s ingested=%d failed=%d", sourceCfg.Provider, stats.Ingested, stats.Failed)

	// Post-run sweep (job-lifecycle spec): close THIS provider's open jobs unseen for
	// the whole grace window. Scoped to the provider so one provider's run never
	// closes another's jobs. Guarded so a run that ingested nothing (a total crawl
	// outage for this provider) can never mass-close its catalogue.
	if shouldSweep(stats) {
		cutoff := pgtype.Timestamptz{Time: time.Now().Add(-staleAfter), Valid: true}
		closed, err := db.New(pool).CloseUnseenJobs(ctx, db.CloseUnseenJobsParams{
			Source: sourceCfg.Provider,
			Cutoff: cutoff,
		})
		if err != nil {
			log.Fatalf("close stale jobs: %v", err)
		}
		log.Printf("closed %d stale %s jobs (unseen for %s)", closed, sourceCfg.Provider, staleAfter)
	}
}

// staleAfter is the grace window before an unseen job is closed: many crawl cycles
// at the hourly per-provider cadence, so a board failing several runs in a row keeps
// its jobs open.
const staleAfter = 48 * time.Hour

// shouldSweep reports whether the run saw enough of the world to justify closing
// jobs: a run that ingested nothing proves only that the crawl failed.
func shouldSweep(stats pipeline.Stats) bool {
	return stats.Ingested > 0
}
