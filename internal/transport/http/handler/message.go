package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/realtime-chat/internal/domain"
	"github.com/realtime-chat/internal/pkg/validator"
	"github.com/realtime-chat/internal/transport/http/dto"
	"github.com/realtime-chat/internal/transport/http/middleware"
)

type MessageHandler struct {
	svc       domain.MessageService
	rooms     domain.RoomService
	validator *validator.Validator
}

func NewMessageHandler(svc domain.MessageService, rooms domain.RoomService, v *validator.Validator) *MessageHandler {
	return &MessageHandler{svc: svc, rooms: rooms, validator: v}
}

func (h *MessageHandler) Send(w http.ResponseWriter, r *http.Request) {
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid room id")
		return
	}
	var req dto.SendMessageRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := h.validator.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	msg, err := h.svc.Send(r.Context(), roomID, middleware.UserID(r.Context()), req.Content)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.MessageFrom(msg))
}

func (h *MessageHandler) History(w http.ResponseWriter, r *http.Request) {
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid room id")
		return
	}
	if err := h.rooms.EnsureMember(r.Context(), roomID, middleware.UserID(r.Context())); err != nil {
		writeDomainError(w, err)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	var before time.Time
	if b := r.URL.Query().Get("before"); b != "" {
		t, err := time.Parse(time.RFC3339Nano, b)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid before timestamp")
			return
		}
		before = t
	}

	msgs, err := h.svc.History(r.Context(), roomID, before, limit)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MessagesFrom(msgs))
}
