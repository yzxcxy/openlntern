package account

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var authSecret string
var authExpire time.Duration

type TokenClaims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func InitAuth(secret string, expireMinutes int) {
	authSecret = secret
	if expireMinutes <= 0 {
		expireMinutes = 60
	}
	authExpire = time.Duration(expireMinutes) * time.Minute
}

func GenerateToken(userID, role string) (string, int64, error) {
	if authSecret == "" {
		return "", 0, errors.New("auth secret not configured")
	}
	now := time.Now()
	expiresAt := now.Add(authExpire)
	claims := TokenClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(authSecret))
	if err != nil {
		return "", 0, err
	}
	return signed, expiresAt.Unix(), nil
}

func ParseToken(tokenString string) (*TokenClaims, error) {
	if authSecret == "" {
		return nil, errors.New("auth secret not configured")
	}
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(authSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// CurrentTokenSecret exposes the configured JWT secret to sibling business packages.
func CurrentTokenSecret() string {
	return authSecret
}
