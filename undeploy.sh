#!/bin/bash
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${RED}⚠ Удаление ЯчМан${NC}"
echo ""
echo "Будет удалено:"
echo "  - Сервис /etc/systemd/system/yachman.service"
echo "  - Файлы /opt/yachman/"
echo "  - База данных yachman (ДАННЫЕ БУДУТ ПОТЕРЯНЫ!)"
echo ""

read -rp "Продолжить? (y/N): " confirm
if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo "Отмена."
    exit 0
fi

# Остановка сервиса
systemctl stop yachman 2>/dev/null || true
systemctl disable yachman 2>/dev/null || true
rm -f /etc/systemd/system/yachman.service
systemctl daemon-reload
echo -e "${GREEN}✓${NC} Сервис удалён"

# Удаление файлов
rm -rf /opt/yachman
echo -e "${GREEN}✓${NC} Файлы удалены"

# Удаление БД
su - postgres -c "psql -c \"DROP DATABASE IF EXISTS yachman;\"" 2>/dev/null || true
su - postgres -c "psql -c \"DROP USER IF EXISTS yachman;\"" 2>/dev/null || true
echo -e "${GREEN}✓${NC} База данных удалена"

echo ""
echo -e "${GREEN}ЯчМан полностью удалён.${NC}"
