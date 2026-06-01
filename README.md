# realtime-chat

> 🇺🇿 [O'zbek tilida o'qish](README.uz.md)

A realtime chat backend in Go (chi + gorilla WebSocket) with PostgreSQL and
Redis Pub/Sub for multi-instance message fan-out, built following **Clean
Architecture** principles. Ships with a minimal single-page browser client at
`/` so you can try it without writing any client code.

## Features (MVP)

- User registration and login (JWT)
- Public and private rooms — create, join, leave
- Send messages and fetch paginated history (REST)
- Realtime message delivery over WebSocket
- Multi-node fan-out via Redis Pub/Sub (horizontal scaling)
- Structured logs (`slog`), graceful shutdown

## Quick start

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

### Running locally with Go

```bash
cp .env.example .env
# edit .env, especially JWT_SECRET

# install golang-migrate (one-time):
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# bring up Postgres and Redis yourself, then:
make migrate-up
make run
```

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

## Architecture

Dependencies point *inward* — outer layers depend on inner layers, never the
other way around.

```
   ┌─────────────────────────────────────────────────────────────┐
   │  cmd/server            (wiring, main)                       │
   ├─────────────────────────────────────────────────────────────┤
   │  transport/http   transport/websocket                       │
   │  (handlers, dto, middleware, hub, client)                   │
   ├─────────────────────────────────────────────────────────────┤
   │  service          (Auth, Room, Message — use cases)         │
   ├─────────────────────────────────────────────────────────────┤
   │  domain           (entities, repository/service ports,      │
   │                    EventBus, errors — pure Go)              │
   ├─────────────────────────────────────────────────────────────┤
   │  repository/postgres  repository/redis                      │
   │  (port implementations)                                     │
   └─────────────────────────────────────────────────────────────┘
```

`internal/domain` holds business entities and ports (interfaces) — no external
dependencies. `service` implements use cases against those ports.
`repository` provides concrete implementations. `transport` exposes the
application over HTTP/WebSocket, depending on services. `cmd/server` wires
everything together.

## Project layout

```
.
├── cmd/server/main.go              # bootstrap, wiring, graceful shutdown
├── internal/
│   ├── config/                     # env-based configuration
│   ├── domain/                     # entities, ports, errors
│   ├── pkg/                        # jwt, hasher, logger, validator
│   ├── repository/
│   │   ├── postgres/               # UserRepo, RoomRepo, MessageRepo
│   │   └── redis/                  # EventBus (Pub/Sub)
│   ├── service/                    # AuthService, RoomService, MessageService
│   └── transport/
│       ├── http/                   # chi router, handlers, dto, middleware
│       └── websocket/              # Hub, Client, upgrade handler
├── migrations/                     # golang-migrate up/down files
├── deployments/                    # Dockerfile, docker-compose.yml
├── scripts/migrate.sh
└── Makefile
```

## API

All authenticated endpoints expect the `Authorization: Bearer <token>` header.

### Auth

| Method | Path                          | Description                  |
|--------|-------------------------------|------------------------------|
| POST   | `/api/v1/auth/register`       | register a new user          |
| POST   | `/api/v1/auth/login`          | log in, receive JWT          |
| GET    | `/api/v1/me`                  | current user                 |

### Rooms

| Method | Path                                  | Description                  |
|--------|---------------------------------------|------------------------------|
| GET    | `/api/v1/rooms`                       | list my rooms                |
| GET    | `/api/v1/rooms/public`                | list all public rooms        |
| POST   | `/api/v1/rooms`                       | create a new room            |
| GET    | `/api/v1/rooms/{id}`                  | room details                 |
| POST   | `/api/v1/rooms/{id}/join`             | join (public rooms only)     |
| POST   | `/api/v1/rooms/{id}/leave`            | leave                        |
| GET    | `/api/v1/rooms/{id}/members`          | list members                 |
| GET    | `/api/v1/rooms/{id}/messages`         | history (`before`, `limit`)  |
| POST   | `/api/v1/rooms/{id}/messages`         | send a message (REST)        |

### Realtime

Connect with `GET /ws?token=<jwt>` (or with an `Authorization` header).

Inbound frames:

```json
{ "type": "subscribe",   "room_id": "<uuid>" }
{ "type": "unsubscribe", "room_id": "<uuid>" }
{ "type": "message",     "room_id": "<uuid>", "content": "hello" }
{ "type": "ping" }
```

Outbound frames:

```json
{ "type": "subscribed",   "payload": { "room_id": "<uuid>" } }
{ "type": "message",      "payload": { "id": "...", "room_id": "...", "user_id": "...", "content": "...", "created_at": "..." } }
{ "type": "message.error","error": "forbidden" }
```

## Multi-node operation

`MessageService.Send` persists the message in Postgres and publishes it to a
Redis Pub/Sub channel (`REDIS_CHANNEL`, default `chat.messages`). Each server
instance subscribes to the channel via its WebSocket Hub; incoming events are
fanned out to local subscribers. Scale horizontally by running more instances
behind a load balancer.

## Configuration

`.env.example` lists every recognized variable. `JWT_SECRET` and
`POSTGRES_DSN` are required.

## Roadmap

- Online presence (Redis `SETEX user:<id>:online`)
- Typing indicators (transient WS event)
- Unit + integration tests (testcontainers-go)
- Rate limiting (per-user, per-room)
- Refresh tokens, password reset
- Example frontend (React / Vue)
