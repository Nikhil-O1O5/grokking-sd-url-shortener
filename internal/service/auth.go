package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/Nikhil-O1O5/url-shortener/internal/model"
	"github.com/Nikhil-O1O5/url-shortener/internal/store"
)

const (
	tokenExpiry  = 24 * time.Hour
	bcryptCost   = 12
)

var (
	ErrEmailTaken     = errors.New("email already registered")
	ErrInvalidCreds   = errors.New("invalid email or password")
	ErrInvalidToken   = errors.New("invalid or expired token")
)

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

type AuthService struct {
	userStore *store.UserStore
	jwtSecret []byte
}

func NewAuthService(userStore *store.UserStore, jwtSecret string) *AuthService {
	return &AuthService{
		userStore: userStore,
		jwtSecret: []byte(jwtSecret),
	}
}

type RegisterRequest struct {
	Name     string
	Email    string
	Password string
}

type LoginRequest struct {
	Email    string
	Password string
}

type AuthResponse struct {
	Token string
	User  *model.User
}

func (s *AuthService) Register(req RegisterRequest) (*AuthResponse, error) {
	existing, err := s.userStore.GetByEmail(req.Email)
	if err != nil {
		return nil, fmt.Errorf("register: %w", err)
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hash),
	}

	if err := s.userStore.Create(user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	token, err := s.generateToken(user.UserID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, User: user}, nil
}

func (s *AuthService) Login(req LoginRequest) (*AuthResponse, error) {
	user, err := s.userStore.GetByEmail(req.Email)
	if err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCreds
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCreds
	}

	token, err := s.generateToken(user.UserID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, User: user}, nil
}

func (s *AuthService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (s *AuthService) generateToken(userID string) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signed, nil
}
