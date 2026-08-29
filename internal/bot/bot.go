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

type TGUpdate struct {
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
	ID      string     `json:"id"`
	From    *TGUser    `json:"from"`
	Data    string     `json:"data"`
	Message *TGMessage `json:"message"`
}

// ────────────────────────────────────────────────────────────────────────────
// Entry points
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) HandleUpdate(ctx context.Context, update TGUpdate) {
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
			Ok     bool       `json:"ok"`
			Result []TGUpdate `json:"result"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			log.Printf("unmarshal error: %v", err)
			time.Sleep(2 * time.Second)
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
			b.showMainMenu(ctx, chatID, userID, 0)
		}
	case "/help":
		if isGroup {
			b.showGroupHelp(ctx, chatID)
		} else {
			b.showMainMenu(ctx, chatID, userID, 0)
		}

	// ── DM commands ──────────────────────────────────
	case "/profile", "/balance":
		b.showProfile(ctx, chatID, userID, 0)
	case "/daily":
		b.showDaily(ctx, chatID, userID, 0)
	case "/cities":
		if isGroup {
			b.showGroupHelp(ctx, chatID)
		} else {
			b.showCitiesList(ctx, chatID, userID, 0)
		}
	case "/study":
		if isGroup {
			b.showGroupHelp(ctx, chatID)
		} else {
			b.showStudyMenu(ctx, chatID, userID, 0)
		}
	case "/notifications":
		if isGroup {
			b.showGroupHelp(ctx, chatID)
		} else {
			b.showNotifications(ctx, chatID, userID, 0)
		}
	case "/vip":
		if isGroup {
			b.showGroupHelp(ctx, chatID)
		} else {
			b.showVip(ctx, chatID, userID, 0)
		}

	// ── Group commands ───────────────────────────────
	case "/work":
		b.showWorkDirections(ctx, chatID, userID, 0)
	case "/jobs":
		b.showJobs(ctx, chatID, userID, 0)
	case "/city":
		if isGroup {
			b.handleCityGroup(ctx, chatID, userID, args)
		} else {
			if len(args) > 0 {
				b.showCityDetailFromID(ctx, chatID, userID, args[0], 0)
			} else {
				b.showCitiesList(ctx, chatID, userID, 0)
			}
		}
	case "/market":
		if isGroup {
			b.showMarket(ctx, chatID, userID, 0)
		}
	case "/business":
		if isGroup {
			b.showBusiness(ctx, chatID, userID, 0)
		}
	case "/company":
		if isGroup {
			b.showCompany(ctx, chatID, userID, 0)
		}
	case "/stock":
		if isGroup {
			b.showStock(ctx, chatID, userID, 0)
		}
	case "/trade":
		if isGroup {
			b.showTrade(ctx, chatID, userID, 0)
		}
	case "/events":
		b.showEvents(ctx, chatID, 0)
	case "/pay":
		if isGroup {
			b.handlePayStart(ctx, chatID, userID, args)
		}

	default:
		if isGroup {
			b.showGroupHelp(ctx, chatID)
		} else {
			b.showMainMenu(ctx, chatID, userID, 0)
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
	b.sendMessage(chatID, fmt.Sprintf(
		"🏙 Город «%s» зарегистрирован!\n\nВы — мэр. /city — информация", name))
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

func (b *Bot) showMainMenu(ctx context.Context, chatID, userID int64, msgID int) {
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

	b.sendOrEdit(chatID, msgID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// PROFILE
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showProfile(ctx context.Context, chatID, userID int64, msgID int) {
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
			role := ""
			if user.CorporationRole != nil {
				role = roleToEmoji(*user.CorporationRole)
			}
			corpText = fmt.Sprintf("%s (%s)", corp.Name, role)
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
	b.sendOrEdit(chatID, msgID, text, buttons)
}

func (b *Bot) showSkills(ctx context.Context, chatID, userID int64, msgID int) {
	internalID, err := b.resolveTGID(ctx, userID)
	if err != nil {
		b.sendOrEdit(chatID, msgID, "❌ Профиль не найден",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:profile"}}})
		return
	}
	skills, err := b.services.User.GetSkills(ctx, internalID)
	if err != nil || len(skills) == 0 {
		b.sendOrEdit(chatID, msgID, "📈 Нет данных о навыках",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:profile"}}})
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
	b.sendOrEdit(chatID, msgID, text, buttons)
}
func (b *Bot) showUserEducation(ctx context.Context, chatID, userID int64, msgID int) {
	internalID, err := b.resolveTGID(ctx, userID)
	if err != nil {
		b.sendOrEdit(chatID, msgID, "❌ Профиль не найден",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:profile"}}})
		return
	}
	educations, err := b.services.Education.GetUserEducation(ctx, internalID)
	if err != nil || len(educations) == 0 {
		b.sendOrEdit(chatID, msgID, "🎓 Нет записей об обучении",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:profile"}}})
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
		})
	}
	buttons = append(buttons, []InlineButton{{Text: "◀ Назад", Data: "menu:profile"}})
	b.sendOrEdit(chatID, msgID, text, buttons)
}
	buttons = append(buttons, []InlineButton{{Text: "◀ Назад", Data: "menu:profile"}})
	b.sendOrEdit(chatID, msgID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// DAILY BONUS
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showDaily(ctx context.Context, chatID, userID int64, msgID int) {
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
	b.sendOrEdit(chatID, msgID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// CITIES
// ────────────────────────────────────────────────────────────────────────────


func (b *Bot) showCitiesList(ctx context.Context, chatID, userID int64, msgID int) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "❌ Профиль не найден")
		return
	}

	text := ""
	if user.CityID != nil {
		city, err := b.services.City.GetCityByID(ctx, *user.CityID)
		if err == nil {
			players, _ := b.services.City.GetPlayerCount(ctx, city.ID)
			text = fmt.Sprintf("📍 %s — %s | 👥 %d", city.Name, city.Level, players)
		}
	} else {
		text = "📍 Вы не в городе"
	}

	cities, _ := b.services.City.ListPublicCities(ctx)
	var buttons [][]InlineButton

	if user.CityID != nil {
		buttons = append(buttons, []InlineButton{
			{Text: "📊 Мой город", Data: "menu:city_quick"},
			{Text: "🚪 Покинуть", Data: "city_leave"},
		})
	} else {
		for i, c := range cities {
			if i >= 8 {
				break
			}
			buttons = append(buttons, []InlineButton{{
				Text: fmt.Sprintf("➕ %s", c.Name),
				Data: fmt.Sprintf("city_join:%d", c.ID),
			}})
		}
	}
	buttons = append(buttons, []InlineButton{{Text: "◀ Назад", Data: "menu:main"}})
	b.sendOrEdit(chatID, msgID, text, buttons)
}

func (b *Bot) showCityQuick(ctx context.Context, chatID, userID int64, msgID int) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil || user.CityID == nil {
		b.sendOrEdit(chatID, msgID, "❌ Вы не в городе",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:cities"}}})
		return
	}
	b.showCityInfo(ctx, chatID, *user.CityID, msgID, "menu:cities")
}
func (b *Bot) showCityInfo(ctx context.Context, chatID, cityID int64, msgID int, backData string) {
	city, err := b.services.City.GetCityByID(ctx, cityID)
	if err != nil {
		b.sendOrEdit(chatID, msgID, "❌ Город не найден",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:main"}}})
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
	b.sendOrEdit(chatID, msgID, text, buttons)
}

func (b *Bot) showCityDetailFromID(ctx context.Context, chatID, userID int64, idStr string, msgID int) {
	cityID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		b.sendMessage(chatID, "❌ Неверный ID города")
		return
	}
	b.showCityInfo(ctx, chatID, cityID, msgID, "menu:cities")
}

func (b *Bot) showCityInfoGroup(ctx context.Context, chatID, userID int64) {
	city, err := b.services.City.GetCityByChatID(ctx, chatID)
	if err != nil {
		b.sendMessage(chatID, "Город не зарегистрирован.\nАдминистратор: /city register Название")
		return
	}
	players, _ := b.services.City.GetPlayerCount(ctx, city.ID)
	isMayor := city.MayorUserID != nil && *city.MayorUserID == userID

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

	var buttons [][]InlineButton
	if isMayor {
		buttons = append(buttons, []InlineButton{
			{Text: "📈 Ресурсы", Data: "market_view"},
			{Text: "🏭 Предприятия", Data: "biz_list"},
		})
	}
	buttons = append(buttons, []InlineButton{
		{Text: "📋 Контракты", Data: "trade_list"},
		{Text: "🎯 События", Data: "menu:events"},
	})
	b.sendMessageWithButtons(chatID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// WORK (2-level navigation)
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showWorkDirections(ctx context.Context, chatID, userID int64, msgID int) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "❌ Профиль не найден")
		return
	}
	if user.CityID == nil {
		b.sendOrEdit(chatID, msgID, "❌ Сначала вступите в город: /cities",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:cities"}}})
		return
	}

	skills, _ := b.services.User.GetSkills(ctx, user.ID)
	skillMap := make(map[string]int)
	for _, s := range skills {
		skillMap[s.Direction] = s.XP
	}

	// Compact 2-column button grid: no text body, just buttons
	var rows [][]InlineButton
	for i := 0; i < len(enums.AllSkillDirections); i += 2 {
		var row []InlineButton
		for j := 0; j < 2 && i+j < len(enums.AllSkillDirections); j++ {
			dir := enums.AllSkillDirections[i+j]
			xp := skillMap[string(dir)]
			row = append(row, InlineButton{
				Text: fmt.Sprintf("%s %s %d", directionEmoji(string(dir)), dir, xp),
				Data: fmt.Sprintf("work_dir:%s", dir),
			})
		}
		rows = append(rows, row)
	}
	rows = append(rows, []InlineButton{{Text: "◀ Назад", Data: "menu:main"}})
	b.sendOrEdit(chatID, msgID, "🔨 Направления работ:", rows)
}
func (b *Bot) showWorkByDirection(ctx context.Context, chatID, userID int64, direction string, msgID int) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "❌ Профиль не найден")
		return
	}

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
		b.sendOrEdit(chatID, msgID, "❌ Работы не найдены",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:work"}}})
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
	b.sendOrEdit(chatID, msgID, text, buttons)
}

func (b *Bot) startWork(ctx context.Context, chatID, userID int64, workID string, msgID int) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil || user.CityID == nil {
		b.sendOrEdit(chatID, msgID, "❌ Сначала вступите в город",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:cities"}}})
		return
	}

	work, err := b.services.Work.GetWorkDefinition(ctx, workID)
	if err != nil {
		b.sendOrEdit(chatID, msgID, "❌ Работа не найдена",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:work"}}})
		return
	}

	err = b.services.Work.StartWork(ctx, userID, workID, *user.CityID)
	if err != nil {
		b.sendOrEditWithWorkBack(chatID, msgID, "❌ "+err.Error())
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
	b.sendOrEdit(chatID, msgID, text, buttons)
}

func (b *Bot) showActiveWork(ctx context.Context, chatID, userID int64) {
	internalID, err := b.resolveTGID(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "❌ Профиль не найден")
		return
	}
	run, workName, err := b.services.Work.GetActiveWork(ctx, internalID)
	if err != nil {
		b.sendMessage(chatID, "📭 Нет активной работы.\n\nИспользуйте /jobs для просмотра работ.")
		return
	}

	remaining := time.Until(run.FinishesAt)
	b.sendMessage(chatID, fmt.Sprintf(

// showJobs shows active work status + available work directions
func (b *Bot) showJobs(ctx context.Context, chatID, userID int64, msgID int) {
	internalID, err := b.resolveTGID(ctx, userID)
	if err != nil {
		b.sendOrEdit(chatID, msgID, "❌ Профиль не найден",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:main"}}})
		return
	}

	user, _ := b.services.User.GetUserByTGID(ctx, userID)

	text := "👷 Мои работы\n\n"
	buttons := [][]InlineButton{}

	// Active work
	if user.ActiveJob != nil {
		run, workName, err2 := b.services.Work.GetActiveWork(ctx, internalID)
		if err2 == nil {
			remaining := time.Until(run.FinishesAt)
			text += fmt.Sprintf("🔨 Выполняется: %s\n⏱ Осталось: %s\n⏰ Завершится: %s\n\n",
				workName, formatDuration(remaining), run.FinishesAt.Format("15:04"))
		}
	} else {
		text += "📭 Нет активной работы\n\n"
	}

	// Available directions with XP
	if user.CityID != nil {
		text += "📌 Доступные направления:\n"
		skills, _ := b.services.User.GetSkills(ctx, user.ID)
		skillMap := make(map[string]int)
		for _, s := range skills {
			skillMap[s.Direction] = s.XP
		}

		for i := 0; i < len(enums.AllSkillDirections); i += 2 {
			var row []InlineButton
			for j := 0; j < 2 && i+j < len(enums.AllSkillDirections); j++ {
				dir := enums.AllSkillDirections[i+j]
				xp := skillMap[string(dir)]
				row = append(row, InlineButton{
					Text: fmt.Sprintf("%s %s %d", directionEmoji(string(dir)), dir, xp),
					Data: fmt.Sprintf("work_dir:%s", dir),
				})
			}
			buttons = append(buttons, row)
		}
	} else {
		text += "🏙 Вступите в город, чтобы работать\n"
		buttons = append(buttons, []InlineButton{{Text: "🏙 Города", Data: "menu:cities"}})
	}

	buttons = append(buttons, []InlineButton{{Text: "◀ Назад", Data: "menu:main"}})
	b.sendOrEdit(chatID, msgID, text, buttons)
}
		"🔨 Активная работа\n\n%s\n⏱ Осталось: %s\n⏰ Завершится: %s",
		workName, formatDuration(remaining), run.FinishesAt.Format("15:04")))
}
// ────────────────────────────────────────────────────────────────────────────
// STUDY
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showStudyMenu(ctx context.Context, chatID, userID int64, msgID int) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "❌ Профиль не найден")
		return
	}

	educations, _ := b.services.Education.GetUserEducation(ctx, user.ID)
	var buttons [][]InlineButton

	// Active courses — compact buttons
	for _, e := range educations {
		if !e.Completed {
			prog, _ := b.services.Education.GetProgram(ctx, e.ProgramID)
			name := e.ProgramID
			lessons := 0
			if prog != nil {
				name = prog.Name
				lessons = prog.LessonCount
			}
			canStudy := true
			if e.NextLessonAt != nil && e.NextLessonAt.After(time.Now()) {
				canStudy = false
			}
			if canStudy {
				buttons = append(buttons, []InlineButton{{
					Text: fmt.Sprintf("📖 %s (%d/%d)", name, e.Progress, lessons),
					Data: fmt.Sprintf("study_lesson:%s", e.ProgramID),
				}})
			} else {
				buttons = append(buttons, []InlineButton{{
					Text: fmt.Sprintf("⏳ %s (%d/%d)", name, e.Progress, lessons),
					Data: "noop",
				}})
			}
		}
	}

	// Available programs — compact buttons
	programs, _ := b.services.Education.ListPrograms(ctx)
	for _, p := range programs {
		enrolled := false
		for _, e := range educations {
			if e.ProgramID == p.ID {
				enrolled = true
				break
			}
		}
		locked := user.GlobalXP < p.RequiredXP
		if locked || enrolled {
			continue
		}
		buttons = append(buttons, []InlineButton{{
			Text: fmt.Sprintf("📝 %s", p.Name),
			Data: fmt.Sprintf("study_enroll:%s", p.ID),
		}})
	}

	buttons = append(buttons, []InlineButton{{Text: "◀ Назад", Data: "menu:main"}})
	b.sendOrEdit(chatID, msgID, "📚 Обучение:", buttons)
}
func (b *Bot) showNotifications(ctx context.Context, chatID, userID int64, msgID int) {
	internalID, err := b.resolveTGID(ctx, userID)
	if err != nil {
		b.sendOrEdit(chatID, msgID, "❌ Профиль не найден",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:main"}}})
		return
	}
	notifs, err := b.services.Notif.GetUnread(ctx, internalID)
	if err != nil || len(notifs) == 0 {
		b.sendOrEdit(chatID, msgID, "📭 Нет новых уведомлений",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:main"}}})
		return
	}

	text := "📬 Уведомления:\n\n"
	for _, n := range notifs {
		text += fmt.Sprintf("• %s: %s\n", n["title"], n["body"])
	}

	buttons := [][]InlineButton{
		{{Text: "✅ Прочитать все", Data: "notif_read"}},
		{{Text: "◀ Назад", Data: "menu:main"}},
	}
	b.sendOrEdit(chatID, msgID, text, buttons)
}
// ────────────────────────────────────────────────────────────────────────────
// MARKET
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showMarket(ctx context.Context, chatID, userID int64, msgID int) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil || user.CityID == nil {
		b.sendOrEdit(chatID, msgID, "❌ Вы не в городе",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:main"}}})
		return
	}

	stock, err := b.services.Market.GetCityResources(ctx, *user.CityID)
	if err != nil {
		b.sendOrEdit(chatID, msgID, "❌ Ошибка загрузки рынка",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:city_quick"}}})
		return
	}

	text := "📈 Рынок ресурсов\n\n"
	for resID, qty := range stock {
		price, _ := b.services.Market.CalculatePrice(ctx, *user.CityID, resID)
		text += fmt.Sprintf("• %s: %d ед. @ %d ₽\n", resourceName(resID), qty, price)
	}
	if len(stock) == 0 {
		text += "Ресурсов пока нет.\n"
	}

	buttons := [][]InlineButton{
		{{Text: "◀ Назад", Data: "menu:city_quick"}},
	}
	b.sendOrEdit(chatID, msgID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// BUSINESS
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showBusiness(ctx context.Context, chatID, userID int64, msgID int) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, "❌ Профиль не найден")
		return
	}

	businesses, err := b.services.Business.ListUserBusinesses(ctx, user.ID)
	if err != nil || len(businesses) == 0 {
		b.sendOrEdit(chatID, msgID, "🏭 У вас нет предприятий\n\nПредприятия создаёт мэр города.",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:main"}}})
		return
	}

	text := "🏭 Мои предприятия:\n\n"
	for _, biz := range businesses {
		text += fmt.Sprintf("• %s (%s)\n  ⚡ %d%% | %s → %s\n  👷 NPC: %d\n  🏙 %s\n\n",
			biz["name"], biz["type"], biz["power"],
			biz["input_a"], biz["output"], biz["npc_staff"], biz["city"])
	}

	buttons := [][]InlineButton{
		{{Text: "◀ Назад", Data: "menu:main"}},
	}
	b.sendOrEdit(chatID, msgID, text, buttons)
}
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showCompany(ctx context.Context, chatID, userID int64, msgID int) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil || user.CorporationID == nil {
		b.sendOrEdit(chatID, msgID, "🏢 Вы не в корпорации",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:main"}}})
		return
	}

	corp, err := b.services.Corp.GetCorporation(ctx, *user.CorporationID)
	if err != nil {
		b.sendOrEdit(chatID, msgID, "❌ Корпорация не найдена",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:main"}}})
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
	b.sendOrEdit(chatID, msgID, text, buttons)
}

func (b *Bot) showCorpStaff(ctx context.Context, chatID, userID int64, msgID int) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil || user.CorporationID == nil {
		b.sendOrEdit(chatID, msgID, "❌ Не в корпорации",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:main"}}})
		return
	}

	staff, _ := b.services.Corp.GetStaff(ctx, *user.CorporationID)
	text := "👥 Штат корпорации:\n\n"
	for _, s := range staff {
		text += fmt.Sprintf("• #%d — %s\n", s.UserID, roleToEmoji(s.Position))
	}
	if len(staff) == 0 {
		text += "Нет сотрудников\n"
	}

	buttons := [][]InlineButton{
		{{Text: "◀ Назад", Data: "menu:company"}},
	}
	b.sendOrEdit(chatID, msgID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// STOCK
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showStock(ctx context.Context, chatID, userID int64, msgID int) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil || user.CorporationID == nil {
		b.sendOrEdit(chatID, msgID, "🏢 Вы не в корпорации",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:main"}}})
		return
	}

	corp, err := b.services.Corp.GetCorporation(ctx, *user.CorporationID)
	if err != nil {
		b.sendOrEdit(chatID, msgID, "❌ Корпорация не найдена",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:main"}}})
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
	b.sendOrEdit(chatID, msgID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// TRADE
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showTrade(ctx context.Context, chatID, userID int64, msgID int) {
	user, err := b.services.User.GetUserByTGID(ctx, userID)
	if err != nil || user.CityID == nil {
		b.sendOrEdit(chatID, msgID, "❌ Вы не в городе",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:main"}}})
		return
	}

	contracts, err := b.services.Trade.GetCityContracts(ctx, *user.CityID)
	if err != nil || len(contracts) == 0 {
		b.sendOrEdit(chatID, msgID, "📋 Нет активных контрактов",
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
	b.sendOrEdit(chatID, msgID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// EVENTS
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showEvents(ctx context.Context, chatID int64, msgID int) {
	events, err := b.services.Events.GetActiveEvents(ctx)
	if err != nil || len(events) == 0 {
		b.sendOrEdit(chatID, msgID, "📭 Нет активных событий",
			[][]InlineButton{{{Text: "◀ Назад", Data: "menu:main"}}})
		return
	}

	text := "🎯 Активные события:\n\n"
	for _, e := range events {
		remaining := time.Until(e.EndAt)
		text += fmt.Sprintf("%s [%s] %s\n   %s\n   ⏰ Осталось: %s\n\n",
			eventEmoji(e.Type), e.Type, e.Name, e.Description, formatDuration(remaining))
	}

	buttons := [][]InlineButton{
		{{Text: "◀ Назад", Data: "menu:main"}},
	}
	b.sendOrEdit(chatID, msgID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// VIP
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showVip(ctx context.Context, chatID, userID int64, msgID int) {
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
	b.sendOrEdit(chatID, msgID, text, buttons)
}

// ────────────────────────────────────────────────────────────────────────────
// HELP
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) showHelp(ctx context.Context, chatID int64, msgID int) {
	buttons := [][]InlineButton{
		{{Text: "👤 Профиль", Data: "menu:profile"}, {Text: "💰 Бонус", Data: "menu:daily"}},
		{{Text: "🏙 Города", Data: "menu:cities"}, {Text: "📚 Обучение", Data: "menu:study"}},
		{{Text: "🔔 Уведомления", Data: "menu:notif"}, {Text: "⭐ VIP", Data: "menu:vip"}},
		{{Text: "◀ Назад", Data: "menu:main"}},
	}
	b.sendOrEdit(chatID, msgID, "❓ Навигация:", buttons)
}
func (b *Bot) showGroupHelp(ctx context.Context, chatID int64) {
	text := "🏗 Команды города:\n\n" +
		"🔨 /work — выбрать направление\n" +
		"👷 /jobs — моя работа / доступные\n" +
		"🏙 /city — информация о городе\n" +
		"🏙 /city register Название — создать город\n" +
		"🏙 /city leave — покинуть город\n" +
		"📈 /market — рынок ресурсов\n" +
		"🏭 /business — предприятия\n" +
		"🏢 /company — корпорация\n" +
		"📊 /stock — акции\n" +
		"📋 /trade — контракты\n" +
		"🎯 /events — события\n" +
		"💸 /pay @user сумма — перевод\n\n" +
		"📱 Личные команды (в ЛС бота):\n" +
		"/start, /profile, /balance, /daily, /study, /help"
	buttons := [][]InlineButton{
		{
			{Text: "🔨 Работа", Data: "menu:work"},
			{Text: "🏙 Город", Data: "noop"},
		},
		{
			{Text: "📈 Рынок", Data: "noop"},
			{Text: "🎯 События", Data: "noop"},
		},
		{{Text: "💡 Откройте ЛС для полного меню", Data: "noop"}},
	}
	b.sendMessageWithButtons(chatID, text, buttons)
}
// ────────────────────────────────────────────────────────────────────────────
// PAY
// ────────────────────────────────────────────────────────────────────────────

func (b *Bot) handlePayStart(ctx context.Context, chatID, userID int64, args []string) {
	if len(args) < 2 {
		b.sendMessage(chatID,
			"💸 Перевод\n\nИспользование:\n/pay @user сумма\n/pay 123456 500")
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

	var targetTGID int64
	if strings.HasPrefix(target, "@") {
		b.sendMessage(chatID,
			"💡 Введите числовой ID получателя\nИли ответьте на сообщение: /pay сумма")
		return
	}
	id, err := strconv.ParseInt(target, 10, 64)
	if err != nil {
		b.sendMessage(chatID, "❌ Неверный ID получателя")
		return
	}
	targetTGID = id

	if amount > 100000 {
		text := fmt.Sprintf(
			"💸 Подтвердите перевод\n\nПолучатель: %d\nСумма: %s ₽\n\n⚠️ Это действие необратимо!",
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

	if err := b.services.Payment.Transfer(ctx, userID, targetTGID, amount); err != nil {
		b.sendMessage(chatID, "❌ "+err.Error())
		return
	}
	b.sendMessage(chatID, fmt.Sprintf("✅ Перевод: %s ₽ → %d", formatMoney(amount), targetTGID))
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

	switch {

	// ── Menu navigation ──────────────────────────────
	case data == "noop":
		b.answerCallback(cb.ID, "")
	case data == "menu:main":
		// Main menu always creates a new message (clean entry point)
		b.showMainMenu(ctx, chatID, userID, 0)

	case data == "menu:profile":
		b.showProfile(ctx, chatID, userID, msgID)
	case data == "menu:skills":
		b.showSkills(ctx, chatID, userID, msgID)
	case data == "menu:edu":
		b.showUserEducation(ctx, chatID, userID, msgID)
	case data == "menu:daily":
		b.showDaily(ctx, chatID, userID, msgID)
	case data == "menu:cities":
		b.showCitiesList(ctx, chatID, userID, msgID)
	case data == "menu:city_quick":
		b.showCityQuick(ctx, chatID, userID, msgID)
	case data == "menu:study":
		b.showStudyMenu(ctx, chatID, userID, msgID)
	case data == "menu:notif":
		b.showNotifications(ctx, chatID, userID, msgID)
	case data == "menu:vip":
		b.showVip(ctx, chatID, userID, msgID)
	case data == "menu:help":
		b.showHelp(ctx, chatID, msgID)
	case data == "menu:events":
		b.showEvents(ctx, chatID, msgID)
	case data == "menu:work":
		b.showWorkDirections(ctx, chatID, userID, msgID)
	case data == "menu:company":
		b.showCompany(ctx, chatID, userID, msgID)

	// ── Work ─────────────────────────────────────────
	case strings.HasPrefix(data, "work_dir:"):
		dir := strings.TrimPrefix(data, "work_dir:")
		b.showWorkByDirection(ctx, chatID, userID, dir, msgID)

	case strings.HasPrefix(data, "work_go:"):
		workID := strings.TrimPrefix(data, "work_go:")
		b.startWork(ctx, chatID, userID, workID, msgID)

	// ── Daily ────────────────────────────────────────
	case data == "daily_claim":
		internalID, resolveErr := b.resolveTGID(ctx, userID)
		if resolveErr != nil {
			b.answerCallback(cb.ID, "❌ Ошибка")
			return
		}
		bonus, err := b.services.User.ClaimDailyBonus(ctx, internalID)
		if err != nil {
			b.answerCallback(cb.ID, "❌ "+err.Error())
			return
		}
		b.answerCallback(cb.ID, fmt.Sprintf("💰 +%d ₽", bonus))
		b.showDaily(ctx, chatID, userID, msgID)

	// ── City ─────────────────────────────────────────
	case strings.HasPrefix(data, "city_join:"):
		cityIDStr := strings.TrimPrefix(data, "city_join:")
		cityID, _ := strconv.ParseInt(cityIDStr, 10, 64)
		if err := b.services.City.JoinCity(ctx, userID, cityID); err != nil {
			b.answerCallback(cb.ID, "❌ "+err.Error())
			return
		}
		b.answerCallback(cb.ID, "✅ Добро пожаловать!")
		b.showCitiesList(ctx, chatID, userID, msgID)

	case data == "city_leave":
		if err := b.services.City.LeaveCity(ctx, userID); err != nil {
			b.answerCallback(cb.ID, "❌ "+err.Error())
			return
		}
		b.answerCallback(cb.ID, "👋 Город покинут")
		b.showCitiesList(ctx, chatID, userID, msgID)

	// ── Sub-screens (no state change, just navigation)
	case data == "market_view":
		b.showMarket(ctx, chatID, userID, msgID)
	case data == "biz_list":
		b.showBusiness(ctx, chatID, userID, msgID)
	case data == "corp_staff":
		b.showCorpStaff(ctx, chatID, userID, msgID)
	case data == "stock_view":
		b.showStock(ctx, chatID, userID, msgID)
	case data == "trade_list":
		b.showTrade(ctx, chatID, userID, msgID)

	// ── Stock buy/sell placeholder ───────────────────
	case data == "stock_buy" || data == "stock_sell":
		b.answerCallback(cb.ID, "💡 Используйте /stock buy количество")

	// ── Notifications ────────────────────────────────
	case data == "notif_read":
		notifInternalID, resolveErr := b.resolveTGID(ctx, userID)
		if resolveErr == nil {
			b.services.Notif.MarkRead(ctx, notifInternalID)
		}
		b.answerCallback(cb.ID, "✅ Все прочитано")
		b.showNotifications(ctx, chatID, userID, msgID)

	// ── Pay ──────────────────────────────────────────
	case strings.HasPrefix(data, "pay_yes:"):
		parts := strings.Split(data, ":")
		if len(parts) == 3 {
			targetID, _ := strconv.ParseInt(parts[1], 10, 64)
			amount, _ := strconv.Atoi(parts[2])
			if err := b.services.Payment.Transfer(ctx, cb.From.ID, targetID, amount); err != nil {
				b.answerCallback(cb.ID, "❌ "+err.Error())
				return
			}
			b.answerCallback(cb.ID, "✅ Перевод выполнен!")
			if msgID > 0 {
				b.editMessage(chatID, msgID, fmt.Sprintf("✅ Перевод %s ₽ → %d", formatMoney(amount), targetID))
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
		if err := b.services.Education.Enroll(ctx, userID, progID); err != nil {
			b.answerCallback(cb.ID, "❌ "+err.Error())
			return
		}
		b.answerCallback(cb.ID, "✅ Записан!")
		b.showStudyMenu(ctx, chatID, userID, msgID)

	case strings.HasPrefix(data, "study_lesson:"):
		progID := strings.TrimPrefix(data, "study_lesson:")
		progress, err := b.services.Education.Study(ctx, userID, progID)
		if err != nil {
			b.answerCallback(cb.ID, "❌ "+err.Error())
			return
		}
		b.answerCallback(cb.ID, fmt.Sprintf("📖 Урок пройден! %d", progress))
		b.showStudyMenu(ctx, chatID, userID, msgID)

	default:
		b.answerCallback(cb.ID, "")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Telegram API helpers
// ────────────────────────────────────────────────────────────────────────────

// apiResponse is used to parse Telegram API responses.
type apiResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		MessageID int `json:"message_id"`
	} `json:"result"`
}

func (b *Bot) apiPost(method string, data url.Values) apiURLResult {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/%s", b.token, method)
	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		log.Printf("API %s error: %v", method, err)
		return apiURLResult{}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var r apiResponse
	json.Unmarshal(body, &r)
	return apiURLResult{OK: r.OK, MessageID: r.Result.MessageID}
}

type apiURLResult struct {
	OK        bool
	MessageID int
}

func (b *Bot) sendMessage(chatID int64, text string) {
	data := url.Values{
		"chat_id": {strconv.FormatInt(chatID, 10)},
		"text":    {text},
	}
	b.apiPost("sendMessage", data)
}

// sendMessageWithButtons sends a message and returns the Telegram message_id.
func (b *Bot) sendMessageWithButtons(chatID int64, text string, buttons [][]InlineButton) int {
	markup := buildInlineKeyboard(buttons)
	data := url.Values{
		"chat_id":      {strconv.FormatInt(chatID, 10)},
		"text":         {text},
		"reply_markup": {markup},
	}
	r := b.apiPost("sendMessage", data)
	return r.MessageID
}

// editMessageWithButtons edits an existing message. Returns true on success.
func (b *Bot) editMessageWithButtons(chatID int64, messageID int, text string, buttons [][]InlineButton) bool {
	markup := buildInlineKeyboard(buttons)
	data := url.Values{
		"chat_id":      {strconv.FormatInt(chatID, 10)},
		"message_id":   {strconv.Itoa(messageID)},
		"text":         {text},
		"reply_markup": {markup},
	}
	r := b.apiPost("editMessageText", data)
	return r.OK
}

// editMessage edits text without buttons.
func (b *Bot) editMessage(chatID int64, messageID int, text string) {
	data := url.Values{
		"chat_id":    {strconv.FormatInt(chatID, 10)},
		"message_id": {strconv.Itoa(messageID)},
		"text":       {text},
	}
	b.apiPost("editMessageText", data)
}

// sendOrEdit is the core navigation helper:
//   - if msgID > 0: try to edit the existing message
//   - if msgID == 0 or edit fails: send a new message
func (b *Bot) sendOrEdit(chatID int64, msgID int, text string, buttons [][]InlineButton) {
	if msgID > 0 {
		if b.editMessageWithButtons(chatID, msgID, text, buttons) {
			return
		}
		// Edit failed (message deleted, text unchanged, etc.) — send new
	}
	b.sendMessageWithButtons(chatID, text, buttons)
}


// resolveTGID converts a Telegram user ID to the internal user ID.
func (b *Bot) resolveTGID(ctx context.Context, telegramUserID int64) (int64, error) {
	return b.services.User.ResolveInternalID(ctx, telegramUserID)
}
func (b *Bot) answerCallback(callbackID, text string) {
	data := url.Values{
		"callback_query_id": {callbackID},
		"text":              {text},
	}
	b.apiPost("answerCallbackQuery", data)
}

// sendOrEditWithWorkBack is a convenience for work-related error messages.
func (b *Bot) sendOrEditWithWorkBack(chatID int64, msgID int, text string) {
	b.sendOrEdit(chatID, msgID, text,
		[][]InlineButton{{{Text: "◀ Назад", Data: "menu:work"}}})
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
	l := len(s)
	result := ""
	for i, c := range s {
		if i > 0 && (l-i)%3 == 0 {
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
		"R1":  "Продовольствие",
		"R2":  "Руда",
		"R3":  "Древесина",
		"R4":  "Топливо",
		"R5":  "Энергия",
		"R6":  "Металл",
		"R7":  "Материалы",
		"R8":  "Химикаты",
		"R9":  "Технологии",
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
