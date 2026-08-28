package webapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ValidateTelegramInitData validates the initData string from Telegram Web App.
// Returns the authenticated user_id or an error.
// See: https://core.telegram.org/bots/webapps#validating-data-received-via-the-mini-app
func ValidateTelegramInitData(initData, botToken string) (int64, error) {
	if initData == "" || botToken == "" {
		return 0, fmt.Errorf("missing initData or botToken")
	}

	parsed, err := url.ParseQuery(initData)
	if err != nil {
		return 0, fmt.Errorf("invalid initData format: %w", err)
	}

	hash := parsed.Get("hash")
	if hash == "" {
		return 0, fmt.Errorf("missing hash in initData")
	}

	// Remove hash from data check
	dataCheckMap := make(url.Values)
	for k, v := range parsed {
		if k != "hash" {
			dataCheckMap[k] = v
		}
	}

	// Sort keys alphabetically
	keys := make([]string, 0, len(dataCheckMap))
	for k := range dataCheckMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build data-check-string
	var checkParts []string
	for _, k := range keys {
		checkParts = append(checkParts, k+"="+dataCheckMap.Get(k))
	}
	dataCheckString := strings.Join(checkParts, "\n")

	// HMAC-SHA256 with bot token as secret
	mac := hmac.New(sha256.New, []byte(botToken))
	mac.Write([]byte(dataCheckString))
	expectedHash := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(hash), []byte(expectedHash)) {
		return 0, fmt.Errorf("invalid hash: data integrity check failed")
	}

	// Check auth_date is not too old (24h max)
	authDateStr := parsed.Get("auth_date")
	if authDateStr == "" {
		return 0, fmt.Errorf("missing auth_date")
	}
	authDate, err := strconv.ParseInt(authDateStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid auth_date: %w", err)
	}
	if time.Since(time.Unix(authDate, 0)) > 24*time.Hour {
		return 0, fmt.Errorf("initData expired: auth_date is older than 24h")
	}

	// Extract user ID
	userStr := parsed.Get("user")
	if userStr == "" {
		return 0, fmt.Errorf("missing user in initData")
	}
	// Parse the nested user JSON to get id
	var userData struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(userStr), &userData); err != nil {
		return 0, fmt.Errorf("invalid user data: %w", err)
	}
	if userData.ID == 0 {
		return 0, fmt.Errorf("invalid user id in initData")
	}

	return userData.ID, nil
}

// AuthMiddleware validates Telegram Web App initData from the X-Telegram-Init-Data header.
// On success, it sets the validated user_id in the request context.
func AuthMiddleware(botToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			initData := r.Header.Get("X-Telegram-Init-Data")
			if initData == "" {
				// Also check query param for dev/testing
				initData = r.URL.Query().Get("initData")
			}
			if initData == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			userID, err := ValidateTelegramInitData(initData, botToken)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusUnauthorized)
				return
			}
			// Store user_id in context
			ctx := r.Context()
			ctx = context.WithValue(ctx, "user_id", userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext extracts the validated user_id from request context.
func UserIDFromContext(r *http.Request) (int64, bool) {
	v := r.Context().Value("user_id")
	if v == nil {
		return 0, false
	}
	uid, ok := v.(int64)
	return uid, ok
}
