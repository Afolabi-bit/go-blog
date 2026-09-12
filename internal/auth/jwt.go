package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	Role  string `json:"role"`
}

func CreateToken(jwtSecret string, userID string, email string, role string) (string, error) {
	now := time.Now()
	expiry := now.Add(time.Hour * 24 * 30)

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiry),
			Issuer:    "blog-api",
		},
		Email: email,
		Role:  role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("Failed to sign token: %v", err)
	}

	return signed, nil
}

func ParseToken(jwtSecret string, tokenStr string) (Claims, error) {
	var claims Claims

	parsed, err := jwt.ParseWithClaims(tokenStr, &claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})

	if err != nil {
		return Claims{}, fmt.Errorf("failed to parse token: %v", err)
	}

	if claims, ok := parsed.Claims.(*Claims); ok && parsed.Valid {
		return *claims, nil
	}

	return Claims{}, fmt.Errorf("invalid token")
}
