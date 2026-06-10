package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/strelov1/hire/internal/config"
	"github.com/strelov1/hire/internal/database"
	"github.com/strelov1/hire/internal/handler"
)

func main() {
	cfg := config.Load()

	// Never boot the auth surface with a guessable signing key.
	if cfg.JWTSecret == "" {
		log.Fatal("config: JWT_SECRET is required")
	}

	pool, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	app := fiber.New(fiber.Config{
		AppName:      "hire",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorHandler: handler.ErrorHandler,
	})

	app.Use(recover.New())
	app.Use(logger.New())

	handler.Register(app, pool, cfg.FrontendOrigin, cfg.JWTSecret, cfg.JWTTTL, cfg.CookieSecure)

	// Run the server in a goroutine so main can wait for a shutdown signal.
	// Fiber's Listen returns nil on graceful shutdown, so any error is fatal.
	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil {
			log.Fatalf("listen: %v", err)
		}
	}()
	log.Printf("hire listening on :%s", cfg.Port)

	// Graceful shutdown on SIGINT/SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
