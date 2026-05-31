package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/realtime-chat/internal/domain"
)

type RoomService struct {
	rooms domain.RoomRepository
}

func NewRoomService(rooms domain.RoomRepository) *RoomService {
	return &RoomService{rooms: rooms}
}

func (s *RoomService) Create(ctx context.Context, creatorID uuid.UUID, name string, roomType domain.RoomType) (*domain.Room, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrInvalidInput
	}
	if roomType == "" {
		roomType = domain.RoomTypePublic
	}
	switch roomType {
	case domain.RoomTypePublic, domain.RoomTypePrivate, domain.RoomTypeDirect:
	default:
		return nil, domain.ErrInvalidInput
	}

	room := &domain.Room{
		ID:        uuid.New(),
		Name:      name,
		Type:      roomType,
		CreatedBy: creatorID,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.rooms.Create(ctx, room); err != nil {
		return nil, err
	}
	if err := s.rooms.AddMember(ctx, room.ID, creatorID); err != nil {
		return nil, err
	}
	return room, nil
}

func (s *RoomService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Room, error) {
	return s.rooms.GetByID(ctx, id)
}

func (s *RoomService) ListForUser(ctx context.Context, userID uuid.UUID) ([]domain.Room, error) {
	return s.rooms.ListForUser(ctx, userID)
}

func (s *RoomService) ListPublic(ctx context.Context, limit, offset int) ([]domain.Room, error) {
	return s.rooms.ListPublic(ctx, limit, offset)
}

func (s *RoomService) Join(ctx context.Context, roomID, userID uuid.UUID) error {
	room, err := s.rooms.GetByID(ctx, roomID)
	if err != nil {
		return err
	}
	if room.Type == domain.RoomTypePrivate || room.Type == domain.RoomTypeDirect {
		return domain.ErrForbidden
	}
	return s.rooms.AddMember(ctx, roomID, userID)
}

func (s *RoomService) Leave(ctx context.Context, roomID, userID uuid.UUID) error {
	return s.rooms.RemoveMember(ctx, roomID, userID)
}

func (s *RoomService) Members(ctx context.Context, roomID uuid.UUID) ([]domain.RoomMember, error) {
	return s.rooms.ListMembers(ctx, roomID)
}

func (s *RoomService) EnsureMember(ctx context.Context, roomID, userID uuid.UUID) error {
	ok, err := s.rooms.IsMember(ctx, roomID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrForbidden
	}
	return nil
}
