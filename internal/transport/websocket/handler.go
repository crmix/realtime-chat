package websocket

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/realtime-chat/internal/domain"
	"github.com/realtime-chat/internal/transport/http/middleware"
)

type Handler struct {
	hub      *Hub
	auth     domain.AuthService
	rooms    domain.RoomService
	messages domain.MessageService
	log      *slog.Logger
	upgrader websocket.Upgrader
}

// AllowedOriginFunc is the origin policy for the WebSocket upgrader. The
// caller passes either a list of explicit origins (handled here) or "*" for
// no origin check. Production deployments should configure a tight list.
func NewHandler(hub *Hub, auth domain.AuthService, rooms domain.RoomService, messages domain.MessageService, log *slog.Logger, allowedOrigins []string) *Handler {
	allowAny := false
	allow := map[string]struct{}{}
	for _, o := range allowedOrigins {
		if o == "*" {
			allowAny = true
		}
		allow[o] = struct{}{}
	}

	return &Handler{
		hub:      hub,
		auth:     auth,
		rooms:    rooms,
		messages: messages,
		log:      log,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" || allowAny {
					return true
				}
				_, ok := allow[origin]
				return ok
			},
		},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := middleware.TokenFromRequest(r)
	if token == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID, err := h.auth.ParseToken(token)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Debug("ws upgrade failed", "err", err)
		return
	}

	client := newClient(conn, h.hub, h.rooms, h.messages, h.log, userID)
	h.hub.register <- client

	// Hub's root context (NOT r.Context, which is cancelled post-hijack)
	go client.writePump()
	go client.readPump(h.hub.Context())
}
