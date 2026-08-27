package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"yachman/internal/bot"
	"yachman/internal/config"
	"yachman/internal/db"
	"yachman/internal/services"
	"yachman/internal/webapp"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	log.SetFlags(0)
	log.SetOutput(logger)

	if cfg.DatabaseURL != "" {
		log.Println("Running database migrations...")
		if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		log.Println("Loading seed data...")
		if err := db.RunSeedData(cfg.DatabaseURL); err != nil {
			log.Fatalf("Seed data failed: %v", err)
		}
	}

	pool, err := db.NewPool(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	defer pool.Close()

	// Initialize services
	ledger := services.NewLedgerService(pool)
	userSvc := services.NewUserService(pool, ledger)
	workSvc := services.NewWorkService(pool, ledger, userSvc)
	eduSvc := services.NewEducationService(pool, ledger)
	citySvc := services.NewCityService(pool)
	bizSvc := services.NewBusinessService(pool, ledger)
	corpSvc := services.NewCorporationService(pool, ledger)
	stockSvc := services.NewStockService(pool, ledger)
	marketSvc := services.NewMarketService(pool)
	eventSvc := services.NewEventService(pool)
	tradeSvc := services.NewTradeService(pool)
	notifSvc := services.NewNotificationService(pool)
	paySvc := services.NewPaymentService(pool, ledger)
	deliverySvc := services.NewNotificationDelivery(pool, cfg.BotToken)

	// Start scheduler
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()		scheduler := services.NewScheduler(services.SchedulerDeps{
			Work: workSvc, Business: bizSvc, Market: marketSvc,
			Corp: corpSvc, Trade: tradeSvc, Notif: notifSvc, Events: eventSvc,
			Delivery: deliverySvc,
		})
	scheduler.Start(ctx)

	// Telegram bot
	if cfg.BotToken != "" {
		b := bot.NewBot(cfg.BotToken, bot.Services{
			User: userSvc, Work: workSvc, Education: eduSvc,
			City: citySvc, Business: bizSvc, Corp: corpSvc,
			Stock: stockSvc, Market: marketSvc, Events: eventSvc,
			Trade: tradeSvc, Notif: notifSvc, Payment: paySvc,
		}, pool)
		go b.GetUpdatesLongPolling(ctx)
		log.Println("Telegram bot started (long polling)")
	}

	// HTTP router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/health"))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("YachMan server is running"))
	})

	wh := webapp.NewHandler(webapp.Services{
		User: userSvc, City: citySvc, Market: marketSvc,
		Events: eventSvc, Work: workSvc,
	})
	wh.RegisterRoutes(r)

	addr := fmt.Sprintf(":%d", cfg.ServerPort)
	log.Printf("Server starting on %s", addr)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()
		os.Exit(0)
	}()

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
