package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"yachman/internal/enums"
	"yachman/internal/services"
)

// ────────────────────────────────────────────────────────────────────────────
// Types
// ────────────────────────────────────────────────────────────────────────────

type Bot struct {
	token   string
	services Services
	db      *pgxpool.Pool
}

type Services struct {
	User      *services.UserService
	Work      *services.WorkService
	Education *services.EducationService
	City      *services.CityService
	Business  *services.BusinessService
	Corp      *services.CorporationService
	Stock     *services.StockService
	Market    *services.MarketService
	Events    *services.EventService
	Trade     *services.TradeService
	Notif     *services.NotificationService
	Payment   *services.PaymentService
}

type InlineButton struct {
	Text string
	Data string
}

func NewBot(token string, svc Services, db *pgxpool.Pool) *Bot {
	return &Bot{token: token, services: svc, db: db}
}

// ────────────────────────────────────────────────────────────────────────────
// Telegram API types
// ────────────────────────────────────────────────────────────────────────────

type Update struct {
	UpdateID_     int            `json:"update_id"`
	Message       *TGMessage     `json:"message"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

type TGMessage struct {
	MessageID int    `json:"message_id"`
	From      *TGUser `json:"from"`
	Chat      *TGChat `json:"chat"`
	Text      string  `json:"text"`
}

type TGUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type TGChat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type CallbackQuery struct {
	ID       string   `json:"id"`
	From     *TGUser  `json:"from"`
	Data     string   `json:"data"`
	Message  *TGMessage `json:"message"`
}

// ────────────────────────────────────────────────────────────────────────────
// Entry points
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) HandleUpdate(ctx context.Context, update Update) {
	if update.Message != nil {
		b.handleMessage(ctx, update.Message)
	} else if update.CallbackQuery != nil {
		b.handleCallback(ctx, update.CallbackQuery)
	}
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
			if b.isUpdateProcessed(ctx, update.UpdateID_) {
				offset = update.UpdateID_ + 1
				continue
			}
			b.markUpdateProcessed(ctx, update.UpdateID_)
			b.HandleUpdate(ctx, update)
			offset = update.UpdateID_ + 1
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Message routing
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) handleMessage(ctx context.Context, msg *TGMessage) {
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
		if isGroup {
			b.showGroupHelp(ctx, chatID)
		} else {
			b.showMainMenu(ctx, chatID, userID)
		}
	case "/help":
		if isGroup {
			b.showGroupHelp(ctx, chatID)
		} else {
			b.showMainMenu(ctx, chatID, userID)
		}

	// ── DM commands ──────────────────────────────────
	case "/profile", "/balance":
		b.showProfile(ctx, chatID, userID, false)
	case "/daily":
		b.showDaily(ctx, chatID, userID, false)
	case "/cities":
		if isGroup {
			b.showGroupHelp(ctx, chatID)
		} else {
			b.showCitiesList(ctx, chatID, userID)
		}
	case "/study":
		if isGroup {
			b.showGroupHelp(ctx, chatID)
		} else {
			b.showStudyMenu(ctx, chatID, userID)
		}
	case "/notifications":
		if isGroup {
			b.showGroupHelp(ctx, chatID)
		} else {
			b.showNotifications(ctx, chatID, userID)
		}
	case "/vip":
		if isGroup {
			b.showGroupHelp(ctx, chatID)
		} else {
			b.showVip(ctx, chatID, userID, false)
		}

	// ── Group commands ───────────────────────────────
	case "/work":
		if isGroup {
			b.showWorkDirections(ctx, chatID, userID)
		}
	case "/jobs":
		if isGroup {
			b.showWorkDirections(ctx, chatID, userID)
		} else {
			// In DM, show active work if any
			b.showActiveWork(ctx, chatID, userID)
		}
	case "/city":
		if isGroup {
			b.handleCityGroup(ctx, chatID, userID, args)
		} else {
			// In DM: if args have city ID, show city info; otherwise cities list
			if len(args) > 0 {
				b.showCityDetailFromID(ctx, chatID, userID, args[0])
			} else {
				b.showCitiesList(ctx, chatID, userID)
			}
		}
	case "/market":
		if isGroup {
			b.showMarket(ctx, chatID, userID)
		}
	case "/business":
		if isGroup {
			b.showBusiness(ctx, chatID, userID)
		}
	case "/company":
		if isGroup {
			b.showCompany(ctx, chatID, userID)
		}
	case "/stock":
		if isGroup {
			b.showStock(ctx, chatID, userID)
		}
	case "/trade":
		if isGroup {
			b.showTrade(ctx, chatID, userID)
		}
	case "/events":
		b.showEvents(ctx, chatID, false)
	case "/pay":
		if isGroup {
			b.handlePayStart(ctx, chatID, userID, args)
		}

	default:
		if isGroup {
			b.showGroupHelp(ctx, chatID)
		} else {
			b.showMainMenu(ctx, chatID, userID)
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// GROUP: /city sub-routing
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) handleCityGroup(ctx context.Context, chatID, userID int64, args []string) {
	if len(args) == 0 {
		b.showCityInfoGroup(ctx, chatID, userID)
		return
	}
	switch args[0] {
	case "register":
		name := strings.Join(args[1:], " ")
		if name == "" {
			b.sendMessage(chatID, "Формат: /city register Название города")
			return
		}
		b.registerCity(ctx, chatID, userID, name)
	case "leave":
		b.leaveCity(ctx, chatID, userID)
	default:
		b.showCityInfoGroup(ctx, chatID, userID)
	}
}

func (b *Bot) registerCity(ctx context.Context, chatID, userID int64, name string) {
	err := b.services.City.RegisterCity(ctx, chatID, name, userID)
	if err != nil {
		b.sendMessage(chatID, "❌ "+err.Error())
		return
	}
	text := fmt.Sprintf("🏙 Город «%s» зарегистрирован!\n\n"+
		"Вы — мэр. Используйте:\n"+
		"/city — информация о городе\n"+
		"/work — начать работу",
		name)
	b.sendMessage(chatID, text)
}

func (b *Bot) leaveCity(ctx context.Context, chatID, userID int64) {
	err := b.services.City.LeaveCity(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "❌ "+err.Error())
		return
	}
	b.sendMessage(chatID, "👋 Вы покинули город.")
}

// ────────────────────────────────────────────────────────────────────────────
// MAIN MENU (DM)
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showMainMenu(ctx context.Context, chatID, userID int64) {
	user, err := b.services.User.GetOrCreateUser(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "Ошибка: "+err.Error())
		return
	}

	cityText := "не выбран"
	if user.CityID != nil {
		city, err := b.services.City.GetCityByID(ctx, *user.CityID)
		if err == nil {
			cityText = city.Name
		}
	}

	activeWork := "нет"
	if user.ActiveJob != nil {
		w, err := b.services.Work.GetWorkDefinition(ctx, *user.ActiveJob)
		if err == nil {
			activeWork = w.Name
		}
	}

	text := fmt.Sprintf(
		"🎮 ЯчМан\n\n"+
			"👤 Уровень %d | XP %d\n"+
			"💰 %s ₽\n"+
			"🏙 Город: %s\n"+
			"🔨 Работа: %s\n",
		user.GlobalLevel, user.GlobalXP,
		formatMoney(user.Balance),
		cityText,
		activeWork,
	)

	buttons := [][]InlineButton{
		{
			{Text: "👤 Профиль", Data: "menu:profile"},
			{Text: "💰 Бонус", Data: "menu:daily"},
		},
		{
			{Text: "🏙 Города", Data: "menu:cities"},
			{Text: "📚 Обучение", Data: "menu:study"},
		},
		{
			{Text: "🔔 Уведомления", Data: "menu:notif"},
			{Text: "⭐ VIP", Data: "menu:vip"},
		},
	}

	// If in a city, show city quick actions
	if user.CityID != nil {
		buttons = append(buttons, []InlineButton{
			{Text: "📊 Город", Data: "menu:city_quick"},
			{Text: "❓ Помощь", Data: "menu:help"},
		})
	} else {
		buttons = append(buttons, []InlineButton{
			{Text: "❓ Помощь", Data: "menu:help"},
		})
	}

	b.sendMessageWithButtons(chatID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// PROFILE
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showProfile(ctx context.Context, chatID, userID int64, edit bool) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "Профиль не найден. /start")
		return
	}

	cityText := "не выбран"
	if user.CityID != nil {
		city, err := b.services.City.GetCityByID(ctx, *user.CityID)
		if err == nil {
			cityText = city.Name
		}
	}

	corpText := "нет"
	if user.CorporationID != nil {
		corp, err := b.services.Corp.GetCorporation(ctx, *user.CorporationID)
		if err == nil {
			corpText = fmt.Sprintf("%s (%s)", corp.Name, roleToEmoji(*user.CorporationRole))
		}
	}

	activeWork := "нет"
	if user.ActiveJob != nil {
		w, err := b.services.Work.GetWorkDefinition(ctx, *user.ActiveJob)
		if err == nil {
			activeWork = w.Name
		}
	}

	vipText := "❌ не активен"
	if user.VipUntil != nil && user.VipUntil.After(time.Now()) {
		vipText = fmt.Sprintf("✅ до %s", user.VipUntil.Format("02.01.2006"))
	}

	// Top skills
	skills, _ := b.services.User.GetSkills(ctx, user.ID)
	skillsText := ""
	shown := 0
	for _, s := range skills {
		if s.XP == 0 || shown >= 5 {
			continue
		}
		skillsText += fmt.Sprintf("  • %s: %d XP\n", s.Direction, s.XP)
		shown++
	}
	if skillsText == "" {
		skillsText = "  пока нет навыков\n"
	}

	text := fmt.Sprintf(
		"👤 Профиль\n\n"+
			"💰 Баланс: %s ₽\n"+
			"📊 Уровень: %d | XP: %d\n"+
			"🏙 Город: %s\n"+
			"🏢 Корпорация: %s\n"+
			"🔨 Работа: %s\n"+
			"🔥 Серия: %d дней\n"+
			"⭐ VIP: %s\n\n"+
			"📈 Топ навыки:\n%s",
		formatMoney(user.Balance),
		user.GlobalLevel, user.GlobalXP,
		cityText,
		corpText,
		activeWork,
		user.DailyStreak,
		vipText,
		skillsText,
	)

	buttons := [][]InlineButton{
		{
			{Text: "📈 Навыки", Data: "menu:skills"},
			{Text: "🎓 Обучение", Data: "menu:edu"},
		},
		{{Text: "◀ Назад", Data: "menu:main"}},
	}

	if edit {
		msgID := b.extractMessageID(ctx, chatID)
		if msgID > 0 {
			b.editMessageWithButtons(chatID, msgID, text, buttons)
			return
		}
	}
	b.sendMessageWithButtons(chatID, text, buttons)
}

func (b *Bot) showSkills(ctx context.Context, chatID, userID int64) {
	skills, err := b.services.User.GetSkills(ctx, userID)
	if err != nil || len(skills) == 0 {
		b.sendMessage(chatID, "📈 Нет данных о навыках")
		return
	}

	text := "📈 Все навыки:\n\n"
	for _, s := range skills {
		if s.XP == 0 {
			continue
		}
		bar := progressBar(s.XP, 3500)
		text += fmt.Sprintf("%s %s\n  %s %d XP\n\n", directionEmoji(s.Direction), s.Direction, bar, s.XP)
	}

	buttons := [][]InlineButton{
		{{Text: "◀ Назад", Data: "menu:profile"}},
	}
	b.sendMessageWithButtons(chatID, text, buttons)
}

func (b *Bot) showUserEducation(ctx context.Context, chatID, userID int64) {
	educations, err := b.services.Education.GetUserEducation(ctx, userID)
	if err != nil || len(educations) == 0 {
		b.sendMessage(chatID, "🎓 Нет записей об обучении")
		return
	}

	text := "🎓 Моё обучение:\n\n"
	for _, e := range educations {
		prog, _ := b.services.Education.GetProgram(ctx, e.ProgramID)
		status := "📖"
		if e.Completed {
			status = "✅"
		}
		name := e.ProgramID
		if prog != nil {
			name = prog.Name
		}
		if e.Completed {
			text += fmt.Sprintf("%s %s — завершено\n\n", status, name)
		} else {
			lessonText := ""
			if e.NextLessonAt != nil && e.NextLessonAt.After(time.Now()) {
				lessonText = fmt.Sprintf("  ⏰ След. урок: %s", e.NextLessonAt.Format("02.01 15:04"))
			}
			text += fmt.Sprintf("%s %s\n  Прогресс: %d уроков%s\n\n", status, name, e.Progress, lessonText)
		}
	}

	buttons := [][]InlineButton{}
	for _, e := range educations {
		if e.Completed {
			continue
		}
		buttons = append(buttons, []InlineButton{{
			Text: fmt.Sprintf("📖 Урок: %s", e.ProgramID),
			Data: fmt.Sprintf("study_lesson:%s", e.ProgramID),
		}})
	}
	buttons = append(buttons, []InlineButton{{Text: "◀ Назад", Data: "menu:profile"}})
	b.sendMessageWithButtons(chatID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// DAILY BONUS
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showDaily(ctx context.Context, chatID, userID int64, edit bool) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "❌ Профиль не найден")
		return
	}

	streak := user.DailyStreak
	bonus := 250 + (streak)*50
	if bonus > 600 {
		bonus = 600
	}

	// Check if already claimed today
	claimed := false
	if user.LastDailyAt != nil {
		hoursSince := time.Since(*user.LastDailyAt).Hours()
		if hoursSince < 20 {
			claimed = true
		}
	}

	streakBar := ""
	for i := 0; i < 7; i++ {
		if i < streak {
			streakBar += "🟢"
		} else {
			streakBar += "⚫"
		}
	}

	text := fmt.Sprintf(
		"💰 Ежедневный бонус\n\n"+
			"Серия: %d/7 %s\n\n"+
			"Текущий бонус: %d ₽\n"+
			"Максимум: 600 ₽ (серия 7 дней)\n\n"+
			"Каждый день серии: +50 ₽",
		streak, streakBar,
		bonus,
	)

	var buttons [][]InlineButton
	if claimed {
		text += "\n\n✅ Бонус уже получен сегодня!"
		buttons = [][]InlineButton{
			{{Text: "◀ Назад", Data: "menu:main"}},
		}
	} else {
		buttons = [][]InlineButton{
			{{Text: fmt.Sprintf("💰 Забрать %d ₽", bonus), Data: "daily_claim"}},
			{{Text: "◀ Назад", Data: "menu:main"}},
		}
	}

	if edit {
		msgID := b.extractMessageID(ctx, chatID)
		if msgID > 0 {
			b.editMessageWithButtons(chatID, msgID, text, buttons)
			return
		}
	}
	b.sendMessageWithButtons(chatID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// CITIES
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showCitiesList(ctx context.Context, chatID, userID int64) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "❌ Профиль не найден")
		return
	}

	text := "🏙 Города\n\n"

	// Current city
	if user.CityID != nil {
		city, err := b.services.City.GetCityByID(ctx, *user.CityID)
		if err == nil {
			players, _ := b.services.City.GetPlayerCount(ctx, city.ID)
			text += fmt.Sprintf("📍 Текущий город: %s\n"+
				"   Уровень: %s | NPC: %d | Игроков: %d | DP: %d\n\n",
				city.Name, city.Level, city.NPCPopulation, players, city.DevelopmentPoints)
			text += "─ ─ ─ ─ ─ ─ ─ ─\n\n"
		}
	} else {
		text += "📍 Вы не в городе. Вступите в один из городов ниже!\n\n"
	}

	// Public cities
	cities, err := b.services.City.ListPublicCities(ctx)
	if err != nil || len(cities) == 0 {
		text += "Публичных городов пока нет.\n"
	} else {
		for _, c := range cities {
			players, _ := b.services.City.GetPlayerCount(ctx, c.ID)
			marker := ""
			if user.CityID != nil && *user.CityID == c.ID {
				marker = " 📍"
			}
			text += fmt.Sprintf("🏙 %s%s\n   %s | 👥 %d | NPC %d | DP %d\n\n",
				c.Name, marker, c.Level, players, c.NPCPopulation, c.DevelopmentPoints)
		}
	}

	// Buttons
	var buttons [][]InlineButton
	if user.CityID != nil {
		buttons = append(buttons, []InlineButton{
			{Text: "📊 Мой город", Data: "menu:city_quick"},
		})
		buttons = append(buttons, []InlineButton{
			{Text: "🚪 Покинуть город", Data: "city_leave"},
		})
	} else if len(cities) > 0 {
		for _, c := range cities {
			buttons = append(buttons, []InlineButton{
				{Text: fmt.Sprintf("➕ Вступить в %s", c.Name),
					Data: fmt.Sprintf("city_join:%d", c.ID)},
			})
		}
	}
	buttons = append(buttons, []InlineButton{{Text: "◀ Назад", Data: "menu:main"}})
	b.sendMessageWithButtons(chatID, text, buttons)
}

func (b *Bot) showCityQuick(ctx context.Context, chatID, userID int64) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil || user.CityID == nil {
		b.sendMessage(chatID, "❌ Вы не в городе")
		return
	}
	b.showCityInfo(ctx, chatID, *user.CityID, "menu:cities")
}

func (b *Bot) showCityInfo(ctx context.Context, chatID, cityID int64, backData string) {
	city, err := b.services.City.GetCityByID(ctx, cityID)
	if err != nil {
		b.sendMessage(chatID, "❌ Город не найден")
		return
	}

	players, _ := b.services.City.GetPlayerCount(ctx, city.ID)

	text := fmt.Sprintf(
		"🏙 %s\n\n"+
			"📊 Уровень: %s\n"+
			"👥 Игроков: %d | NPC: %d\n"+
			"💰 Казна: %s ₽\n"+
			"📈 DP: %d\n"+
			"🔒 Доступ: %s\n\n"+
			"Налоги:\n"+
			"  Предприятия: %.0f%%\n"+
			"  Корпорации: %.0f%%\n"+
			"  Подоходный: %.0f%%",
		city.Name,
		city.Level,
		players, city.NPCPopulation,
		formatMoney(city.Treasury),
		city.DevelopmentPoints,
		city.AccessMode,
		city.TaxRateBusiness,
		city.TaxRateCorporate,
		city.TaxRateIncome,
	)

	buttons := [][]InlineButton{
		{
			{Text: "📈 Ресурсы", Data: "market_view"},
			{Text: "🏭 Предприятия", Data: "biz_list"},
		},
		{
			{Text: "📋 Контракты", Data: "trade_list"},
			{Text: "🎯 События", Data: "menu:events"},
		},
		{{Text: "◀ Назад", Data: backData}},
	}
	b.sendMessageWithButtons(chatID, text, buttons)
}

func (b *Bot) showCityDetailFromID(ctx context.Context, chatID, userID int64, idStr string) {
	cityID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		b.sendMessage(chatID, "❌ Неверный ID города")
		return
	}
	b.showCityInfo(ctx, chatID, cityID, "menu:cities")
}

// ────────────────────────────────────────────────────────────────────────────
// WORK (2-level navigation)
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showWorkDirections(ctx context.Context, chatID, userID int64) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "❌ Профиль не найден")
		return
	}
	if user.CityID == nil {
		b.sendMessage(chatID, "❌ Сначала вступите в город: /cities")
		return
	}

	text := "🔨 Выберите направление работы:\n\n"

	// Get user skills to show XP per direction
	skills, _ := b.services.User.GetSkills(ctx, user.ID)
	skillMap := make(map[string]int)
	for _, s := range skills {
		skillMap[s.Direction] = s.XP
	}

	var buttons [][]InlineButton
	for _, dir := range enums.AllSkillDirections {
		xp := skillMap[string(dir)]
		text += fmt.Sprintf("%s %s — %d XP\n", directionEmoji(string(dir)), dir, xp)
		buttons = append(buttons, []InlineButton{{
			Text: fmt.Sprintf("%s %s (%d XP)", directionEmoji(string(dir)), dir, xp),
			Data: fmt.Sprintf("work_dir:%s", dir),
		}})
	}

	text += "\nНажмите на направление для просмотра работ."
	buttons = append(buttons, []InlineButton{{Text: "◀ Назад", Data: "menu:main"}})

	b.sendMessageWithButtons(chatID, text, buttons)
}

func (b *Bot) showWorkByDirection(ctx context.Context, chatID, userID int64, direction string) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "❌ Профиль не найден")
		return
	}

	// Get XP in this direction
	var dirXP int
	skills, _ := b.services.User.GetSkills(ctx, user.ID)
	for _, s := range skills {
		if s.Direction == direction {
			dirXP = s.XP
			break
		}
	}

	works, err := b.services.Work.ListWorksByDirection(ctx, direction)
	if err != nil || len(works) == 0 {
		b.sendMessage(chatID, "❌ Работы не найдены")
		return
	}

	text := fmt.Sprintf("🔨 %s\nВаш XP: %d\n\n", direction, dirXP)

	var buttons [][]InlineButton
	for _, w := range works {
		status := "✅"
		if dirXP < w.RequiredXP {
			status = "🔒"
		}
		text += fmt.Sprintf("%s %s\n  ⏱ %d мин | 💰 %d ₽ | +%d XP | +%d ед.\n\n",
			status, w.Name, w.DurationMinutes, w.Payout, w.XPReward, w.ResourceAmount)

		if dirXP >= w.RequiredXP {
			buttons = append(buttons, []InlineButton{{
				Text: fmt.Sprintf("▶ %s (%d мин)", w.Name, w.DurationMinutes),
				Data: fmt.Sprintf("work_go:%s", w.ID),
			}})
		}
	}

	// Active work indicator
	if user.ActiveJob != nil {
		run, workName, err2 := b.services.Work.GetActiveWork(ctx, user.ID)
		if err2 == nil {
			remaining := time.Until(run.FinishesAt)
			text = fmt.Sprintf("🔨 Выполняется: %s\n⏱ Осталось: %s\n\n"+
				"Доступные работы:\n\n", workName, formatDuration(remaining)) + text
		}
	}

	buttons = append(buttons, []InlineButton{
		{Text: "◀ К направлениям", Data: "menu:work"},
	})
	b.sendMessageWithButtons(chatID, text, buttons)
}

func (b *Bot) startWork(ctx context.Context, chatID, userID int64, workID string, msgID int) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil || user.CityID == nil {
		b.sendMessage(chatID, "❌ Сначала вступите в город")
		return
	}

	work, err := b.services.Work.GetWorkDefinition(ctx, workID)
	if err != nil {
		b.sendMessage(chatID, "❌ Работа не найдена")
		return
	}

	err = b.services.Work.StartWork(ctx, userID, workID, *user.CityID)
	if err != nil {
		b.answerCallbackByMsg(chatID, msgID, "❌ "+err.Error())
		return
	}

	text := fmt.Sprintf("✅ Работа начата!\n\n"+
		"🔨 %s\n"+
		"⏱ Время: %d мин\n"+
		"💰 Награда: %d ₽\n"+
		"🎯 XP: +%d\n"+
		"📦 Ресурс: +%d ед. %s",
		work.Name, work.DurationMinutes, work.Payout,
		work.XPReward, work.ResourceAmount, resourceName(work.ResourceType))

	buttons := [][]InlineButton{
		{{Text: "◀ К работам", Data: fmt.Sprintf("work_dir:%s", work.Direction)}},
	}
	b.editMessageWithButtons(chatID, msgID, text, buttons)
}

func (b *Bot) showActiveWork(ctx context.Context, chatID, userID int64) {
	run, workName, err := b.services.Work.GetActiveWork(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "📭 Нет активной работы.\n\nИспользуйте /jobs для просмотра доступных работ.")
		return
	}

	remaining := time.Until(run.FinishesAt)
	text := fmt.Sprintf("🔨 Активная работа\n\n"+
		"%s\n"+
		"⏱ Осталось: %s\n"+
		"⏰ Завершится: %s",
		workName,
		formatDuration(remaining),
		run.FinishesAt.Format("15:04"))

	b.sendMessage(chatID, text)
}

// ────────────────────────────────────────────────────────────────────────────
// STUDY
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showStudyMenu(ctx context.Context, chatID, userID int64) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "❌ Профиль не найден")
		return
	}

	// Check active education
	educations, _ := b.services.Education.GetUserEducation(ctx, userID)
	hasActive := false
	for _, e := range educations {
		if !e.Completed {
			hasActive = true
			break
		}
	}

	text := "📚 Обучение\n\n"
	var buttons [][]InlineButton

	if hasActive {
		text += "📖 Активные курсы:\n"
		for _, e := range educations {
			if e.Completed {
				continue
			}
			prog, _ := b.services.Education.GetProgram(ctx, e.ProgramID)
			name := e.ProgramID
			lessons := 0
			if prog != nil {
				name = prog.Name
				lessons = prog.LessonCount
			}
			canStudy := true
			cooldownText := ""
			if e.NextLessonAt != nil && e.NextLessonAt.After(time.Now()) {
				canStudy = false
				cooldownText = fmt.Sprintf(" (через %s)", formatDuration(time.Until(*e.NextLessonAt)))
			}
			text += fmt.Sprintf("  📖 %s — %d/%d уроков%s\n", name, e.Progress, lessons, cooldownText)
			if canStudy {
				buttons = append(buttons, []InlineButton{{
					Text: fmt.Sprintf("📖 Урок: %s", name),
					Data: fmt.Sprintf("study_lesson:%s", e.ProgramID),
				}})
			}
		}
		text += "\n"
	}

	// Available programs
	programs, _ := b.services.Education.ListPrograms(ctx)
	text += "📋 Доступные программы:\n"
	for _, p := range programs {
		status := "✅"
		for _, e := range educations {
			if e.ProgramID == p.ID && e.Completed {
				status = "✅"
				break
			}
		}
		locked := user.GlobalXP < p.RequiredXP
		if locked {
			status = "🔒"
		}
		text += fmt.Sprintf("\n%s %s\n  %s ₽ | XP: %d | Уроков: %d\n",
			status, p.Name, formatMoney(p.Cost), p.RequiredXP, p.LessonCount)
		if !locked {
			buttons = append(buttons, []InlineButton{{
				Text: fmt.Sprintf("📝 %s", p.Name),
				Data: fmt.Sprintf("study_enroll:%s", p.ID),
			}})
		}
	}

	buttons = append(buttons, []InlineButton{{Text: "◀ Назад", Data: "menu:main"}})
	b.sendMessageWithButtons(chatID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// NOTIFICATIONS
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showNotifications(ctx context.Context, chatID, userID int64) {
	notifs, err := b.services.Notif.GetUnread(ctx, userID)
	if err != nil || len(notifs) == 0 {
		b.sendMessageWithButtons(chatID, "📭 Нет новых уведомлений",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:main"}}})
		return
	}

	text := "📬 Уведомления:\n\n"
	for _, n := range notifs {
		text += fmt.Sprintf("• %s: %s\n", n["title"], n["body"])
	}

	b.services.Notif.MarkRead(ctx, userID)

	buttons := [][]InlineButton{
		{{Text: "✅ Прочитать все", Data: "notif_read"}},
		{{Text: "◀ Назад", Data: "menu:main"}},
	}
	b.sendMessageWithButtons(chatID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// MARKET
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showMarket(ctx context.Context, chatID, userID int64) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil || user.CityID == nil {
		b.sendMessage(chatID, "❌ Вы не в городе")
		return
	}

	stock, err := b.services.Market.GetCityResources(ctx, *user.CityID)
	if err != nil {
		b.sendMessage(chatID, "❌ Ошибка загрузки рынка")
		return
	}

	text := "📈 Рынок ресурсов\n\n"
	for resID, qty := range stock {
		// Calculate price
		price, _ := b.services.Market.CalculatePrice(ctx, *user.CityID, resID)
		text += fmt.Sprintf("• %s: %d ед. @ %d ₽\n", resourceName(resID), qty, price)
	}

	if len(stock) == 0 {
		text += "Ресурсов пока нет.\n"
	}

	buttons := [][]InlineButton{
		{{Text: "◀ Назад", Data: "menu:city_quick"}},
	}
	b.sendMessageWithButtons(chatID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// BUSINESS
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showBusiness(ctx context.Context, chatID, userID int64) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil || user.CityID == nil {
		b.sendMessage(chatID, "❌ Вы не в городе")
		return
	}

	stock, err := b.services.Market.GetCityResources(ctx, *user.CityID)
	if err != nil {
		b.sendMessage(chatID, "❌ Ошибка")
		return
	}

	text := "🏭 Ресурсы города\n\n"
	for resID, qty := range stock {
		text += fmt.Sprintf("📦 %s: %d ед.\n", resourceName(resID), qty)
	}
	if len(stock) == 0 {
		text += "Ресурсов пока нет.\n"
	}

	buttons := [][]InlineButton{
		{{Text: "◀ Назад", Data: "menu:city_quick"}},
	}
	b.sendMessageWithButtons(chatID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// COMPANY
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showCompany(ctx context.Context, chatID, userID int64) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil || user.CorporationID == nil {
		b.sendMessage(chatID, "🏢 Вы не в корпорации")
		return
	}

	corp, err := b.services.Corp.GetCorporation(ctx, *user.CorporationID)
	if err != nil {
		b.sendMessage(chatID, "❌ Корпорация не найдена")
		return
	}

	staff, _ := b.services.Corp.GetStaff(ctx, corp.ID)
	price, _ := b.services.Stock.GetSharePrice(ctx, corp.ID)
	myShares, _ := b.services.Stock.GetUserShares(ctx, user.ID, corp.ID)

	text := fmt.Sprintf(
		"🏢 %s\n\n"+
			"💰 Баланс: %s ₽\n"+
			"👥 Сотрудников: %d\n"+
			"📈 Курс акции: %s ₽\n"+
			"📊 Всего акций: %s\n"+
			"🔑 Мои акции: %s\n"+
			"👤 Владелец: #%d",
		corp.Name,
		formatMoney(corp.Balance),
		len(staff),
		formatMoney(price),
		formatMoney(corp.TotalShares),
		formatMoney(myShares),
		corp.OwnerUserID,
	)

	buttons := [][]InlineButton{
		{
			{Text: "📈 Акции", Data: "stock_view"},
			{Text: "👥 Штат", Data: "corp_staff"},
		},
		{{Text: "◀ Назад", Data: "menu:main"}},
	}
	b.sendMessageWithButtons(chatID, text, buttons)
}

func (b *Bot) showCorpStaff(ctx context.Context, chatID, userID int64) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil || user.CorporationID == nil {
		b.sendMessage(chatID, "❌ Не в корпорации")
		return
	}

	staff, _ := b.services.Corp.GetStaff(ctx, *user.CorporationID)
	text := "👥 Штат корпорации:\n\n"
	for _, s := range staff {
		text += fmt.Sprintf("• #%d — %s\n", s.UserID, roleToEmoji(s.Role))
	}
	if len(staff) == 0 {
		text += "Нет сотрудников\n"
	}

	buttons := [][]InlineButton{
		{{Text: "◀ Назад", Data: "menu:company"}},
	}
	b.sendMessageWithButtons(chatID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// STOCK
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showStock(ctx context.Context, chatID, userID int64) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil || user.CorporationID == nil {
		b.sendMessage(chatID, "🏢 Вы не в корпорации")
		return
	}

	corp, err := b.services.Corp.GetCorporation(ctx, *user.CorporationID)
	if err != nil {
		b.sendMessage(chatID, "❌ Корпорация не найдена")
		return
	}

	price, _ := b.services.Stock.GetSharePrice(ctx, corp.ID)
	myShares, _ := b.services.Stock.GetUserShares(ctx, user.ID, corp.ID)

	text := fmt.Sprintf(
		"📈 Акции %s\n\n"+
			"💰 Курс: %s ₽/акция\n"+
			"📊 Всего: %s акций\n"+
			"🔑 Мои: %s акций\n"+
			"📈 Доля: %.1f%%",
		corp.Name,
		formatMoney(price),
		formatMoney(corp.TotalShares),
		formatMoney(myShares),
		float64(myShares)/math.Max(1, float64(corp.TotalShares))*100,
	)

	var buttons [][]InlineButton
	if corp.TotalShares > 0 {
		buttons = append(buttons, []InlineButton{
			{Text: "📥 Купить", Data: "stock_buy"},
			{Text: "📤 Продать", Data: "stock_sell"},
		})
	}
	buttons = append(buttons, []InlineButton{
		{Text: "◀ Назад", Data: "menu:company"},
	})
	b.sendMessageWithButtons(chatID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// TRADE
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showTrade(ctx context.Context, chatID, userID int64) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil || user.CityID == nil {
		b.sendMessage(chatID, "❌ Вы не в городе")
		return
	}

	contracts, err := b.services.Trade.GetCityContracts(ctx, *user.CityID)
	if err != nil || len(contracts) == 0 {
		b.sendMessageWithButtons(chatID, "📋 Нет активных контрактов",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:city_quick"}}})
		return
	}

	text := "📋 Торговые контракты:\n\n"
	for _, c := range contracts {
		text += fmt.Sprintf("• %s → %s\n  %s | %d ед./день | %d ₽/ед.\n\n",
			c["from"], c["to"], c["resource"], c["qty_per_day"], c["price"])
	}

	buttons := [][]InlineButton{
		{{Text: "◀ Назад", Data: "menu:city_quick"}},
	}
	b.sendMessageWithButtons(chatID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// EVENTS
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showEvents(ctx context.Context, chatID int64, edit bool) {
	events, err := b.services.Events.GetActiveEvents(ctx)
	if err != nil || len(events) == 0 {
		text := "📭 Нет активных событий"
		buttons := [][]InlineButton{{{Text: "◀ Назад", Data: "menu:main"}}}
		if edit {
			msgID := b.extractMessageID(ctx, chatID)
			if msgID > 0 {
				b.editMessageWithButtons(chatID, msgID, text, buttons)
				return
			}
		}
		b.sendMessageWithButtons(chatID, text, buttons)
		return
	}

	text := "🎯 Активные события:\n\n"
	for _, e := range events {
		emoji := eventEmoji(e.Type)
		remaining := time.Until(e.EndAt)
		text += fmt.Sprintf("%s [%s] %s\n   %s\n   ⏰ Осталось: %s\n\n",
			emoji, e.Type, e.Name, e.Description, formatDuration(remaining))
	}

	buttons := [][]InlineButton{
		{{Text: "◀ Назад", Data: "menu:main"}},
	}
	if edit {
		msgID := b.extractMessageID(ctx, chatID)
		if msgID > 0 {
			b.editMessageWithButtons(chatID, msgID, text, buttons)
			return
		}
	}
	b.sendMessageWithButtons(chatID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// VIP
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showVip(ctx context.Context, chatID, userID int64, edit bool) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "❌ Профиль не найден")
		return
	}

	vipText := "❌ VIP не активен"
	if user.VipUntil != nil && user.VipUntil.After(time.Now()) {
		vipText = fmt.Sprintf("✅ VIP до %s", user.VipUntil.Format("02.01.2006"))
	}

	text := fmt.Sprintf(
		"⭐ VIP\n\n"+
			"Статус: %s\n\n"+
			"Тарифы:\n"+
			"• 30 дней — 100 Stars\n"+
			"• 90 дней — 250 Stars\n"+
			"• 365 дней — 800 Stars\n\n"+
			"Преимущества:\n"+
			"• Косметика и титулы\n"+
			"• Расширенная аналитика\n"+
			"• Оформление профиля",
		vipText)

	buttons := [][]InlineButton{
		{{Text: "◀ Назад", Data: "menu:main"}},
	}
	if edit {
		msgID := b.extractMessageID(ctx, chatID)
		if msgID > 0 {
			b.editMessageWithButtons(chatID, msgID, text, buttons)
			return
		}
	}
	b.sendMessageWithButtons(chatID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// HELP
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showHelp(ctx context.Context, chatID int64, edit bool) {
	text := "❓ Справка ЯчМан\n\n" +
		"📱 Личные команды:\n" +
		"/profile — профиль\n" +
		"/daily — ежедневный бонус\n" +
		"/cities — список городов\n" +
		"/study — обучение\n" +
		"/notifications — уведомления\n" +
		"/vip — информация о VIP\n\n" +
		"🏙 Команды города (в группе):\n" +
		"/work — начать работу\n" +
		"/city — информация о городе\n" +
		"/market — рынок ресурсов\n" +
		"/business — предприятия\n" +
		"/company — корпорация\n" +
		"/stock — акции\n" +
		"/trade — контракты\n" +
		"/events — события\n" +
		"/pay — перевод\n\n" +
		"💡 Совет: используйте кнопки для навигации!"

	buttons := [][]InlineButton{
		{{Text: "◀ Назад", Data: "menu:main"}},
	}
	if edit {
		msgID := b.extractMessageID(ctx, chatID)
		if msgID > 0 {
			b.editMessageWithButtons(chatID, msgID, text, buttons)
			return
		}
	}
	b.sendMessageWithButtons(chatID, text, buttons)
}

func (b *Bot) showGroupHelp(ctx context.Context, chatID int64) {
	b.sendMessage(chatID,
		"🏗 Команды города:\n\n"+
			"/work — начать работу\n"+
			"/city — информация\n"+
			"/city register Название — создать город\n"+
			"/city leave — покинуть город\n"+
			"/market — рынок\n"+
			"/business — предприятия\n"+
			"/company — корпорация\n"+
			"/stock — акции\n"+
			"/trade — контракты\n"+
			"/events — события\n"+
			"/pay — перевод")
}

// ────────────────────────────────────────────────────────────────────────────
// PAY
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) handlePayStart(ctx context.Context, chatID, userID int64, args []string) {
	if len(args) < 2 {
		b.sendMessage(chatID,
			"💸 Перевод\n\n"+
				"Использование:\n"+
				"/pay @user сумма\n"+
				"/pay 123456 500")
		return
	}

	target := args[0]
	amountStr := args[1]

	amount, err := strconv.Atoi(amountStr)
	if err != nil || amount < 1 {
		b.sendMessage(chatID, "❌ Некорректная сумма")
		return
	}
	if amount > 10000000 {
		b.sendMessage(chatID, "❌ Максимум: 10 000 000 ₽")
		return
	}

	// Resolve target
	var targetTGID int64
	if strings.HasPrefix(target, "@") {
		b.sendMessage(chatID,
			"💡 Введите числовой ID получателя\n"+
				"Или ответьте на сообщение получателя: /pay сумма")
		return
	}
	id, err := strconv.ParseInt(target, 10, 64)
	if err != nil {
		b.sendMessage(chatID, "❌ Неверный ID получателя")
		return
	}
	targetTGID = id

	// Large amounts need confirmation
	if amount > 100000 {
		text := fmt.Sprintf(
			"💸 Подтвердите перевод\n\n"+
				"Получатель: %d\n"+
				"Сумма: %s ₽\n\n"+
				"⚠️ Это действие необратимо!",
			targetTGID, formatMoney(amount))
		buttons := [][]InlineButton{
			{
				{Text: "✅ Подтвердить", Data: fmt.Sprintf("pay_yes:%d:%d", targetTGID, amount)},
				{Text: "❌ Отмена", Data: "pay_no"},
			},
		}
		b.sendMessageWithButtons(chatID, text, buttons)
		return
	}

	// Execute directly
	if err := b.services.Payment.Transfer(ctx, userID, targetTGID, amount); err != nil {
		b.sendMessage(chatID, "❌ "+err.Error())
		return
	}
	b.sendMessage(chatID, fmt.Sprintf("✅ Перевод выполнен: %s ₽ → %d", formatMoney(amount), targetTGID))
}

// ────────────────────────────────────────────────────────────────────────────
// CALLBACK HANDLER
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) handleCallback(ctx context.Context, cb *CallbackQuery) {
	data := cb.Data
	userID := cb.From.ID
	chatID := int64(0)
	msgID := 0
	if cb.Message != nil {
		chatID = cb.Message.Chat.ID
		msgID = cb.Message.MessageID
	}

	// ── Menu navigation ──────────────────────────────
	switch {
	case data == "menu:main":
		b.showMainMenu(ctx, chatID, userID)

	case data == "menu:profile":
		b.showProfile(ctx, chatID, userID, true)

	case data == "menu:skills":
		b.showSkills(ctx, chatID, userID)

	case data == "menu:edu":
		b.showUserEducation(ctx, chatID, userID)

	case data == "menu:daily":
		b.showDaily(ctx, chatID, userID, true)

	case data == "menu:cities":
		b.showCitiesList(ctx, chatID, userID)

	case data == "menu:city_quick":
		b.showCityQuick(ctx, chatID, userID)

	case data == "menu:study":
		b.showStudyMenu(ctx, chatID, userID)

	case data == "menu:notif":
		b.showNotifications(ctx, chatID, userID)

	case data == "menu:vip":
		b.showVip(ctx, chatID, userID, true)

	case data == "menu:help":
		b.showHelp(ctx, chatID, true)

	case data == "menu:events":
		b.showEvents(ctx, chatID, true)

	case data == "menu:work":
		b.showWorkDirections(ctx, chatID, userID)

	case data == "menu:company":
		b.showCompany(ctx, chatID, userID)

	// ── Work ─────────────────────────────────────────
	case strings.HasPrefix(data, "work_dir:"):
		dir := strings.TrimPrefix(data, "work_dir:")
		b.showWorkByDirection(ctx, chatID, userID, dir)

	case strings.HasPrefix(data, "work_go:"):
		workID := strings.TrimPrefix(data, "work_go:")
		b.startWork(ctx, chatID, userID, workID, msgID)

	// ── Daily ────────────────────────────────────────
	case data == "daily_claim":
		bonus, err := b.services.User.ClaimDailyBonus(ctx, userID)
		if err != nil {
			b.answerCallback(cb.ID, "❌ "+err.Error())
			return
		}
		b.answerCallback(cb.ID, fmt.Sprintf("💰 +%d ₽", bonus))
		b.showDaily(ctx, chatID, userID, true)

	// ── City ─────────────────────────────────────────
	case strings.HasPrefix(data, "city_join:"):
		cityIDStr := strings.TrimPrefix(data, "city_join:")
		cityID, _ := strconv.ParseInt(cityIDStr, 10, 64)
		err := b.services.City.JoinCity(ctx, userID, cityID)
		if err != nil {
			b.answerCallback(cb.ID, "❌ "+err.Error())
			return
		}
		b.answerCallback(cb.ID, "✅ Добро пожаловать!")
		b.showCitiesList(ctx, chatID, userID)

	case data == "city_leave":
		err := b.services.City.LeaveCity(ctx, userID)
		if err != nil {
			b.answerCallback(cb.ID, "❌ "+err.Error())
			return
		}
		b.answerCallback(cb.ID, "👋 Город покинут")
		b.showCitiesList(ctx, chatID, userID)

	// ── Market ───────────────────────────────────────
	case data == "market_view":
		b.showMarket(ctx, chatID, userID)

	// ── Business ─────────────────────────────────────
	case data == "biz_list":
		b.showBusiness(ctx, chatID, userID)

	// ── Company ──────────────────────────────────────
	case data == "corp_staff":
		b.showCorpStaff(ctx, chatID, userID)

	// ── Stock ────────────────────────────────────────
	case data == "stock_view":
		b.showStock(ctx, chatID, userID)

	case data == "stock_buy" || data == "stock_sell":
		b.answerCallback(cb.ID, "💡 Используйте /stock buy количество")

	// ── Trade ────────────────────────────────────────
	case data == "trade_list":
		b.showTrade(ctx, chatID, userID)

	// ── Notifications ────────────────────────────────
	case data == "notif_read":
		b.services.Notif.MarkRead(ctx, userID)
		b.answerCallback(cb.ID, "✅ Все прочитано")
		b.showNotifications(ctx, chatID, userID)

	// ── Pay ──────────────────────────────────────────
	case strings.HasPrefix(data, "pay_yes:"):
		parts := strings.Split(data, ":")
		if len(parts) == 3 {
			targetID, _ := strconv.ParseInt(parts[1], 10, 64)
			amount, _ := strconv.Atoi(parts[2])
			err := b.services.Payment.Transfer(ctx, cb.From.ID, targetID, amount)
			if err != nil {
				b.answerCallback(cb.ID, "❌ "+err.Error())
				return
			}
			b.answerCallback(cb.ID, "✅ Перевод выполнен!")
			if msgID > 0 {
				b.editMessage(chatID, msgID, fmt.Sprintf("✅ Перевод %s ₽ выполнен → %d", formatMoney(amount), targetID))
			}
		}

	case data == "pay_no":
		b.answerCallback(cb.ID, "❌ Отменено")
		if msgID > 0 {
			b.editMessage(chatID, msgID, "❌ Перевод отменён")
		}

	// ── Study ────────────────────────────────────────
	case strings.HasPrefix(data, "study_enroll:"):
		progID := strings.TrimPrefix(data, "study_enroll:")
		err := b.services.Education.Enroll(ctx, userID, progID)
		if err != nil {
			b.answerCallback(cb.ID, "❌ "+err.Error())
			return
		}
		b.answerCallback(cb.ID, "✅ Записан!")
		b.showStudyMenu(ctx, chatID, userID)

	case strings.HasPrefix(data, "study_lesson:"):
		progID := strings.TrimPrefix(data, "study_lesson:")
		progress, err := b.services.Education.Study(ctx, userID, progID)
		if err != nil {
			b.answerCallback(cb.ID, "❌ "+err.Error())
			return
		}
		b.answerCallback(cb.ID, fmt.Sprintf("📖 Урок пройден! %d", progress))
		b.showStudyMenu(ctx, chatID, userID)

	default:
		b.answerCallback(cb.ID, "")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Telegram API helpers
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) sendMessage(chatID int64, text string) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.token)
	data := url.Values{
		"chat_id": {strconv.FormatInt(chatID, 10)},
		"text":    {text},
	}
	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		log.Printf("sendMessage error: %v", err)
		return
	}
	resp.Body.Close()
}

func (b *Bot) sendMessageWithButtons(chatID int64, text string, buttons [][]InlineButton) {
	markup := buildInlineKeyboard(buttons)
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.token)
	data := url.Values{
		"chat_id":      {strconv.FormatInt(chatID, 10)},
		"text":         {text},
		"reply_markup": {markup},
	}
	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		log.Printf("sendMessageWithButtons error: %v", err)
		return
	}
	resp.Body.Close()
}

func (b *Bot) editMessageWithButtons(chatID int64, messageID int, text string, buttons [][]InlineButton) {
	markup := buildInlineKeyboard(buttons)
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", b.token)
	data := url.Values{
		"chat_id":      {strconv.FormatInt(chatID, 10)},
		"message_id":   {strconv.Itoa(messageID)},
		"text":         {text},
		"reply_markup": {markup},
	}
	http.PostForm(apiURL, data)
}

func (b *Bot) editMessage(chatID int64, messageID int, text string) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", b.token)
	data := url.Values{
		"chat_id":    {strconv.FormatInt(chatID, 10)},
		"message_id": {strconv.Itoa(messageID)},
		"text":       {text},
	}
	http.PostForm(apiURL, data)
}

func (b *Bot) answerCallback(callbackID, text string) {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", b.token)
	data := url.Values{
		"callback_query_id": {callbackID},
		"text":              {text},
	}
	http.PostForm(apiURL, data)
}

func (b *Bot) answerCallbackByMsg(chatID int64, msgID int, text string) {
	// Send as a new message since we can't answer callback without ID
	b.sendMessage(chatID, text)
}

func (b *Bot) extractMessageID(ctx context.Context, chatID int64) int {
	// We can't reliably extract the last message ID from chat history via Telegram API
	// For now, return 0 to force send as new message
	return 0
}

func (b *Bot) isUpdateProcessed(ctx context.Context, updateID int) bool {
	if b.db == nil {
		return false
	}
	var exists bool
	_ = b.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM processed_updates WHERE update_id = $1)`, updateID).Scan(&exists)
	return exists
}

func (b *Bot) markUpdateProcessed(ctx context.Context, updateID int) {
	if b.db == nil {
		return
	}
	_, _ = b.db.Exec(ctx,
		`INSERT INTO processed_updates (update_id) VALUES ($1) ON CONFLICT DO NOTHING`, updateID)
}

// ────────────────────────────────────────────────────────────────────────────
// Inline keyboard builder
// ────────────────────────────────────────────────────────────────────────────

func buildInlineKeyboard(rows [][]InlineButton) string {
	var keyboard [][]map[string]string
	for _, row := range rows {
		var btns []map[string]string
		for _, btn := range row {
			btns = append(btns, map[string]string{
				"text":          btn.Text,
				"callback_data": btn.Data,
			})
		}
		keyboard = append(keyboard, btns)
	}
	b, _ := json.Marshal(map[string]interface{}{"inline_keyboard": keyboard})
	return string(b)
}

// ────────────────────────────────────────────────────────────────────────────
// Formatting helpers
// ────────────────────────────────────────────────────────────────────────────

func formatMoney(n int) string {
	if n < 1000 {
		return strconv.Itoa(n)
	}
	s := strconv.Itoa(n)
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += " "
		}
		result += string(c)
	}
	return result
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		return "завершается..."
	}
	d = d.Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%d ч %d мин", h, m)
	}
	return fmt.Sprintf("%d мин", m)
}

func progressBar(current, max int) string {
	if max <= 0 {
		max = 1
	}
	pct := float64(current) / float64(max)
	if pct > 1 {
		pct = 1
	}
	filled := int(pct * 10)
	bar := ""
	for i := 0; i < 10; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	return bar
}

func directionEmoji(dir string) string {
	emojis := map[string]string{
		"добыча":              "⛏",
		"лес":                 "🌲",
		"топливо":             "🛢",
		"энергетика":          "⚡",
		"металлургия":         "🔩",
		"строительство":       "🏗",
		"химия":               "🧪",
		"IT":                  "💻",
		"торговля":            "🛒",
		"агро":                "🌾",
		"транспорт":           "🚛",
		"питание":             "🍳",
		"ремонт":              "🔧",
		"медицина":            "🏥",
		"образование":         "🎓",
		"наука":               "🔬",
		"безопасность":        "🛡",
		"медиа":               "📺",
		"коммунальные услуги": "🏠",
		"переработка":         "♻",
	}
	if e, ok := emojis[dir]; ok {
		return e
	}
	return "•"
}

func eventEmoji(eventType string) string {
	switch eventType {
	case "world":
		return "🌍"
	case "city":
		return "🏙"
	case "economic":
		return "📊"
	case "social":
		return "🎉"
	default:
		return "🎯"
	}
}

func resourceName(resID string) string {
	names := map[string]string{
		"R1": "Продовольствие",
		"R2": "Руда",
		"R3": "Древесина",
		"R4": "Топливо",
		"R5": "Энергия",
		"R6": "Металл",
		"R7": "Материалы",
		"R8": "Химикаты",
		"R9": "Технологии",
		"R10": "Потребтовары",
	}
	if n, ok := names[resID]; ok {
		return n
	}
	return resID
}

func roleToEmoji(role string) string {
	switch role {
	case "owner":
		return "👑 Владелец"
	case "executive":
		return "👔 Руководитель"
	case "manager":
		return "📋 Менеджер"
	case "employee":
		return "👷 Сотрудник"
	default:
		return role
	}
}
