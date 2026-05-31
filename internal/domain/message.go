package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID        uuid.UUID
	RoomID    uuid.UUID
	UserID    uuid.UUID
	Content   string
	CreatedAt time.Time
}

// HistoryFilter is used to paginate message history. If Before is zero, the
// repository returns the most recent messages.
type HistoryFilter struct {
	Before time.Time
	Limit  int
}

type MessageRepository interface {
	Create(ctx context.Context, m *Message) error
	ListByRoom(ctx context.Context, roomID uuid.UUID, f HistoryFilter) ([]Message, error)
}
