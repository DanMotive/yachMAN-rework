package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"yachman/internal/services"
)

type Bot struct {
	token   string
	services Services
}

type Services struct {
	User     *services.UserService
	Work     *services.WorkService
	Education *services.EducationService
	City     *services.CityService
	Business *services.BusinessService
	Corp     *services.CorporationService
	Stock    *services.StockService
	Market   *services.MarketService
	Events   *services.EventService
	Trade    *services.TradeService
	Notif    *services.NotificationService
}

func NewBot(token string, svc Services) *Bot {
	return &Bot{token: token, services: svc}
}

// HandleUpdate processes a Telegram update (for webhook mode).
func (b *Bot) HandleUpdate(ctx context.Context, update Update) {
	if update.Message != nil {
		b.handleMessage(ctx, update.Message)
	} else if update.CallbackQuery != nil {
		b.handleCallback(ctx, update.CallbackQuery)
	}
}

func (b *Bot) handleMessage(ctx context.Context, msg *Message) {
	text := msg.Text
	chatID := msg.Chat.ID
	userID := msg.From.ID
	isGroup := msg.Chat.Type == "group" || msg.Chat.Type == "supergroup"

	if !strings.HasPrefix(text, "/") {
		return
	}

	parts := strings.Fields(text)
	cmd := strings.ToLower(parts[0])
	if strings.Contains(cmd, "@") {
		cmd = strings.Split(cmd, "@")[0]
	}
	args := parts[1:]

	switch cmd {
	case "/start":
		b.handleStart(ctx, chatID, userID)
	case "/help":
		b.handleHelp(ctx, chatID, isGroup)
	case "/profile", "/balance":
		b.handleProfile(ctx, chatID, userID)
	case "/jobs":
		if isGroup {
			b.handleJobs(ctx, chatID, userID)
		}
	case "/work":
		if isGroup {
			b.handleWork(ctx, chatID, userID, args)
		}
	case "/cities", "/city":
		if !isGroup {
			b.handleCities(ctx, chatID, userID, args)
		} else {
			b.handleCityInfo(ctx, chatID, userID)
		}
	case "/study":
		b.handleStudy(ctx, chatID, userID, args)
	case "/notifications":
		b.handleNotifications(ctx, chatID, userID)
	case "/daily":
		b.handleDaily(ctx, chatID, userID)
	case "/market":
		if isGroup {
			b.handleMarket(ctx, chatID, userID)
		}
	case "/events":
		b.handleEvents(ctx, chatID)
	default:
		b.sendMessage(chatID, "Неизвестная команда. /help")
	}
}

func (b *Bot) handleStart(ctx context.Context, chatID, userID int64) {
	user, err := b.services.User.GetOrCreateUser(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "Ошибка: "+err.Error())
		return
	}
	text := fmt.Sprintf("👋 Добро пожаловать в ЯчМан!\n\n"+
		"💰 Баланс: %d ₽\n📊 Уровень: %d\n🎯 XP: %d\n\n"+
		"Команды:\n"+
		"/profile — профиль\n"+
		"/daily — ежедневный бонус\n"+
		"/cities — список городов\n"+
		"/help — справка",
		user.Balance, user.GlobalLevel, user.GlobalXP)
	b.sendMessage(chatID, text)
}

func (b *Bot) handleHelp(ctx context.Context, chatID int64, isGroup bool) {
	if isGroup {
		b.sendMessage(chatID,
			"🏗 Команды города:\n"+
				"/work [ID] — начать работу\n"+
				"/jobs — доступные работы\n"+
				"/city — информация о городе\n"+
				"/market — рынок ресурсов\n"+
				"/events — события\n"+
				"/pay @user сумма — перевод")
	} else {
		b.sendMessage(chatID,
			"📱 Личные команды:\n"+
				"/profile — профиль\n"+
				"/balance — баланс\n"+
				"/daily — ежедневный бонус\n"+
				"/cities — города\n"+
				"/study — обучение\n"+
				"/notifications — уведомления")
	}
}

func (b *Bot) handleProfile(ctx context.Context, chatID, userID int64) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "Профиль не найден. /start")
		return
	}
	text := fmt.Sprintf("👤 Профиль\n\n"+
		"💰 Баланс: %d ₽\n📊 Уровень: %d\n🎯 XP: %d\n🔥 Серия дней: %d",
		user.Balance, user.GlobalLevel, user.GlobalXP, user.DailyStreak)
	if user.CityID != nil {
		text += fmt.Sprintf("\n🏙 Город: #%d", *user.CityID)
	}
	if user.CorporationID != nil {
		text += fmt.Sprintf("\n🏢 Корпорация: #%d (%s)", *user.CorporationID, *user.CorporationRole)
	}
	if user.ActiveJob != nil {
		text += fmt.Sprintf("\n🔨 Работа: %s", *user.ActiveJob)
	}
	b.sendMessage(chatID, text)
}

func (b *Bot) handleJobs(ctx context.Context, chatID, userID int64) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil || user.CityID == nil {
		b.sendMessage(chatID, "Сначала вступите в город")
		return
	}
	works, err := b.services.Work.ListWorksByDirection(ctx, "добыча")
	if err != nil || len(works) == 0 {
		b.sendMessage(chatID, "Работы не найдены")
		return
	}
	text := "🔨 Доступные работы (добыча):\n\n"
	limit := 10
	if len(works) < limit {
		limit = len(works)
	}
	for _, w := range works[:limit] {
		marker := "✅"
		if user.GlobalXP < w.RequiredXP {
			marker = "🔒"
		}
		text += fmt.Sprintf("%s %s [%s]\n   XP: %d | ⏱ %d мин | 💰 %d ₽ | +%d XP | +%d ед.\n\n",
			marker, w.ID, w.Name, w.RequiredXP, w.DurationMinutes, w.Payout, w.XPReward, w.ResourceAmount)
	}
	text += "/work ID — начать работу"
	b.sendMessage(chatID, text)
}

func (b *Bot) handleWork(ctx context.Context, chatID, userID int64, args []string) {
	if len(args) == 0 {
		b.sendMessage(chatID, "Использование: /work W001")
		return
	}
	workID := strings.ToUpper(args[0])

	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil || user.CityID == nil {
		b.sendMessage(chatID, "Сначала вступите в город: /cities")
		return
	}

	err = b.services.Work.StartWork(ctx, userID, workID, *user.CityID)
	if err != nil {
		b.sendMessage(chatID, "❌ "+err.Error())
		return
	}

	work, _ := b.services.Work.GetWorkDefinition(ctx, workID)
	b.sendMessage(chatID, fmt.Sprintf("🔨 Работа начата: %s\n⏱ Время: %d мин", work.Name, work.DurationMinutes))
}

func (b *Bot) handleDaily(ctx context.Context, chatID, userID int64) {
	bonus, err := b.services.User.ClaimDailyBonus(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "❌ "+err.Error())
		return
	}
	b.sendMessage(chatID, fmt.Sprintf("🎉 Ежедневный бонус: +%d ₽", bonus))
}

func (b *Bot) handleStudy(ctx context.Context, chatID, userID int64, args []string) {
	if len(args) == 0 {
		programs, _ := b.services.Education.ListPrograms(ctx)
		text := "📚 Программы обучения:\n\n"
		for _, p := range programs {
			text += fmt.Sprintf("• %s [%s] — %d ₽ (XP: %d)\n", p.ID, p.Name, p.Cost, p.RequiredXP)
		}
		text += "\n/study ID — записаться\n/study ID study — пройти урок"
		b.sendMessage(chatID, text)
		return
	}

	programID := args[0]
	if len(args) >= 2 && args[1] == "study" {
		progress, err := b.services.Education.Study(ctx, userID, programID)
		if err != nil {
			b.sendMessage(chatID, "❌ "+err.Error())
			return
		}
		b.sendMessage(chatID, fmt.Sprintf("📖 Урок пройден! Прогресс: %d", progress))
		return
	}

	err := b.services.Education.Enroll(ctx, userID, programID)
	if err != nil {
		b.sendMessage(chatID, "❌ "+err.Error())
		return
	}
	b.sendMessage(chatID, "✅ Записан на курс!")
}

func (b *Bot) handleNotifications(ctx context.Context, chatID, userID int64) {
	notifs, err := b.services.Notif.GetUnread(ctx, userID)
	if err != nil || len(notifs) == 0 {
		b.sendMessage(chatID, "📭 Нет новых уведомлений")
		return
	}
	text := "📬 Уведомления:\n\n"
	for _, n := range notifs {
		text += fmt.Sprintf("• %s: %s\n", n["title"], n["body"])
	}
	b.sendMessage(chatID, text)
	_ = b.services.Notif.MarkRead(ctx, userID)
}

func (b *Bot) handleCities(ctx context.Context, chatID, userID int64, args []string) {
	cities, err := b.services.City.ListPublicCities(ctx)
	if err != nil || len(cities) == 0 {
		b.sendMessage(chatID, "🏙 Публичных городов пока нет")
		return
	}
	text := "🏙 Публичные города:\n\n"
	for _, c := range cities {
		text += fmt.Sprintf("• %s [%s]\n  NPC: %d | Уровень: %s\n  ID: %d | Чат: %d\n\n",
			c.Name, c.Level, c.NPCPopulation, c.Level, c.ID, c.ChatID)
	}
	b.sendMessage(chatID, text)
}

func (b *Bot) handleCityInfo(ctx context.Context, chatID, userID int64) {
	city, err := b.services.City.GetCityByChatID(ctx, chatID)
	if err != nil {
		b.sendMessage(chatID, "Город не зарегистрирован.\nАдминистратор группы: /city register Название")
		return
	}
	text := fmt.Sprintf("🏙 %s\n📊 Уровень: %s\n👥 NPC: %d\n💰 Бюджет: %d ₽\n📈 DP: %d\n🔒 Доступ: %s",
		city.Name, city.Level, city.NPCPopulation, city.Treasury, city.DevelopmentPoints, city.AccessMode)
	b.sendMessage(chatID, text)
}

func (b *Bot) handleMarket(ctx context.Context, chatID, userID int64) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil || user.CityID == nil {
		b.sendMessage(chatID, "Вы не в городе")
		return
	}
	stock, err := b.services.Market.GetCityResources(ctx, *user.CityID)
	if err != nil {
		b.sendMessage(chatID, "Рынок пуст")
		return
	}
	text := "📈 Рынок ресурсов:\n\n"
	for resID, qty := range stock {
		text += fmt.Sprintf("• %s: %d ед.\n", resID, qty)
	}
	b.sendMessage(chatID, text)
}

func (b *Bot) handleEvents(ctx context.Context, chatID int64) {
	events, err := b.services.Events.GetActiveEvents(ctx)
	if err != nil || len(events) == 0 {
		b.sendMessage(chatID, "📭 Нет активных событий")
		return
	}
	text := "🎯 Активные события:\n\n"
	for _, e := range events {
		text += fmt.Sprintf("• [%s] %s\n  %s\n  До: %s\n\n", e.Type, e.Name, e.Description,
			e.EndAt.Format("02.01 15:04"))
	}
	b.sendMessage(chatID, text)
}

func (b *Bot) handleCallback(ctx context.Context, cb *CallbackQuery) {
	b.answerCallback(cb.ID, "")
}

func (b *Bot) sendMessage(chatID int64, text string) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.token)
	data := url.Values{
		"chat_id": {strconv.FormatInt(chatID, 10)},
		"text":    {text},
	}
	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		log.Printf("send message error: %v", err)
		return
	}
	defer resp.Body.Close()
}

func (b *Bot) answerCallback(callbackID, text string) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", b.token)
	data := url.Values{
		"callback_query_id": {callbackID},
		"text":              {text},
	}
	http.PostForm(apiURL, data)
}

// Telegram API types

type Update struct {
	UpdateID_      int            `json:"update_id"`
	Message        *Message       `json:"message"`
	CallbackQuery  *CallbackQuery `json:"callback_query"`
}

type Message struct {
	MessageID int    `json:"message_id"`
	From      *User  `json:"from"`
	Chat      *Chat  `json:"chat"`
	Text      string `json:"text"`
}

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type CallbackQuery struct {
	ID   string `json:"id"`
	From *User  `json:"from"`
}

func (b *Bot) GetUpdatesLongPolling(ctx context.Context) {
	offset := 0
	apiBase := fmt.Sprintf("https://api.telegram.org/bot%s", b.token)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		u := fmt.Sprintf("%s/getUpdates?timeout=30&offset=%d", apiBase, offset)
		resp, err := http.Get(u)
		if err != nil {
			log.Printf("getUpdates error: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var result struct {
			Ok     bool     `json:"ok"`
			Result []Update `json:"result"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			continue
		}
		for _, update := range result.Result {
			b.HandleUpdate(ctx, update)
			offset = update.UpdateID_ + 1
		}
	}
}
