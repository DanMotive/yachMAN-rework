package services

import (
	"context"
	"log"
	"time"
)

type Scheduler struct {
	work       *WorkService
	business   *BusinessService
	market     *MarketService
	corp       *CorporationService
	trade      *TradeService
	notif      *NotificationService
	events     *EventService
}

type SchedulerDeps struct {
	Work     *WorkService
	Business *BusinessService
	Market   *MarketService
	Corp     *CorporationService
	Trade    *TradeService
	Notif    *NotificationService
	Events   *EventService
}

func NewScheduler(deps SchedulerDeps) *Scheduler {
	return &Scheduler{
		work:     deps.Work,
		business: deps.Business,
		market:   deps.Market,
		corp:     deps.Corp,
		trade:    deps.Trade,
		notif:    deps.Notif,
		events:   deps.Events,
	}
}

// Start launches all scheduler loops.
func (s *Scheduler) Start(ctx context.Context) {
	go s.minuteLoop(ctx)
	go s.hourlyLoop(ctx)
	go s.sixHourLoop(ctx)
	go s.dailyLoop(ctx)
	log.Println("Scheduler started")
}

// Every minute: complete expired works, deliver notifications
func (s *Scheduler) minuteLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if n, err := s.work.FinishExpiredWorks(ctx); err == nil && n > 0 {
				log.Printf("[scheduler] completed %d work runs", n)
			}
			if n, err := s.notif.DeliverPending(ctx); err == nil && n > 0 {
				log.Printf("[scheduler] expired %d old notifications", n)
			}
		}
	}
}

// Every hour: business ticks, trade contracts, salary, market prices
func (s *Scheduler) hourlyLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if n, err := s.business.ProcessBusinessTicks(ctx); err == nil && n > 0 {
				log.Printf("[scheduler] processed %d business ticks", n)
			}
			if n, err := s.trade.ExecuteTradeContracts(ctx); err == nil && n > 0 {
				log.Printf("[scheduler] executed %d trade contracts", n)
			}
			_ = s.corp.PaySalaries(ctx)
			_ = s.market.UpdateAllPrices(ctx)
		}
	}
}

// Every 6 hours: NPC population, DP, market prices, corp valuation
func (s *Scheduler) sixHourLoop(ctx context.Context) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.market.UpdateAllPrices(ctx)
			log.Println("[scheduler] 6h tick: prices updated")
		}
	}
}

// Daily: daily bonus reset, global summary
func (s *Scheduler) dailyLoop(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.Println("[scheduler] daily tick complete")
		}
	}
}
