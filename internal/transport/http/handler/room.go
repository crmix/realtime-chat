package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/realtime-chat/internal/domain"
	"github.com/realtime-chat/internal/pkg/validator"
	"github.com/realtime-chat/internal/transport/http/dto"
	"github.com/realtime-chat/internal/transport/http/middleware"
)

type RoomHandler struct {
	svc       domain.RoomService
	validator *validator.Validator
}

func NewRoomHandler(svc domain.RoomService, v *validator.Validator) *RoomHandler {
	return &RoomHandler{svc: svc, validator: v}
}

func (h *RoomHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateRoomRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := h.validator.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	roomType := domain.RoomType(req.Type)
	if roomType == "" {
		roomType = domain.RoomTypePublic
	}

	room, err := h.svc.Create(r.Context(), middleware.UserID(r.Context()), req.Name, roomType)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.RoomFrom(room))
}

func (h *RoomHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	rooms, err := h.svc.ListForUser(r.Context(), middleware.UserID(r.Context()))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.RoomsFrom(rooms))
}

func (h *RoomHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	rooms, err := h.svc.ListPublic(r.Context(), limit, offset)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.RoomsFrom(rooms))
}

func (h *RoomHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid room id")
		return
	}
	room, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.RoomFrom(room))
}

func (h *RoomHandler) Join(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid room id")
		return
	}
	if err := h.svc.Join(r.Context(), id, middleware.UserID(r.Context())); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RoomHandler) Leave(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid room id")
		return
	}
	if err := h.svc.Leave(r.Context(), id, middleware.UserID(r.Context())); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RoomHandler) Members(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid room id")
		return
	}
	if err := h.svc.EnsureMember(r.Context(), id, middleware.UserID(r.Context())); err != nil {
		writeDomainError(w, err)
		return
	}
	members, err := h.svc.Members(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.MembersFrom(members))
}
