package handler

import (
	"net/http"

	"github.com/realtime-chat/internal/domain"
	"github.com/realtime-chat/internal/transport/http/dto"
	"github.com/realtime-chat/internal/transport/http/middleware"
)

type UserHandler struct {
	users domain.UserRepository
}

func NewUserHandler(users domain.UserRepository) *UserHandler {
	return &UserHandler{users: users}
}

func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	u, err := h.users.GetByID(r.Context(), uid)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.UserFrom(u))
}
