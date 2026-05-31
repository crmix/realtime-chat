# realtime-chat

A realtime chat backend in Go (chi + gorilla WebSocket) with PostgreSQL and
Redis Pub/Sub for multi-instance message fan-out. Ships with a minimal
single-page browser client at `/` so you can try it without writing any client
code.

## Quick start (English)

**Requires:** Docker. (Go 1.25 only if you want to run outside Docker.)

```bash
git clone <this-repo-url>
cd realtime-chat
cp .env.example .env          # optional: set a unique JWT_SECRET
make docker-up                # builds + boots Postgres, Redis, migrations, server
```

Open <http://localhost:8080/> in your browser. Register a user, create a public
room, send a message. Open the same URL in a second browser window (or another
profile so localStorage is separate), register a second user, join the room
from the public list — you'll see messages exchanged in real time.

To stop: `Ctrl+C` in the terminal, then `make docker-down` if you also want to
wipe the Postgres volume.

### Two-laptop demo (Cloudflare quick tunnel)

The chat database lives on one machine, so for two people on different laptops
to actually message each other, **one of you hosts** the stack and the other
connects through a public URL. The easiest way is a free Cloudflare quick
tunnel (no account required):

```bash
# install cloudflared (Ubuntu/Debian — see cloudflared docs for other OSes)
curl -L -o cloudflared.deb \
  https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
sudo dpkg -i cloudflared.deb

# in one terminal, host the stack:
make docker-up

# in another terminal, expose it:
cloudflared tunnel --url http://localhost:8080
```

cloudflared prints a `https://<random>.trycloudflare.com` URL. Share that URL
with your friend. Both of you open it in a browser, register accounts, join
the same public room, and chat. The page detects HTTPS and upgrades the
WebSocket to `wss://` automatically.

**LAN fallback:** if you're on the same Wi-Fi, the friend can just open
`http://<your-lan-ip>:8080/` instead — no tunnel needed (make sure your
firewall allows inbound 8080).

### Before sharing publicly

- `.env` is gitignored so your local `JWT_SECRET` won't leak. The Docker
  Compose stack has its own `JWT_SECRET` in `deployments/docker-compose.yml` —
  change it before exposing the service to anyone who shouldn't be able to
  forge tokens.
- `HTTP_ALLOWED_ORIGINS=*` is wide-open by default — fine for a demo, tighten
  it for production.

---

Real vaqt rejimida ishlovchi chat backend — Go, PostgreSQL, Redis Pub/Sub
ustida, **Clean Architecture** tamoyillariga amal qilingan holda yozilgan.

## Imkoniyatlari (MVP)

- Foydalanuvchini ro'yxatdan o'tkazish va kirish (JWT)
- Public va private xonalar yaratish, qo'shilish/chiqish
- Xona ichida xabar yuborish va tarixini olish (REST)
- WebSocket orqali real vaqtli xabar yetkazish
- Redis Pub/Sub orqali multi-node fan-out (horizontal scale uchun)
- `slog` asosida tuzilgan loglar, graceful shutdown

## Arxitektura

Dependency yo'nalishi *ichkarigaga* yo'nalgan — tashqi qatlam ichki qatlamga
bog'liq, lekin teskarisi yo'q.

```
   ┌─────────────────────────────────────────────────────────────┐
   │  cmd/server            (wiring, main)                       │
   ├─────────────────────────────────────────────────────────────┤
   │  transport/http   transport/websocket                       │
   │  (handlers, dto, middleware, hub, client)                   │
   ├─────────────────────────────────────────────────────────────┤
   │  service          (Auth, Room, Message — use-case lar)      │
   ├─────────────────────────────────────────────────────────────┤
   │  domain           (entities, repository/service portlari,   │
   │                    EventBus, xatoliklar — pure Go)          │
   ├─────────────────────────────────────────────────────────────┤
   │  repository/postgres  repository/redis                      │
   │  (port implementatsiyalari)                                 │
   └─────────────────────────────────────────────────────────────┘
```

`internal/domain` — biznes obyektlar va portlar (interfeyslar). U hech kimga
bog'liq emas. `service` — domain portlariga bog'lanib use-case'larni amalga
oshiradi. `repository` — portlarning konkret implementatsiyasini beradi.
`transport` — service'lardan foydalanib tashqi protokollarni (HTTP, WS)
ochadi. `cmd/server` — barchasini bog'laydi.

## Papkalar tuzilishi

```
.
├── cmd/server/main.go              # bootstrap, wiring, graceful shutdown
├── internal/
│   ├── config/                     # env-asosli konfiguratsiya
│   ├── domain/                     # entities, ports, errors
│   ├── pkg/                        # jwt, hasher, logger, validator
│   ├── repository/
│   │   ├── postgres/               # UserRepo, RoomRepo, MessageRepo
│   │   └── redis/                  # EventBus (Pub/Sub)
│   ├── service/                    # AuthService, RoomService, MessageService
│   └── transport/
│       ├── http/                   # chi router, handlers, dto, middleware
│       └── websocket/              # Hub, Client, upgrade handler
├── migrations/                     # golang-migrate uchun .up/.down fayllar
├── deployments/                    # Dockerfile, docker-compose.yml
├── scripts/migrate.sh
└── Makefile
```

## Tezda ishga tushirish

### Docker Compose bilan (eng oson yo'l)

```bash
docker compose -f deployments/docker-compose.yml up --build
```

Bu Postgres, Redis, migrate va serverni ko'taradi. Server `:8080` da
tinglaydi.

### Lokal Go bilan

```bash
cp .env.example .env
# .env ni tahrirlang, ayniqsa JWT_SECRET ni almashtiring

# golang-migrate o'rnatish (bir martalik):
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Postgres va Redis ni shaxsiy ravishda ko'taring, so'ng:
make migrate-up
make run
```

## API

Barcha autentifikatsiya talab qiluvchi endpointlar `Authorization: Bearer
<token>` headerini kutadi.

### Auth

| Metod | Yo'l                          | Tavsif                    |
|-------|-------------------------------|---------------------------|
| POST  | `/api/v1/auth/register`       | foydalanuvchini ro'yxatga olish |
| POST  | `/api/v1/auth/login`          | tizimga kirish, token olish |
| GET   | `/api/v1/me`                  | joriy foydalanuvchi          |

### Xonalar

| Metod | Yo'l                                  | Tavsif                       |
|-------|---------------------------------------|------------------------------|
| GET   | `/api/v1/rooms`                       | mening xonalarim             |
| GET   | `/api/v1/rooms/public`                | barcha public xonalar        |
| POST  | `/api/v1/rooms`                       | yangi xona yaratish          |
| GET   | `/api/v1/rooms/{id}`                  | xona ma'lumotlari            |
| POST  | `/api/v1/rooms/{id}/join`             | qo'shilish (faqat public)    |
| POST  | `/api/v1/rooms/{id}/leave`            | chiqish                       |
| GET   | `/api/v1/rooms/{id}/members`          | a'zolar ro'yxati             |
| GET   | `/api/v1/rooms/{id}/messages`         | tarix (`before`, `limit`)    |
| POST  | `/api/v1/rooms/{id}/messages`         | xabar yuborish (REST)        |

### Realtime

`GET /ws?token=<jwt>` yoki `Authorization` header bilan WebSocket'ni
ulang. Inbound xabarlar formati:

```json
{ "type": "subscribe",   "room_id": "<uuid>" }
{ "type": "unsubscribe", "room_id": "<uuid>" }
{ "type": "message",     "room_id": "<uuid>", "content": "salom" }
{ "type": "ping" }
```

Outbound:

```json
{ "type": "subscribed",   "payload": { "room_id": "<uuid>" } }
{ "type": "message",      "payload": { "id": "...", "room_id": "...", "user_id": "...", "content": "...", "created_at": "..." } }
{ "type": "message.error","error": "forbidden" }
```

## Multi-node ishlash

`MessageService.Send` xabarni Postgres'ga yozadi va Redis Pub/Sub kanaliga
(`REDIS_CHANNEL`, default `chat.messages`) e'lon qiladi. Har bir server
instance o'sha kanalga obuna bo'lgan WebSocket Hub'ga ega; kanalga kelgan
xabar lokal subscribers'ga yetkaziladi. Shu sababli xohlagancha
instance ko'paytirishingiz mumkin.

## Konfiguratsiya

`.env.example` faylida barcha qabul qilinadigan o'zgaruvchilar bor.
`JWT_SECRET` va `POSTGRES_DSN` — majburiy.

## Yo'l xaritasi (keyingi qadamlar)

- Online presence (Redis `SETEX user:<id>:online`)
- Typing indicator (transient WS event)
- `unit + integration` testlar (testcontainers-go)
- Rate limiting (per-user, per-room)
- Refresh tokenlar, parolni tiklash
- Frontend (React / Vue) misol
