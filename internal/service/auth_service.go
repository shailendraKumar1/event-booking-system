package service

import (
	"strings"
	"time"

	"event-booking-system/internal/config"
	"event-booking-system/internal/domain"
	"event-booking-system/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles user registration, login, and JWT token generation.
// DIP: depends on repository.UserRepository abstraction, not concrete implementation.
type AuthService struct {
	userRepo  repository.UserRepository
	jwtSecret []byte
	jwtExpiry time.Duration
}

func NewAuthService(userRepo repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: []byte(cfg.JWTSecret),
		jwtExpiry: cfg.JWTExpiry,
	}
}

type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string      `json:"token"`
	User  domain.User `json:"user"`
}

func (s *AuthService) Register(input RegisterInput) (AuthResponse, error) {
	if strings.TrimSpace(input.Email) == "" || len(input.Password) < 6 || strings.TrimSpace(input.Name) == "" {
		return AuthResponse{}, domain.ErrInvalidInput
	}
	role := domain.Role(strings.ToUpper(input.Role))
	if role != domain.RoleOrganizer && role != domain.RoleCustomer {
		return AuthResponse{}, domain.ErrInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return AuthResponse{}, err
	}
	user := domain.User{
		Email:        strings.ToLower(strings.TrimSpace(input.Email)),
		PasswordHash: string(hash),
		Name:         strings.TrimSpace(input.Name),
		Role:         role,
	}
	user, err = s.userRepo.Create(user)
	if err != nil {
		return AuthResponse{}, err
	}
	token, err := s.generateToken(user)
	if err != nil {
		return AuthResponse{}, err
	}
	return AuthResponse{Token: token, User: user}, nil
}

func (s *AuthService) Login(input LoginInput) (AuthResponse, error) {
	if strings.TrimSpace(input.Email) == "" || input.Password == "" {
		return AuthResponse{}, domain.ErrInvalidInput
	}
	user, err := s.userRepo.FindByEmail(strings.ToLower(strings.TrimSpace(input.Email)))
	if err != nil {
		return AuthResponse{}, domain.ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return AuthResponse{}, domain.ErrInvalidCredentials
	}
	token, err := s.generateToken(user)
	if err != nil {
		return AuthResponse{}, err
	}
	return AuthResponse{Token: token, User: user}, nil
}

type Claims struct {
	UserID string      `json:"user_id"`
	Email  string      `json:"email"`
	Role   domain.Role `json:"role"`
	Name   string      `json:"name"`
	jwt.RegisteredClaims
}

func (s *AuthService) generateToken(user domain.User) (string, error) {
	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, domain.ErrUnauthorized
	}
	return claims, nil
}
