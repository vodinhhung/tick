package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/api/idtoken"
	"gorm.io/gorm"

	"tick/be/internal/middleware"
	"tick/be/internal/model"
)

type AuthHandler struct {
	DB             *gorm.DB
	JWTSecret      string
	GoogleClientID string
}

func NewAuthHandler(db *gorm.DB, jwtSecret, googleClientID string) *AuthHandler {
	return &AuthHandler{
		DB:             db,
		JWTSecret:      jwtSecret,
		GoogleClientID: googleClientID,
	}
}

type googleAuthRequest struct {
	IDToken string `json:"id_token"`
}

type authResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      authUserInfo `json:"user"`
}

type authUserInfo struct {
	ID      uint   `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

type refreshResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req googleAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.IDToken == "" {
		respondError(w, http.StatusBadRequest, "MISSING_ID_TOKEN", "id_token field is required")
		return
	}

	payload, err := idtoken.Validate(context.Background(), req.IDToken, h.GoogleClientID)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "INVALID_ID_TOKEN", "Token failed Google verification")
		return
	}

	googleID, _ := payload.Claims["sub"].(string)
	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)
	givenName, _ := payload.Claims["given_name"].(string)
	familyName, _ := payload.Claims["family_name"].(string)
	emailVerified, _ := payload.Claims["email_verified"].(bool)

	user := &model.User{}
	result := h.DB.Where("google_id = ?", googleID).First(user)

	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		respondError(w, http.StatusInternalServerError, "SERVER_ERROR", "Database error")
		return
	}

	if result.RowsAffected > 0 {
		h.DB.Model(user).Updates(model.User{
			Email:         email,
			VerifiedEmail: emailVerified,
			Name:          name,
			GivenName:     givenName,
			FamilyName:    familyName,
			Picture:       picture,
		})
	} else {
		user = &model.User{
			GoogleID:      googleID,
			Email:         email,
			VerifiedEmail: emailVerified,
			Name:          name,
			GivenName:     givenName,
			FamilyName:    familyName,
			Picture:       picture,
		}
		if err := h.DB.Create(user).Error; err != nil {
			respondError(w, http.StatusInternalServerError, "SERVER_ERROR", "Failed to create user")
			return
		}
	}

	token, expiresAt, err := h.generateJWT(user.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "SERVER_ERROR", "Failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, authResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User: authUserInfo{
			ID:      user.ID,
			Email:   user.Email,
			Name:    user.Name,
			Picture: user.Picture,
		},
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == 0 {
		respondError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid token")
		return
	}

	token, expiresAt, err := h.generateJWT(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "SERVER_ERROR", "Failed to generate token")
		return
	}

	writeJSON(w, http.StatusOK, refreshResponse{
		Token:     token,
		ExpiresAt: expiresAt,
	})
}

func (h *AuthHandler) generateJWT(userID uint) (string, time.Time, error) {
	expiresAt := time.Now().UTC().Add(24 * time.Hour)

	claims := middleware.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.JWTSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}
