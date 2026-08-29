package webapp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"yachman/internal/services"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	services Services
	botToken string
}

type Services struct {
	User    *services.UserService
	City    *services.CityService
	Market  *services.MarketService
	Events  *services.EventService
	Work    *services.WorkService
	Payment *services.PaymentService
}

func NewHandler(svc Services, botToken string) *Handler {
	return &Handler{services: svc, botToken: botToken}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api", func(r chi.Router) {
		// Public routes (no auth)
		r.Get("/cities", h.listCities)
		r.Get("/cities/{id}", h.getCity)
		r.Get("/events", h.listEvents)
		r.Get("/resources/{cityId}", h.cityResources)
		r.Get("/works", h.listWorks)
		r.Get("/works/{direction}", h.worksByDirection)

		// Protected routes (Telegram init data required)
		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(h.botToken))
			r.Get("/me", h.getMe)
			r.Get("/me/skills", h.getMySkills)
			r.Post("/work/start", h.startWork)
			r.Post("/pay", h.processPay)
		})
	})
}

func (h *Handler) listCities(w http.ResponseWriter, r *http.Request) {
	cities, err := h.services.City.ListPublicCities(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type cityJSON struct {
		ID      int64  `json:"id"`
		Name    string `json:"name"`
		Level   string `json:"level"`
		NPCPop  int    `json:"npc_population"`
		Players int    `json:"real_players"`
		Public  bool   `json:"public"`
	}
	var result []cityJSON
	for _, c := range cities {
		playerCount, _ := h.services.City.GetPlayerCount(r.Context(), c.ID)
		result = append(result, cityJSON{
			ID: c.ID, Name: c.Name, Level: c.Level,
			NPCPop: c.NPCPopulation, Players: playerCount, Public: c.PublicListing,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"cities": result})
}

func (h *Handler) getCity(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	cityID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	city, err := h.services.City.GetCityByID(r.Context(), cityID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(city)
}

func (h *Handler) listEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.services.Events.GetRecentWorldEvents(r.Context(), 100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"events": events})
}

func (h *Handler) cityResources(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "cityId")
	cityID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	stock, err := h.services.Market.GetCityResources(r.Context(), cityID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"resources": stock})
}

func (h *Handler) listWorks(w http.ResponseWriter, r *http.Request) {
	directions := []string{
		"добыча", "лес", "топливо", "энергетика", "металлургия",
		"строительство", "химия", "IT", "торговля", "агро",
		"транспорт", "питание", "ремонт", "медицина", "образование",
		"наука", "безопасность", "медиа", "коммунальные услуги", "переработка",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"directions": directions})
}

func (h *Handler) worksByDirection(w http.ResponseWriter, r *http.Request) {
	dir := chi.URLParam(r, "direction")
	works, err := h.services.Work.ListWorksByDirection(r.Context(), dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"works": works})
}

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	uid, ok := UserIDFromContext(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	user, err := h.services.User.GetUserByTGID(r.Context(), uid)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"user": user})
}

func (h *Handler) getMySkills(w http.ResponseWriter, r *http.Request) {
	uid, ok := UserIDFromContext(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	user, err := h.services.User.GetUserByTGID(r.Context(), uid)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}
	skills, err := h.services.User.GetSkills(r.Context(), user.ID)
	if err != nil {
		http.Error(w, `{"error":"failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"skills": skills})
}
func (h *Handler) startWork(w http.ResponseWriter, r *http.Request) {
	uid, ok := UserIDFromContext(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	var req struct {
		WorkID string `json:"work_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
		return
	}
	user, err := h.services.User.GetUserByTGID(r.Context(), uid)
	if err != nil || user.CityID == nil {
		http.Error(w, `{"error":"join a city first"}`, http.StatusBadRequest)
		return
	}
	err = h.services.Work.StartWork(r.Context(), uid, req.WorkID, *user.CityID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true})
}

func (h *Handler) processPay(w http.ResponseWriter, r *http.Request) {
	uid, ok := UserIDFromContext(r)
	if !ok {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	_ = uid // future: use for /pay via Web App
	http.Error(w, `{"error":"use Telegram bot for /pay"}`, http.StatusNotImplemented)
}
