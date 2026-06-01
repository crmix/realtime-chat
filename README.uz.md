# realtime-chat

> 🇬🇧 [Read in English](README.md)

Real vaqt rejimida ishlovchi chat backend — Go (chi + gorilla WebSocket),
PostgreSQL va Redis Pub/Sub ustida qurilgan, **Clean Architecture**
tamoyillariga amal qilingan holda yozilgan. `/` yo'lida oddiy single-page
brauzer mijozi bor — klient kod yozmasdan sinab ko'rishingiz mumkin.

## Imkoniyatlari (MVP)

- Foydalanuvchini ro'yxatdan o'tkazish va kirish (JWT)
- Public va private xonalar — yaratish, qo'shilish, chiqish
- Xona ichida xabar yuborish va sahifalangan tarixini olish (REST)
- WebSocket orqali real vaqtli xabar yetkazish
- Redis Pub/Sub orqali multi-node fan-out (horizontal scale uchun)
- `slog` asosida tuzilgan loglar, graceful shutdown

## Tezda ishga tushirish

**Talab:** Docker. (Go 1.25 — faqat Docker'dan tashqarida ishlatmoqchi
bo'lsangiz.)

```bash
git clone <repo-url>
cd realtime-chat
cp .env.example .env          # ixtiyoriy: o'z JWT_SECRET ni qo'ying
make docker-up                # Postgres, Redis, migratsiya va serverni ko'taradi
```

Brauzeringizda <http://localhost:8080/> ni oching. Foydalanuvchini ro'yxatdan
o'tkazing, public xona yarating, xabar yuboring. Boshqa brauzer oynasida (yoki
boshqa profilda — localStorage alohida bo'lishi uchun) yana bitta
foydalanuvchini ro'yxatdan o'tkazing, public ro'yxatdan o'sha xonaga
qo'shiling — xabarlar real vaqtda almashinishini ko'rasiz.

To'xtatish: terminalda `Ctrl+C`, agar Postgres ma'lumotlarini ham tozalamoqchi
bo'lsangiz keyin `make docker-down`.

### Lokal Go bilan ishga tushirish

```bash
cp .env.example .env
# .env ni tahrirlang, ayniqsa JWT_SECRET ni almashtiring

# golang-migrate o'rnatish (bir martalik):
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Postgres va Redis ni shaxsiy ravishda ko'taring, so'ng:
make migrate-up
make run
```

### Ikki noutbukli demo (Cloudflare quick tunnel)

Chat ma'lumotlar bazasi bitta kompyuterda turadi, shuning uchun ikki kishi
turli noutbuklardan haqiqatdan ham bir-biriga xabar yuborish uchun
**biringiz** stackni host qilasiz, ikkinchingiz ommaviy URL orqali ulanasiz.
Eng oson yo'l — bepul Cloudflare quick tunnel (account kerak emas):

```bash
# cloudflared o'rnatish (Ubuntu/Debian — boshqa OS uchun cloudflared docs)
curl -L -o cloudflared.deb \
  https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
sudo dpkg -i cloudflared.deb

# bitta terminalda stackni host qiling:
make docker-up

# boshqa terminalda ommaviy URL oching:
cloudflared tunnel --url http://localhost:8080
```

cloudflared `https://<random>.trycloudflare.com` URL'ini chiqaradi. Shu URL'ni
do'stingizga yuboring. Ikkalangiz brauzerda oching, account yarating, bir xil
public xonaga qo'shiling, suhbatlashing. Sahifa HTTPS'ni aniqlab WebSocket'ni
avtomatik `wss://` ga o'tkazadi.

**LAN orqali (alternativa):** agar bir xil Wi-Fi'da bo'lsangiz, do'stingiz
to'g'ridan-to'g'ri `http://<sizning-lan-ip>:8080/` ni ochishi kifoya — tunnel
shart emas (firewall'da 8080 ochiq bo'lsin).

### Ommaviy ulashishdan oldin

- `.env` gitignore'da — lokal `JWT_SECRET` sizniki bilan birga ketmaydi.
  Docker Compose stack'ining o'z `JWT_SECRET`'i
  `deployments/docker-compose.yml` da turibdi — uni boshqa odamlarga
  ko'rsatishdan oldin almashtiring (aks holda token soxtalashtirilishi
  mumkin).
- Default'da `HTTP_ALLOWED_ORIGINS=*` — demo uchun yaxshi, production uchun
  cheklab qo'ying.

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

## API

Barcha autentifikatsiya talab qiluvchi endpointlar `Authorization: Bearer
<token>` headerini kutadi.

### Auth

| Metod | Yo'l                          | Tavsif                          |
|-------|-------------------------------|---------------------------------|
| POST  | `/api/v1/auth/register`       | foydalanuvchini ro'yxatga olish |
| POST  | `/api/v1/auth/login`          | tizimga kirish, token olish     |
| GET   | `/api/v1/me`                  | joriy foydalanuvchi             |

### Xonalar

| Metod | Yo'l                                  | Tavsif                       |
|-------|---------------------------------------|------------------------------|
| GET   | `/api/v1/rooms`                       | mening xonalarim             |
| GET   | `/api/v1/rooms/public`                | barcha public xonalar        |
| POST  | `/api/v1/rooms`                       | yangi xona yaratish          |
| GET   | `/api/v1/rooms/{id}`                  | xona ma'lumotlari            |
| POST  | `/api/v1/rooms/{id}/join`             | qo'shilish (faqat public)    |
| POST  | `/api/v1/rooms/{id}/leave`            | chiqish                      |
| GET   | `/api/v1/rooms/{id}/members`          | a'zolar ro'yxati             |
| GET   | `/api/v1/rooms/{id}/messages`         | tarix (`before`, `limit`)    |
| POST  | `/api/v1/rooms/{id}/messages`         | xabar yuborish (REST)        |

### Realtime

`GET /ws?token=<jwt>` yoki `Authorization` header bilan WebSocket'ni ulang.

Inbound xabarlar formati:

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
xabar lokal subscribers'ga yetkaziladi. Shu sababli load balancer ortida
xohlagancha instance ko'paytirishingiz mumkin.

## Konfiguratsiya

`.env.example` faylida barcha qabul qilinadigan o'zgaruvchilar bor.
`JWT_SECRET` va `POSTGRES_DSN` — majburiy.

## Yo'l xaritasi (keyingi qadamlar)

- Online presence (Redis `SETEX user:<id>:online`)
- Typing indicator (transient WS event)
- Unit + integration testlar (testcontainers-go)
- Rate limiting (per-user, per-room)
- Refresh tokenlar, parolni tiklash
- Frontend (React / Vue) misol
