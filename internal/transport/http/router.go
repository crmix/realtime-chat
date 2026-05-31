package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/realtime-chat/internal/domain"
	"github.com/realtime-chat/internal/pkg/validator"
	"github.com/realtime-chat/internal/transport/http/handler"
	"github.com/realtime-chat/internal/transport/http/middleware"
)

// Deps bundles everything the HTTP router needs. Keeping it as a struct makes
// the cmd/server wiring explicit and easy to extend.
type Deps struct {
	Logger    *slog.Logger
	Validator *validator.Validator
	Auth      domain.AuthService
	Rooms     domain.RoomService
	Messages  domain.MessageService
	Users     domain.UserRepository
	WebSocket http.Handler
	Origins   []string
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recover(d.Logger))
	r.Use(middleware.RequestLogger(d.Logger))
	r.Use(middleware.CORS(d.Origins))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexHTML)
	})

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	authH := handler.NewAuthHandler(d.Auth, d.Validator)
	userH := handler.NewUserHandler(d.Users)
	roomH := handler.NewRoomHandler(d.Rooms, d.Validator)
	msgH := handler.NewMessageHandler(d.Messages, d.Rooms, d.Validator)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", authH.Register)
		r.Post("/auth/login", authH.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(d.Auth))

			r.Get("/me", userH.Me)

			r.Route("/rooms", func(r chi.Router) {
				r.Get("/", roomH.ListMine)
				r.Post("/", roomH.Create)
				r.Get("/public", roomH.ListPublic)
				r.Get("/{id}", roomH.Get)
				r.Post("/{id}/join", roomH.Join)
				r.Post("/{id}/leave", roomH.Leave)
				r.Get("/{id}/members", roomH.Members)
				r.Get("/{id}/messages", msgH.History)
				r.Post("/{id}/messages", msgH.Send)
			})
		})
	})

	if d.WebSocket != nil {
		r.Handle("/ws", d.WebSocket)
	}

	return r
}
