# 🚀 ЯчМан — Деплой на сервер

## Быстрый старт (одна команда)

```bash
# Склонируйте проект и запустите
git clone <repo-url> && cd yachman
sudo ./deploy.sh
```

Скрипт сам:
- Установит Go 1.22 и PostgreSQL
- Создаст базу данных и пользователя
- Соберёт бинарник (~12 МБ)
- Настроит systemd-сервис
- Запустит бота

---

## Требования

- Linux (Ubuntu 22.04+ / Debian 12+ / AlmaLinux 9+)
- 1+ CPU, 512+ RAM, 1+ GB SSD
- Root-доступ
- Telegram Bot Token (от @BotFather)

---

## Ручная установка

### 1. Go

```bash
wget -q https://go.dev/dl/go1.22.4.linux-amd64.tar.gz -O /tmp/go.tar.gz
sudo tar -C /usr/local -xzf /tmp/go.tar.gz
rm /tmp/go.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

### 2. PostgreSQL

```bash
# Ubuntu / Debian
sudo apt update && sudo apt install -y postgresql postgresql-client

# CentOS / AlmaLinux
sudo dnf install -y postgresql-server postgresql
sudo postgresql-setup --initdb

# Запуск
sudo systemctl enable --now postgresql
```

### 3. Создание БД

```bash
sudo -u postgres psql -c "CREATE USER yachman WITH PASSWORD 'ваш_пароль';"
sudo -u postgres psql -c "CREATE DATABASE yachman OWNER yachman;"
```

### 4. Сборка

```bash
cd /opt/yachman
go mod tidy
go build -ldflags="-s -w" -o yachman ./cmd/server/
# Бинарник ~12 МБ, без зависимостей
ls -lh yachman
```

### 5. Конфигурация

```bash
cat > .env << 'EOF'
DATABASE_URL=postgres://yachman:ваш_пароль@localhost:5432/yachman?sslmode=disable
BOT_TOKEN=токен_из_BotFather
SERVER_PORT=8080
ENV=production
EOF
```

### 6. systemd

```bash
cat > /etc/systemd/system/yachman.service << 'EOF'
[Unit]
Description=YachMan Game Server
After=network.target postgresql.service

[Service]
Type=simple
ExecStart=/opt/yachman/yachman
WorkingDirectory=/opt/yachman
Restart=always
RestartSec=5
EnvironmentFile=/opt/yachman/.env

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now yachman
```

---

## Управление

```bash
# Статус
systemctl status yachman

# Логи
journalctl -u yachman -f

# Перезапуск
systemctl restart yachman

# Остановка
systemctl stop yachman

# Health check
curl http://localhost:8080/health
```

---

## Обновление

```bash
cd /opt/yachman
git pull
go build -ldflags="-s -w" -o yachman ./cmd/server/
systemctl restart yachman
```

---

## Бэкап БД

```bash
# Ручной
pg_dump -U yachman yachman | gzip > backup_$(date +%Y%m%d).sql.gz

# Автоматический (cron)
crontab -e
0 3 * * * pg_dump -U yachman yachman | gzip > /opt/yachman/backups/backup_$(date +\%Y\%m\%d).sql.gz
```

---

## Создание Telegram-бота

1. Откройте **@BotFather** в Telegram
2. `/newbot` → введите имя → username
3. Скопируйте токен в `.env`

Команды для @BotFather (`/setcommands`):
```
start - Начать
help - Справка
profile - Профиль
balance - Баланс
daily - Бонус
cities - Города
study - Обучение
notifications - Уведомления
```

---

## Удаление

```bash
sudo ./undeploy.sh
```
