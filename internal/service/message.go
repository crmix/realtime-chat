package service

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/realtime-chat/internal/domain"
)

type MessageService struct {
	messages domain.MessageRepository
	rooms    domain.RoomService
	bus      domain.EventBus
	log      *slog.Logger
}

func NewMessageService(messages domain.MessageRepository, rooms domain.RoomService, bus domain.EventBus, log *slog.Logger) *MessageService {
	return &MessageService{messages: messages, rooms: rooms, bus: bus, log: log}
}

const maxMessageLen = 4000

func (s *MessageService) Send(ctx context.Context, roomID, userID uuid.UUID, content string) (*domain.Message, error) {
	content = strings.TrimSpace(content)
	if content == "" || len(content) > maxMessageLen {
		return nil, domain.ErrInvalidInput
	}
	if err := s.rooms.EnsureMember(ctx, roomID, userID); err != nil {
		return nil, err
	}

	m := &domain.Message{
		ID:        uuid.New(),
		RoomID:    roomID,
		UserID:    userID,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.messages.Create(ctx, m); err != nil {
		return nil, err
	}
	if err := s.bus.PublishMessage(ctx, *m); err != nil {
		// Persistence already succeeded; log and move on so the caller still
		// gets the saved message. Subscribers on other nodes simply miss the
		// realtime fan-out for this delivery.
		s.log.ErrorContext(ctx, "publish message", "err", err, "message_id", m.ID)
	}
	return m, nil
}

func (s *MessageService) History(ctx context.Context, roomID uuid.UUID, before time.Time, limit int) ([]domain.Message, error) {
	return s.messages.ListByRoom(ctx, roomID, domain.HistoryFilter{Before: before, Limit: limit})
}
