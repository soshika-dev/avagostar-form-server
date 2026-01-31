package services

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"avagostar-form-server/internal/config"
	"avagostar-form-server/internal/models"
	"avagostar-form-server/internal/repo"
	"avagostar-form-server/internal/utils"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	users *repo.UserRepo
	cfg   *config.Config
}

type TokenResponse struct {
	AccessToken string      `json:"access_token"`
	TokenType   string      `json:"token_type"`
	ExpiresIn   int64       `json:"expires_in"`
	User        interface{} `json:"user"`
}

type ResetResponse struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func NewAuthService(users *repo.UserRepo, cfg *config.Config) *AuthService {
	return &AuthService{users: users, cfg: cfg}
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*TokenResponse, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return nil, utils.NewAppError(401, "UNAUTHORIZED", "invalid credentials", nil)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, utils.NewAppError(401, "UNAUTHORIZED", "invalid credentials", nil)
	}

	token, expiresIn, err := s.generateToken(user)
	if err != nil {
		return nil, utils.NewAppError(500, "INTERNAL_ERROR", "could not generate token", nil)
	}

	return &TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		User: map[string]string{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	}, nil
}

func (s *AuthService) ForgotPassword(ctx context.Context, username string) (*ResetResponse, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return nil, utils.NewAppError(404, "NOT_FOUND", "user not found", nil)
	}

	code, err := generateCode(6)
	if err != nil {
		return nil, utils.NewAppError(500, "INTERNAL_ERROR", "could not generate code", nil)
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		return nil, utils.NewAppError(500, "INTERNAL_ERROR", "could not secure code", nil)
	}
	codeHash := string(hashBytes)
	expiresAt := time.Now().Add(10 * time.Minute)

	if err := s.users.UpdateResetCode(ctx, user.ID, &codeHash, &expiresAt); err != nil {
		return nil, utils.NewAppError(500, "INTERNAL_ERROR", "could not store reset code", nil)
	}

	resp := &ResetResponse{Message: "reset code sent"}
	if s.cfg.EnableDevResetCodes {
		resp.Code = code
	}
	return resp, nil
}

func (s *AuthService) ResetPassword(ctx context.Context, username, code, newPassword string) error {
	if len(newPassword) < s.cfg.PasswordMinLen {
		return utils.NewAppError(400, "VALIDATION_ERROR", fmt.Sprintf("password must be at least %d characters", s.cfg.PasswordMinLen), nil)
	}

	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return utils.NewAppError(404, "NOT_FOUND", "user not found", nil)
	}

	if user.ResetCodeHash == nil || user.ResetCodeExpiresAt == nil {
		return utils.NewAppError(400, "VALIDATION_ERROR", "reset code not requested", nil)
	}

	if time.Now().After(*user.ResetCodeExpiresAt) {
		return utils.NewAppError(400, "VALIDATION_ERROR", "reset code expired", nil)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.ResetCodeHash), []byte(code)); err != nil {
		return utils.NewAppError(400, "VALIDATION_ERROR", "invalid reset code", nil)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return utils.NewAppError(500, "INTERNAL_ERROR", "could not update password", nil)
	}

	if err := s.users.UpdatePassword(ctx, user.ID, string(passwordHash)); err != nil {
		return utils.NewAppError(500, "INTERNAL_ERROR", "could not update password", nil)
	}

	return nil
}

func (s *AuthService) CreateUser(ctx context.Context, username, password, role string) (*models.User, error) {
	if len(password) < s.cfg.PasswordMinLen {
		return nil, utils.NewAppError(400, "VALIDATION_ERROR", fmt.Sprintf("password must be at least %d characters", s.cfg.PasswordMinLen), nil)
	}

	exists, err := s.users.ExistsByUsername(ctx, username)
	if err != nil {
		return nil, utils.NewAppError(500, "INTERNAL_ERROR", "could not check existing users", nil)
	}
	if exists {
		return nil, utils.NewAppError(409, "CONFLICT", "username already exists", nil)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, utils.NewAppError(500, "INTERNAL_ERROR", "could not secure password", nil)
	}

	user, err := s.users.Create(ctx, username, role, string(passwordHash))
	if err != nil {
		return nil, utils.NewAppError(500, "INTERNAL_ERROR", "could not create user", nil)
	}

	return user, nil
}

func (s *AuthService) generateToken(user *models.User) (string, int64, error) {
	issuedAt := time.Now()
	expiresAt := issuedAt.Add(s.cfg.JWTExpiry)
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", 0, err
	}

	return signed, int64(s.cfg.JWTExpiry.Seconds()), nil
}

func generateCode(length int) (string, error) {
	max := big.NewInt(10)
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		result[i] = byte('0' + n.Int64())
	}
	return string(result), nil
}

func (c Claims) MarshalJSON() ([]byte, error) {
	type Alias Claims
	return json.Marshal(&struct{ Alias }{Alias: Alias(c)})
}
