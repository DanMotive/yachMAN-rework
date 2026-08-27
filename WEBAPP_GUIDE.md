# 🌐 ЯчМан — Telegram Web App: полная инструкция

## Часть 1: Настройка сервера (Nginx + HTTPS)

### 1.1 Установка Nginx и Certbot

```bash
apt update && apt install -y nginx certbot python3-certbot-nginx
```

### 1.2 Проверь что домен работает

```bash
# DNS должен указывать на твой сервер
ping danmotive.duckdns.org
# Должен показать IP твоего сервера
```

### 1.3 Получи сертификат Let's Encrypt

```bash
# Сначала создай временный HTTP конфиг
cat > /etc/nginx/sites-available/yachman << 'EOF'
server {
    listen 80;
    server_name danmotive.duckdns.org;
    location / {
        proxy_pass http://127.0.0.1:9090;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
EOF

ln -sf /etc/nginx/sites-available/yachman /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default
nginx -t && systemctl reload nginx

# Получи сертификат
certbot --nginx -d danmotive.duckdns.org --non-interactive --agree-tos -m your@email.com
```

### 1.4 Замени конфиг на полный

```bash
# Скопируй полный конфиг из репо
cp /opt/yachMAN-rework/nginx/yachman.conf /etc/nginx/sites-available/yachman
nginx -t && systemctl reload nginx
```

### 1.5 Проверь

```bash
curl -I https://danmotive.duckdns.org/health
# Должен показать HTTP/2 200
```

### 1.6 Автопродление сертификата

```bash
# Certbot уже настраивает timer, проверь:
systemctl status certbot.timer
```

---

## Часть 2: Создание Web App в BotFather

### 2.1 Создай Web App

1. Открой **@BotFather** в Telegram
2. Отправь `/newapp`
3. Выбери своего бота: `yachman_bot`
4. Введи **название**: `ЯчМан`
5. Введи **краткое описание**: `Экономическая игра`
6. Введи **описание**: `Глобальная социальная экономическая игра в Telegram`
7. **URL Web App**: `https://danmotive.duckdns.org/webapp/`
8. Загрузи **פון** (640×360 или 1280×720) — можно любой пока что
9. Загрузи **маленькое изображение** (160×160) — иконка

### 2.2 Настрой Menu Button (опционально)

В @BotFather:
```
/setmenubutton
```
Выбери бота → URL: `https://danmotive.duckdns.org/webapp/` → Текст: `🎮 Играть`

---

## Часть 3: Frontend Web App

### 3.1 Структура файлов

```
webapp/
├── index.html          # Главная страница
├── css/
│   └── style.css       # Стили
├── js/
│   ├── app.js          # Основная логика
│   ├── api.js          # API клиент
│   └── tg-webapp.js    # Telegram Web App SDK (загружается из CDN)
└── assets/
    └── logo.png        # Логотип
```

### 3.2 index.html

```html
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, user-scalable=no">
    <title>ЯчМан</title>
    <link rel="stylesheet" href="css/style.css">
    <!-- Telegram Web App SDK -->
    <script src="https://telegram.org/js/telegram-web-app.js"></script>
</head>
<body>
    <div id="app">
        <header id="header">
            <h1 id="city-name">Загрузка...</h1>
        </header>

        <main id="content">
            <!-- Профиль -->
            <section id="tab-profile" class="tab active">
                <div class="card">
                    <h2>👤 Мой профиль</h2>
                    <div id="profile-info">Загрузка...</div>
                </div>
            </section>

            <!-- Карта -->
            <section id="tab-map" class="tab">
                <div class="card">
                    <h2>🗺 Города</h2>
                    <div id="cities-list">Загрузка...</div>
                </div>
            </section>

            <!-- Работа -->
            <section id="tab-work" class="tab">
                <div class="card">
                    <h2>🔨 Работа</h2>
                    <div id="work-list">Загрузка...</div>
                </div>
            </section>

            <!-- Рынок -->
            <section id="tab-market" class="tab">
                <div class="card">
                    <h2>📈 Рынок</h2>
                    <div id="market-list">Загрузка...</div>
                </div>
            </section>
        </main>

        <nav id="bottom-nav">
            <button class="nav-btn active" data-tab="profile">👤</button>
            <button class="nav-btn" data-tab="map">🗺</button>
            <button class="nav-btn" data-tab="work">🔨</button>
            <button class="nav-btn" data-tab="market">📈</button>
        </nav>
    </div>

    <script src="js/api.js"></script>
    <script src="js/app.js"></script>
</body>
</html>
```

### 3.3 css/style.css

```css
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: var(--tg-theme-bg-color, #ffffff);
    color: var(--tg-theme-text-color, #000000);
    min-height: 100vh;
    padding-bottom: 60px;
}

.card {
    background: var(--tg-theme-secondary-bg-color, #f0f0f0);
    border-radius: 12px;
    padding: 16px;
    margin: 8px 12px;
}

.card h2 {
    font-size: 18px;
    margin-bottom: 12px;
}

.stat-row {
    display: flex;
    justify-content: space-between;
    padding: 8px 0;
    border-bottom: 1px solid rgba(128,128,128,0.2);
}

.stat-label {
    color: var(--tg-theme-hint-color, #888);
}

.stat-value {
    font-weight: 600;
}

.work-item {
    background: var(--tg-theme-bg-color, #fff);
    border-radius: 8px;
    padding: 12px;
    margin-bottom: 8px;
    border: 1px solid rgba(128,128,128,0.2);
}

.work-item .name {
    font-weight: 600;
    font-size: 16px;
}

.work-item .details {
    font-size: 13px;
    color: var(--tg-theme-hint-color, #888);
    margin-top: 4px;
}

.btn {
    display: inline-block;
    padding: 10px 20px;
    border-radius: 8px;
    border: none;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    margin-top: 8px;
}

.btn-primary {
    background: var(--tg-theme-button-color, #3390ec);
    color: var(--tg-theme-button-text-color, #ffffff);
}

.btn:active {
    opacity: 0.7;
}

.city-item {
    background: var(--tg-theme-bg-color, #fff);
    border-radius: 8px;
    padding: 12px;
    margin-bottom: 8px;
    border: 1px solid rgba(128,128,128,0.2);
}

.city-item .name {
    font-weight: 600;
    font-size: 16px;
}

.city-item .info {
    font-size: 13px;
    color: var(--tg-theme-hint-color, #888);
    margin-top: 4px;
}

.resource-item {
    display: flex;
    justify-content: space-between;
    padding: 10px 0;
    border-bottom: 1px solid rgba(128,128,128,0.2);
}

.resource-name {
    font-weight: 500;
}

.resource-price {
    font-weight: 600;
    color: var(--tg-theme-button-color, #3390ec);
}

/* Bottom navigation */
#bottom-nav {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    display: flex;
    background: var(--tg-theme-secondary-bg-color, #f0f0f0);
    border-top: 1px solid rgba(128,128,128,0.2);
    padding: 8px 0;
    padding-bottom: env(safe-area-inset-bottom, 8px);
}

.nav-btn {
    flex: 1;
    background: none;
    border: none;
    font-size: 24px;
    padding: 8px;
    cursor: pointer;
    opacity: 0.5;
    transition: opacity 0.2s;
}

.nav-btn.active {
    opacity: 1;
}

/* Hide inactive tabs */
.tab {
    display: none;
}

.tab.active {
    display: block;
}

/* Loading spinner */
.loading {
    text-align: center;
    padding: 40px;
    color: var(--tg-theme-hint-color, #888);
}

/* Progress bar for work timer */
.progress-bar {
    width: 100%;
    height: 6px;
    background: rgba(128,128,128,0.2);
    border-radius: 3px;
    margin-top: 8px;
    overflow: hidden;
}

.progress-fill {
    height: 100%;
    background: var(--tg-theme-button-color, #3390ec);
    border-radius: 3px;
    transition: width 1s linear;
}

/* Theme variables from Telegram */
:root {
    --tg-theme-bg-color: #ffffff;
    --tg-theme-text-color: #000000;
    --tg-theme-hint-color: #888888;
    --tg-theme-link-color: #3390ec;
    --tg-theme-button-color: #3390ec;
    --tg-theme-button-text-color: #ffffff;
    --tg-theme-secondary-bg-color: #f0f0f0;
}

@media (prefers-color-scheme: dark) {
    :root {
        --tg-theme-bg-color: #1a1a2e;
        --tg-theme-text-color: #e0e0e0;
        --tg-theme-hint-color: #888888;
        --tg-theme-button-color: #3390ec;
        --tg-theme-button-text-color: #ffffff;
        --tg-theme-secondary-bg-color: #16213e;
    }
}
```

### 3.4 js/api.js

```javascript
const API_BASE = window.location.origin;

const api = {
    async get(path) {
        const resp = await fetch(`${API_BASE}${path}`);
        if (!resp.ok) throw new Error(`API error: ${resp.status}`);
        return resp.json();
    },

    async post(path, body) {
        const resp = await fetch(`${API_BASE}${path}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        });
        if (!resp.ok) throw new Error(`API error: ${resp.status}`);
        return resp.json();
    },

    // Profile
    getProfile: () => api.get('/api/me'),
    getSkills: () => api.get('/api/me/skills'),

    // Cities
    getCities: () => api.get('/api/cities'),
    getCity: (id) => api.get(`/api/cities/${id}`),

    // Works
    getWorks: (dir) => api.get(`/api/works/${dir || ''}`),

    // Market
    getMarket: (cityId) => api.get(`/api/resources/${cityId}`),

    // Events
    getEvents: () => api.get('/api/events'),

    // Work
    startWork: (workId) => api.post('/api/work/start', { work_id: workId }),

    // Study
    getEducation: () => api.get('/api/me/education'),
    enroll: (programId) => api.post('/api/study', { program_id: programId }),
    study: (programId) => api.post('/api/study/lesson', { program_id: programId }),
};
```

### 3.5 js/app.js

```javascript
// Initialize Telegram Web App
const tg = window.Telegram?.WebApp;

if (tg) {
    tg.ready();
    tg.expand();
    document.body.style.backgroundColor = tg.backgroundColor || '#ffffff';
}

// Tab switching
document.querySelectorAll('.nav-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        document.querySelectorAll('.nav-btn').forEach(b => b.classList.remove('active'));
        document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
        btn.classList.add('active');
        document.getElementById(`tab-${btn.dataset.tab}`).classList.add('active');
        loadTab(btn.dataset.tab);
    });
});

// Load data for each tab
async function loadTab(tab) {
    switch (tab) {
        case 'profile': await loadProfile(); break;
        case 'map': await loadCities(); break;
        case 'work': await loadWorks(); break;
        case 'market': await loadMarket(); break;
    }
}

// Profile
async function loadProfile() {
    try {
        const data = await api.getProfile();
        const user = data.user;
        document.getElementById('profile-info').innerHTML = `
            <div class="stat-row">
                <span class="stat-label">Баланс</span>
                <span class="stat-value">💰 ${user.balance} ₽</span>
            </div>
            <div class="stat-row">
                <span class="stat-label">Уровень</span>
                <span class="stat-value">📊 ${user.global_level}</span>
            </div>
            <div class="stat-row">
                <span class="stat-label">XP</span>
                <span class="stat-value">🎯 ${user.global_xp}</span>
            </div>
            <div class="stat-row">
                <span class="stat-label">Серия дней</span>
                <span class="stat-value">🔥 ${user.daily_streak}</span>
            </div>
            ${user.city_id ? `<div class="stat-row">
                <span class="stat-label">Город</span>
                <span class="stat-value">🏙 #${user.city_id}</span>
            </div>` : ''}
            ${user.corporation_id ? `<div class="stat-row">
                <span class="stat-label">Корпорация</span>
                <span class="stat-value">🏢 #${user.corporation_id}</span>
            </div>` : ''}
        `;
    } catch (e) {
        document.getElementById('profile-info').innerHTML = '<p class="loading">Ошибка загрузки</p>';
    }
}

// Cities
async function loadCities() {
    try {
        const data = await api.getCities();
        const cities = data.cities || [];
        document.getElementById('cities-list').innerHTML = cities.map(c => `
            <div class="city-item">
                <div class="name">${c.name}</div>
                <div class="info">
                    Уровень: ${c.level} | NPC: ${c.npc_population} | Игроков: ${c.real_players}
                </div>
            </div>
        `).join('') || '<p class="loading">Нет городов</p>';
    } catch (e) {
        document.getElementById('cities-list').innerHTML = '<p class="loading">Ошибка загрузки</p>';
    }
}

// Works
async function loadWorks() {
    try {
        const data = await api.getWorks('добыча');
        const works = data.works || [];
        document.getElementById('work-list').innerHTML = works.map(w => `
            <div class="work-item">
                <div class="name">${w.name}</div>
                <div class="details">
                    ⏱ ${w.duration_minutes} мин | 💰 ${w.payout} ₽ | +${w.xp_reward} XP | +${w.resource_amount} ед.
                </div>
                <button class="btn btn-primary" onclick="startWork('${w.id}')">Начать</button>
            </div>
        `).join('') || '<p class="loading">Нет работ</p>';
    } catch (e) {
        document.getElementById('work-list').innerHTML = '<p class="loading">Ошибка загрузки</p>';
    }
}

// Market
async function loadMarket() {
    try {
        const profile = await api.getProfile();
        if (!profile.user.city_id) {
            document.getElementById('market-list').innerHTML = '<p class="loading">Вступите в город</p>';
            return;
        }
        const data = await api.getMarket(profile.user.city_id);
        const resources = data.resources || {};
        document.getElementById('market-list').innerHTML = Object.entries(resources).map(([id, qty]) => `
            <div class="resource-item">
                <span class="resource-name">${id}</span>
                <span class="resource-price">${qty} ед.</span>
            </div>
        `).join('') || '<p class="loading">Рынок пуст</p>';
    } catch (e) {
        document.getElementById('market-list').innerHTML = '<p class="loading">Ошибка загрузки</p>';
    }
}

// Start work
async function startWork(workId) {
    try {
        await api.startWork(workId);
        if (tg) tg.HapticFeedback.notificationOccurred('success');
        alert('Работа начата!');
    } catch (e) {
        if (tg) tg.HapticFeedback.notificationOccurred('error');
        alert('Ошибка: ' + e.message);
    }
}

// Load initial tab
loadTab('profile');
```

---

## Часть 4: Добавить API endpoints в Go backend

### 4.1 Добавить `/api/me` endpoint

В `internal/webapp/handler.go`:

```go
func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
    // В реальности: валидация Telegram Web App initData
    // Пока: принимаем user_id из query
    userID := r.URL.Query().Get("user_id")
    if userID == "" {
        http.Error(w, "user_id required", http.StatusBadRequest)
        return
    }

    var uid int64
    fmt.Sscanf(userID, "%d", &uid)

    user, err := h.services.User.GetUserByTGID(r.Context(), uid)
    if err != nil {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{"user": user})
}
```

### 4.2 Зарегистрировать route

```go
func (h *Handler) RegisterRoutes(r chi.Router) {
    r.Route("/api", func(r chi.Router) {
        r.Get("/cities", h.listCities)
        r.Get("/cities/{id}", h.getCity)
        r.Get("/events", h.listEvents)
        r.Get("/resources/{cityId}", h.cityResources)
        r.Get("/works", h.listWorks)
        r.Get("/works/{direction}", h.worksByDirection)
        r.Get("/me", h.getMe)                // NEW
        r.Post("/work/start", h.startWork)   // NEW
    })
}
```

---

## Часть 5: Деплой Web App

### 5.1 Создай папку

```bash
mkdir -p /opt/yachman/webapp/{css,js,assets}
```

### 5.2 Закинь файлы

Скопируй `index.html`, `css/style.css`, `js/api.js`, `js/app.js` в `/opt/yachman/webapp/`

### 5.3 Проверь

Открой в браузере: `https://danmotive.duckdns.org/webapp/`

Должна открыться страница с профилем.

### 5.4 Проверь в Telegram

1. Открой бота
2. Нажми `/start`
3. Нажми кнопку "🎮 Играть" (если настроена) или открой Web App через меню

---

## Часть 6: Валидация Telegram Web App (важно для безопасности!)

### 6.1 Что это

Telegram передаёт `initData` — подписанную строку с данными пользователя. Backend **обязан** её проверять, чтобы мошенник не мог подделать `user_id`.

### 6.2 Как проверять

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "net/url"
    "sort"
    "strings"
    "time"
)

func ValidateTelegramWebApp(initData string, botToken string) (map[string]string, error) {
    parsed, err := url.ParseQuery(initData)
    if err != nil {
        return nil, err
    }

    // Extract hash and remove it from data
    hash := parsed.Get("hash")
    parsed.Del("hash")

    // Sort remaining params
    var dataCheckParts []string
    for key, values := range parsed {
        for _, v := range values {
            dataCheckParts = append(dataCheckParts, key+"="+v)
        }
    }
    sort.Strings(dataCheckParts)
    dataCheckString := strings.Join(dataCheckParts, "\n")

    // HMAC-SHA256 with bot token
    secret := hmac.New(sha256.New, []byte("WebAppData"))
    secret.Write([]byte(botToken))
    mac := hmac.New(sha256.New, secret.Sum(nil))
    mac.Write([]byte(dataCheckString))
    expectedHash := hex.EncodeToString(mac.Sum(nil))

    if hash != expectedHash {
        return nil, fmt.Errorf("invalid hash")
    }

    // Check auth_date (not older than 24 hours)
    authDate := parsed.Get("auth_date")
    if authDate != "" {
        ts, _ := strconv.ParseInt(authDate, 10, 64)
        if time.Now().Unix()-ts > 86400 {
            return nil, fmt.Errorf("initData expired")
        }
    }

    result := make(map[string]string)
    for key, values := range parsed {
        if len(values) > 0 {
            result[key] = values[0]
        }
    }
    return result, nil
}
```

### 6.3 Использование в Web App API

```go
func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
    initData := r.Header.Get("X-Telegram-Init-Data")
    if initData == "" {
        initData = r.URL.Query().Get("initData")
    }

    data, err := webapp.ValidateTelegramWebApp(initData, h.botToken)
    if err != nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    user, err := h.services.User.GetUserByTGID(r.Context(), ...)
    // ...
}
```

---

## Чеклист

- [ ] DNS `danmotive.duckdns.org` → IP сервера
- [ ] Nginx установлен
- [ ] Let's Encrypt сертификат получен
- [ ] `https://danmotive.duckdns.org/health` отвечает 200
- [ ] Web App создан в BotFather
- [ ] Frontend файлы в `/opt/yachman/webapp/`
- [ ] `https://danmotive.duckdns.org/webapp/` открывается
- [ ] Telegram Web App SDK загружается
- [ ] API `/api/me` работает
- [ ] Валидация `initData` настроена
