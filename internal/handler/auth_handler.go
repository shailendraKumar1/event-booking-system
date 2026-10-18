package handler

import (
	"errors"
	"net/http"

	"event-booking-system/internal/domain"
	"event-booking-system/internal/service"
)

// AuthHandler handles authentication endpoints (SRP: only auth HTTP concerns).
type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var input service.RegisterInput
	if !decode(w, r, &input) {
		return
	}
	resp, err := h.authService.Register(input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrDuplicateEmail):
			writeError(w, http.StatusConflict, "email already registered")
		case errors.Is(err, domain.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid input: email, password (min 6 chars), name, and role (ORGANIZER/CUSTOMER) required")
		default:
			writeError(w, http.StatusInternalServerError, "registration failed")
		}
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input service.LoginInput
	if !decode(w, r, &input) {
		return
	}
	resp, err := h.authService.Login(input)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid email or password")
		} else {
			writeError(w, http.StatusInternalServerError, "login failed")
		}
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
