#!/bin/bash
set -e

# ── Цвета ──────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

ok()   { echo -e "${GREEN}✓${NC} $1"; }
warn() { echo -e "${YELLOW}⚠${NC} $1"; }
err()  { echo -e "${RED}✗${NC} $1"; }

# ── Проверка ОС ────────────────────────────────────────────
if [[ $EUID -ne 0 ]]; then
    err "Запустите от root: sudo ./deploy.sh"
    exit 1
fi

APP_NAME="yachman"
APP_DIR="/opt/$APP_NAME"
SERVICE_FILE="/etc/systemd/system/$APP_NAME.service"
DB_USER="yachman"
DB_NAME="yachman"

echo ""
echo "╔══════════════════════════════════════╗"
echo "║       ЯчМан — Установка на сервер    ║"
echo "╚══════════════════════════════════════╝"
echo ""

# ── 1. Go ──────────────────────────────────────────────────
if command -v go &>/dev/null; then
    ok "Go уже установлен: $(go version)"
else
    warn "Устанавливаю Go 1.22..."
    GO_VERSION="1.22.4"
    wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
    tar -C /usr/local -xzf /tmp/go.tar.gz
    rm /tmp/go.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile.d/go.sh
    ok "Go установлен: $(go version)"
fi

# ── 2. PostgreSQL ──────────────────────────────────────────
if command -v psql &>/dev/null; then
    ok "PostgreSQL уже установлен"
else
    warn "Устанавливаю PostgreSQL..."
    if command -v apt-get &>/dev/null; then
        apt-get update -qq
        apt-get install -y -qq postgresql postgresql-client
    elif command -v dnf &>/dev/null; then
        dnf install -y postgresql-server postgresql
        postgresql-setup --initdb
    elif command -v yum &>/dev/null; then
        yum install -y postgresql-server postgresql
        postgresql-setup --initdb
    fi
    ok "PostgreSQL установлен"
fi

# ── 3. Запуск PostgreSQL ───────────────────────────────────
if systemctl is-active --quiet postgresql; then
    ok "PostgreSQL уже запущен"
else
    systemctl enable postgresql
    systemctl start postgresql
    ok "PostgreSQL запущен"
fi

# ── 4. Создание БД ────────────────────────────────────────
DB_PASS=$(openssl rand -base64 16 | tr -dc 'a-zA-Z0-9' | head -c 20)

su - postgres -c "psql -tc \"SELECT 1 FROM pg_roles WHERE rolname='$DB_USER'\" | grep -q 1" || \
    su - postgres -c "psql -c \"CREATE USER $DB_USER WITH PASSWORD '$DB_PASS';\""

su - postgres -c "psql -tc \"SELECT 1 FROM pg_database WHERE datname='$DB_NAME'\" | grep -q 1" || \
    su - postgres -c "psql -c \"CREATE DATABASE $DB_NAME OWNER $DB_USER;\""

su - postgres -c "psql -c \"GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;\""
ok "База данных создана"

# ── 5. Клонирование / обновление проекта ────────────────────
if [ -d "$APP_DIR/.git" ]; then
    warn "Обновляю проект..."
    cd "$APP_DIR"
    git pull
    ok "Проект обновлён"
else
    warn "Клонирую проект..."
    rm -rf "$APP_DIR"
    mkdir -p "$APP_DIR"
    
    # Определяем директорию скрипта
    SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
    
    if [ -f "$SCRIPT_DIR/go.mod" ]; then
        # Запущено из папки проекта — копируем
        cp -r "$SCRIPT_DIR"/* "$APP_DIR/"
        cp -r "$SCRIPT_DIR"/.* "$APP_DIR/" 2>/dev/null || true
    else
        err "Не могу найти исходники. Запустите скрипт из папки проекта."
        exit 1
    fi
    ok "Проект скопирован в $APP_DIR"
fi

# ── 6. Сборка ──────────────────────────────────────────────
cd "$APP_DIR"
warn "Собираю бинарник..."
/usr/local/go/bin/go mod tidy 2>/dev/null || true
/usr/local/go/bin/go build -ldflags="-s -w" -o "$APP_NAME" ./cmd/server/
ok "Бинарник собран: $APP_DIR/$APP_NAME"
ls -lh "$APP_DIR/$APP_NAME"

# ── 7. Конфигурация ────────────────────────────────────────
ENV_FILE="$APP_DIR/.env"
if [ ! -f "$ENV_FILE" ]; then
    BOT_TOKEN=""
    echo ""
    echo "─────────────────────────────────────"
    echo "  Введите токен Telegram-бота"
    echo "  (получите у @BotFather)"
    echo "─────────────────────────────────────"
    read -rp "BOT_TOKEN: " BOT_TOKEN
    
    cat > "$ENV_FILE" << EOF
DATABASE_URL=postgres://$DB_USER:$DB_PASS@localhost:5432/$DB_NAME?sslmode=disable
BOT_TOKEN=$BOT_TOKEN
SERVER_PORT=8080
ENV=production
EOF
    ok ".env создан"
else
    ok ".env уже существует"
fi

# ── 8. systemd сервис ──────────────────────────────────────
cat > "$SERVICE_FILE" << EOF
[Unit]
Description=YachMan Game Server
After=network.target postgresql.service
Requires=postgresql.service

[Service]
Type=simple
User=root
WorkingDirectory=$APP_DIR
ExecStart=$APP_DIR/$APP_NAME
Restart=always
RestartSec=5
EnvironmentFile=$APP_DIR/.env
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "$APP_NAME"
systemctl restart "$APP_NAME"
ok "Сервис создан и запущен"

# ── 9. Проверка ────────────────────────────────────────────
sleep 2
if systemctl is-active --quiet "$APP_NAME"; then
    ok "Сервер работает!"
else
    err "Сервер не запустился. Проверьте логи:"
    echo "  journalctl -u $APP_NAME -n 50"
    exit 1
fi

# ── 10. Firewall ───────────────────────────────────────────
if command -v ufw &>/dev/null; then
    ufw allow 8080/tcp >/dev/null 2>&1
    ok "Firewall: порт 8080 открыт"
fi

if command -v firewall-cmd &>/dev/null; then
    firewall-cmd --permanent --add-port=8080/tcp >/dev/null 2>&1
    firewall-cmd --reload >/dev/null 2>&1
    ok "Firewall: порт 8080 открыт"
fi

# ── Готово ─────────────────────────────────────────────────
echo ""
echo "╔══════════════════════════════════════════════════════╗"
echo "║              Установка завершена!                    ║"
echo "╠══════════════════════════════════════════════════════╣"
echo "║                                                      ║"
echo "║  Бот:        $BOT_TOKEN"
echo "║  БД:         postgres://$DB_USER:***@localhost/$DB_NAME"
echo "║  Бинарник:   $APP_DIR/$APP_NAME"
echo "║  Конфиг:     $APP_DIR/.env"
echo "║  Сервис:     systemctl status $APP_NAME"
echo "║  Логи:       journalctl -u $APP_NAME -f"
echo "║                                                      ║"
echo "║  Health:     curl http://localhost:8080/health        ║"
echo "║                                                      ║"
echo "╚══════════════════════════════════════════════════════╝"
echo ""
echo "  Полезные команды:"
echo "    systemctl restart $APP_NAME    — перезапуск"
echo "    systemctl stop $APP_NAME       — остановка"
echo "    journalctl -u $APP_NAME -f     — логи в реальном времени"
echo ""
