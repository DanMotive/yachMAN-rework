# YachMan (ЯчМан)

Global social economic Telegram game — Bot + Web App.

## Tech Stack

- **Backend:** Go 1.22+
- **Database:** PostgreSQL 15+
- **Frontend:** Telegram Web App (embedded browser)
- **Bot:** Telegram Bot API via long polling or webhook

## Quick Start

```bash
# 1. Install Go 1.22+ https://go.dev/dl/
# 2. Install PostgreSQL 15+

# 3. Create database
createdb yachman

# 4. Copy and edit config
cp .env.example .env
# Edit .env with your DATABASE_URL and BOT_TOKEN

# 5. Run
go run ./cmd/server
```

Server runs migrations and seeds data automatically on first start.

## Project Structure

```
cmd/server/       — Entry point
internal/
  config/         — Environment config
  db/             — Migrations + seed data
  enums/          — Game constants & enums
  models/         — Domain models (structs + SQL helpers)
migrations/       — SQL migration files
```

## Game Overview

- **Telegram Group = City** — social/economic actions happen here
- **DM = Personal hub** — profile, navigation, education
- **300 jobs** across 20 skill directions
- **65 business types** with NPC workforce
- **Corporations** with real-player employees & stock trading
- **10 resources** with dynamic pricing
- **18 education programs** with 12h lesson intervals

## License

Proprietary — see LICENSE
