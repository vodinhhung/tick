package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDKey contextKey = "userID"

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

func JWTAuth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"code": "MISSING_TOKEN", "message": "Authorization header is required"})
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"code": "INVALID_TOKEN", "message": "Authorization header must be Bearer <token>"})
				return
			}

			tokenString := parts[1]
			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			if err != nil {
				if errors.Is(err, jwt.ErrTokenExpired) {
					writeJSON(w, http.StatusUnauthorized, map[string]string{"code": "TOKEN_EXPIRED", "message": "Token has expired"})
				} else {
					writeJSON(w, http.StatusUnauthorized, map[string]string{"code": "INVALID_TOKEN", "message": "Invalid token"})
				}
				return
			}

			if !token.Valid {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"code": "INVALID_TOKEN", "message": "Invalid token"})
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data) //nolint:errcheck
}
