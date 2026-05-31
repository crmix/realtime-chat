package handler

import (
	"net/http"

	"github.com/realtime-chat/internal/domain"
	"github.com/realtime-chat/internal/pkg/validator"
	"github.com/realtime-chat/internal/transport/http/dto"
)

type AuthHandler struct {
	svc       domain.AuthService
	validator *validator.Validator
}

func NewAuthHandler(svc domain.AuthService, v *validator.Validator) *AuthHandler {
	return &AuthHandler{svc: svc, validator: v}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := h.validator.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	u, token, err := h.svc.Register(r.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.AuthResponse{Token: token, User: dto.UserFrom(u)})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := h.validator.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	u, token, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.AuthResponse{Token: token, User: dto.UserFrom(u)})
}
