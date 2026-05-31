package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type RoomType string

const (
	RoomTypePublic  RoomType = "public"
	RoomTypePrivate RoomType = "private"
	RoomTypeDirect  RoomType = "direct"
)

type Room struct {
	ID        uuid.UUID
	Name      string
	Type      RoomType
	CreatedBy uuid.UUID
	CreatedAt time.Time
}

type RoomMember struct {
	RoomID   uuid.UUID
	UserID   uuid.UUID
	JoinedAt time.Time
}

type RoomRepository interface {
	Create(ctx context.Context, r *Room) error
	GetByID(ctx context.Context, id uuid.UUID) (*Room, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]Room, error)
	ListPublic(ctx context.Context, limit, offset int) ([]Room, error)

	AddMember(ctx context.Context, roomID, userID uuid.UUID) error
	RemoveMember(ctx context.Context, roomID, userID uuid.UUID) error
	IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
	ListMembers(ctx context.Context, roomID uuid.UUID) ([]RoomMember, error)
}
