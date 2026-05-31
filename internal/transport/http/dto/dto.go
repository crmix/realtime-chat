package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/realtime-chat/internal/domain"
)

type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=32"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func UserFrom(u *domain.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}

type CreateRoomRequest struct {
	Name string `json:"name" validate:"required,min=1,max=64"`
	Type string `json:"type" validate:"omitempty,oneof=public private"`
}

type RoomResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

func RoomFrom(r *domain.Room) RoomResponse {
	return RoomResponse{
		ID:        r.ID,
		Name:      r.Name,
		Type:      string(r.Type),
		CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt,
	}
}

func RoomsFrom(rs []domain.Room) []RoomResponse {
	out := make([]RoomResponse, 0, len(rs))
	for i := range rs {
		out = append(out, RoomFrom(&rs[i]))
	}
	return out
}

type SendMessageRequest struct {
	Content string `json:"content" validate:"required,min=1,max=4000"`
}

type MessageResponse struct {
	ID        uuid.UUID `json:"id"`
	RoomID    uuid.UUID `json:"room_id"`
	UserID    uuid.UUID `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func MessageFrom(m *domain.Message) MessageResponse {
	return MessageResponse{
		ID:        m.ID,
		RoomID:    m.RoomID,
		UserID:    m.UserID,
		Content:   m.Content,
		CreatedAt: m.CreatedAt,
	}
}

func MessagesFrom(ms []domain.Message) []MessageResponse {
	out := make([]MessageResponse, 0, len(ms))
	for i := range ms {
		out = append(out, MessageFrom(&ms[i]))
	}
	return out
}

type RoomMemberResponse struct {
	UserID   uuid.UUID `json:"user_id"`
	JoinedAt time.Time `json:"joined_at"`
}

func MembersFrom(ms []domain.RoomMember) []RoomMemberResponse {
	out := make([]RoomMemberResponse, 0, len(ms))
	for _, m := range ms {
		out = append(out, RoomMemberResponse{UserID: m.UserID, JoinedAt: m.JoinedAt})
	}
	return out
}

type ErrorResponse struct {
	Error string `json:"error"`
}
