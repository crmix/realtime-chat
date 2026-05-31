package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AuthService handles user registration, authentication and token verification.
type AuthService interface {
	Register(ctx context.Context, username, email, password string) (*User, string, error)
	Login(ctx context.Context, email, password string) (*User, string, error)
	ParseToken(token string) (uuid.UUID, error)
}

// RoomService encapsulates room-related use cases.
type RoomService interface {
	Create(ctx context.Context, creatorID uuid.UUID, name string, roomType RoomType) (*Room, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Room, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]Room, error)
	ListPublic(ctx context.Context, limit, offset int) ([]Room, error)
	Join(ctx context.Context, roomID, userID uuid.UUID) error
	Leave(ctx context.Context, roomID, userID uuid.UUID) error
	Members(ctx context.Context, roomID uuid.UUID) ([]RoomMember, error)
	EnsureMember(ctx context.Context, roomID, userID uuid.UUID) error
}

// MessageService encapsulates messaging use cases.
type MessageService interface {
	Send(ctx context.Context, roomID, userID uuid.UUID, content string) (*Message, error)
	History(ctx context.Context, roomID uuid.UUID, before time.Time, limit int) ([]Message, error)
}
