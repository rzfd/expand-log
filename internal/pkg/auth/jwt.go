package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rzfd/expand/internal/pkg/logging"
	"golang.org/x/crypto/bcrypt"
)

type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	logging.FromContext(nil).Info().Str("ttl", ttl.String()).Msg("auth new token manager")
	return &TokenManager{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

func (m *TokenManager) Generate(userID int64) (string, time.Time, error) {
	logging.FromContext(nil).Info().Int64("user_id", userID).Msg("auth token generate started")
	expiresAt := time.Now().UTC().Add(m.ttl)
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			Subject:   fmt.Sprintf("%d", userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		logging.FromContext(nil).Error().Err(err).Int64("user_id", userID).Msg("auth token generate failed")
		return "", time.Time{}, err
	}

	logging.FromContext(nil).Info().Int64("user_id", userID).Msg("auth token generate completed")
	return signed, expiresAt, nil
}

func (m *TokenManager) Parse(token string) (*Claims, error) {
	logging.FromContext(nil).Info().Msg("auth token parse started")
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			logging.FromContext(nil).Warn().Msg("auth token parse unexpected signing method")
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		logging.FromContext(nil).Warn().Err(err).Msg("auth token parse failed")
		return nil, err
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		logging.FromContext(nil).Warn().Msg("auth token parse invalid claims")
		return nil, fmt.Errorf("invalid token")
	}

	logging.FromContext(nil).Info().Int64("user_id", claims.UserID).Msg("auth token parse completed")
	return claims, nil
}

func HashPassword(password string) (string, error) {
	logging.FromContext(nil).Info().Msg("auth hash password started")
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logging.FromContext(nil).Error().Err(err).Msg("auth hash password failed")
		return "", err
	}
	logging.FromContext(nil).Info().Msg("auth hash password completed")
	return string(hashed), nil
}

func ComparePassword(hashed, password string) error {
	logging.FromContext(nil).Info().Msg("auth compare password started")
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))
	if err != nil {
		logging.FromContext(nil).Warn().Err(err).Msg("auth compare password failed")
		return err
	}
	logging.FromContext(nil).Info().Msg("auth compare password completed")
	return nil
}
