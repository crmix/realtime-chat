package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/realtime-chat/internal/domain"
)

type RoomRepository struct {
	pool *pgxpool.Pool
}

func NewRoomRepository(pool *pgxpool.Pool) *RoomRepository {
	return &RoomRepository{pool: pool}
}

func (r *RoomRepository) Create(ctx context.Context, room *domain.Room) error {
	const q = `
		INSERT INTO rooms (id, name, type, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.pool.Exec(ctx, q, room.ID, room.Name, string(room.Type), room.CreatedBy, room.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return err
	}
	return nil
}

func (r *RoomRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Room, error) {
	const q = `SELECT id, name, type, created_by, created_at FROM rooms WHERE id = $1`
	var room domain.Room
	var rt string
	err := r.pool.QueryRow(ctx, q, id).Scan(&room.ID, &room.Name, &rt, &room.CreatedBy, &room.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	room.Type = domain.RoomType(rt)
	return &room, nil
}

func (r *RoomRepository) ListForUser(ctx context.Context, userID uuid.UUID) ([]domain.Room, error) {
	const q = `
		SELECT r.id, r.name, r.type, r.created_by, r.created_at
		FROM rooms r
		JOIN room_members rm ON rm.room_id = r.id
		WHERE rm.user_id = $1
		ORDER BY r.created_at DESC
	`
	return r.queryRooms(ctx, q, userID)
}

func (r *RoomRepository) ListPublic(ctx context.Context, limit, offset int) ([]domain.Room, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	const q = `
		SELECT id, name, type, created_by, created_at
		FROM rooms
		WHERE type = 'public'
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	return r.queryRooms(ctx, q, limit, offset)
}

func (r *RoomRepository) queryRooms(ctx context.Context, q string, args ...any) ([]domain.Room, error) {
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Room{}
	for rows.Next() {
		var room domain.Room
		var rt string
		if err := rows.Scan(&room.ID, &room.Name, &rt, &room.CreatedBy, &room.CreatedAt); err != nil {
			return nil, err
		}
		room.Type = domain.RoomType(rt)
		out = append(out, room)
	}
	return out, rows.Err()
}

func (r *RoomRepository) AddMember(ctx context.Context, roomID, userID uuid.UUID) error {
	const q = `
		INSERT INTO room_members (room_id, user_id, joined_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (room_id, user_id) DO NOTHING
	`
	_, err := r.pool.Exec(ctx, q, roomID, userID)
	return err
}

func (r *RoomRepository) RemoveMember(ctx context.Context, roomID, userID uuid.UUID) error {
	const q = `DELETE FROM room_members WHERE room_id = $1 AND user_id = $2`
	_, err := r.pool.Exec(ctx, q, roomID, userID)
	return err
}

func (r *RoomRepository) IsMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error) {
	const q = `SELECT 1 FROM room_members WHERE room_id = $1 AND user_id = $2`
	var x int
	err := r.pool.QueryRow(ctx, q, roomID, userID).Scan(&x)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *RoomRepository) ListMembers(ctx context.Context, roomID uuid.UUID) ([]domain.RoomMember, error) {
	const q = `
		SELECT room_id, user_id, joined_at FROM room_members
		WHERE room_id = $1
		ORDER BY joined_at ASC
	`
	rows, err := r.pool.Query(ctx, q, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.RoomMember{}
	for rows.Next() {
		var m domain.RoomMember
		if err := rows.Scan(&m.RoomID, &m.UserID, &m.JoinedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
