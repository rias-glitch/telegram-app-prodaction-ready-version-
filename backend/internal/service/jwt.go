package service

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte

func InitJWT() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		panic("JWT_SECRET is not set")
	}
	jwtSecret = []byte(secret)
}

func GenerateJWT(userID int64) (string, error) {
	now := time.Now().Unix()
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     now,
		"nbf":     now,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ParseJWT(tokenString string) (int64, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return 0, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("invalid claims")
	}

	// проверяет ограничения по времени для выполнения запросов
	now := time.Now().Unix()
	if exp, ok := claims["exp"].(float64); ok {
		if int64(exp) < now {
			return 0, errors.New("token expired")
		}
	}
	if nbf, ok := claims["nbf"].(float64); ok {
		if int64(nbf) > now {
			return 0, errors.New("token not valid yet")
		}
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, errors.New("user_id not found")
	}

	return int64(userID), nil
}
