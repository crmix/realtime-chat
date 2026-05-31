package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/realtime-chat/internal/domain"
)

type MessageRepository struct {
	pool *pgxpool.Pool
}

func NewMessageRepository(pool *pgxpool.Pool) *MessageRepository {
	return &MessageRepository{pool: pool}
}

func (r *MessageRepository) Create(ctx context.Context, m *domain.Message) error {
	const q = `
		INSERT INTO messages (id, room_id, user_id, content, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING (SELECT username FROM users WHERE id = $3)
	`
	return r.pool.QueryRow(ctx, q, m.ID, m.RoomID, m.UserID, m.Content, m.CreatedAt).Scan(&m.Username)
}

func (r *MessageRepository) ListByRoom(ctx context.Context, roomID uuid.UUID, f domain.HistoryFilter) ([]domain.Message, error) {
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	before := f.Before
	if before.IsZero() {
		before = time.Now().Add(time.Hour) // future sentinel
	}

	const q = `
		SELECT m.id, m.room_id, m.user_id, u.username, m.content, m.created_at
		FROM messages m
		JOIN users u ON u.id = m.user_id
		WHERE m.room_id = $1 AND m.created_at < $2
		ORDER BY m.created_at DESC
		LIMIT $3
	`
	rows, err := r.pool.Query(ctx, q, roomID, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Message{}
	for rows.Next() {
		var m domain.Message
		if err := rows.Scan(&m.ID, &m.RoomID, &m.UserID, &m.Username, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
