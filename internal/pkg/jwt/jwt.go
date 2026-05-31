package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Manager issues and validates HS256 JWT access tokens.
type Manager struct {
	secret []byte
	ttl    time.Duration
	issuer string
}

func NewManager(secret string, ttl time.Duration, issuer string) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl, issuer: issuer}
}

type Claims struct {
	UserID uuid.UUID `json:"uid"`
	jwt.RegisteredClaims
}

func (m *Manager) Issue(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(m.secret)
}

func (m *Manager) Parse(token string) (uuid.UUID, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	if !parsed.Valid {
		return uuid.Nil, errors.New("invalid token")
	}
	if claims.UserID == uuid.Nil {
		return uuid.Nil, errors.New("token has no subject")
	}
	return claims.UserID, nil
}
