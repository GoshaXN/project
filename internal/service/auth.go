package service

import (
	"fmt"
	"project/internal/models"
	"project/internal/repo"
	"project/internal/utils"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type AuthService interface {
	Register(TelegramID int64, username, firstName, password string) (*models.User, string, error)
	Login(TelegramID int64, password string) (string, *models.User, error)
	GenerateToken(user *models.User) (string, error)
	VerifyToken(tokenString string) (*Claims, error)
	AuthenticateUser(tokenString string) (*models.User, error)
	UpdatePassword(userID int, newPassword string) error
	GetUserByTelegramID(TelegramID int64) (*models.User, error)
}

type authService struct {
	userRepo      *repo.UserRepo
	jwtSecret     string
	tokenDuration time.Duration
}

func NewAuthService(userRepo *repo.UserRepo, jwtSecret string, tokenDuration time.Duration) AuthService {
	return &authService{
		userRepo:      userRepo,
		jwtSecret:     jwtSecret,
		tokenDuration: tokenDuration,
	}
}

func (s *authService) Register(TelegramID int64, username, firstName, password string) (*models.User, string, error) {
	existing, err := s.userRepo.SearchUserTGID(TelegramID)
	if err != nil && err.Error() != "user not found" {
		return nil, "", fmt.Errorf("ошибка поиска: %v", err)
	}

	if existing != nil { //если пользователь уже существует
		if existing.Password != "" {
			return nil, "", fmt.Errorf("пользователь уже зарегистрирован, выполните /login")
		}
		if err := s.UpdatePassword(int(existing.ID), password); err != nil {
			return nil, "", fmt.Errorf("ошибка установки пароля: %v", err)
		}
		token, err := s.GenerateToken(existing)
		if err != nil {
			return nil, "", err
		}
		return existing, token, nil
	}

	if username == "" {
		username = strconv.FormatInt(TelegramID, 10)
	}
	newUser := &models.User{
		TelegramID: TelegramID,
		Username:   username,
		FirstName:  firstName,
		Phone:      "",
		Email:      "",
		Role:       "user",
	}
	if err := s.userRepo.CreateUser(newUser, password); err != nil {
		return nil, "", fmt.Errorf("ошибка создания: %v", err)
	}
	token, err := s.GenerateToken(newUser)
	if err != nil {
		return nil, "", err
	}
	return newUser, token, nil
}

func (s *authService) Login(TelegramID int64, password string) (string, *models.User, error) {
	user, err := s.userRepo.SearchUserTGID(TelegramID)
	if err != nil {
		return "", user, fmt.Errorf("пользователь не найден")
	}
	if user.Password == "" {
		return "", user, fmt.Errorf("пароль не задан, используйте /register")
	}
	if !utils.CheckPasswordHash(password, user.Password) {
		return "", user, fmt.Errorf("неверный пароль")
	}
	token, err := s.GenerateToken(user)
	if err != nil {
		return "", user, err
	}
	return token, user, nil
}

func (s *authService) GenerateToken(user *models.User) (string, error) {
	expirationTime := time.Now().Add(s.tokenDuration)
	claims := &Claims{
		UserID:   int64(user.ID),
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime), //когда истечёт
			IssuedAt:  jwt.NewNumericDate(time.Now()),     //текущее время
			Subject:   strconv.FormatInt(user.ID, 10),     //UserID
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *authService) VerifyToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func (s *authService) AuthenticateUser(tokenString string) (*models.User, error) {
	claims, err := s.VerifyToken(tokenString)
	if err != nil {
		return nil, err
	}
	users, err := s.userRepo.SearchUser(fmt.Sprintf("%d", claims.UserID))
	if err != nil || len(users) == 0 {
		return nil, fmt.Errorf("user not found")
	}
	return &users[0], nil
}

func (s *authService) UpdatePassword(userID int, newPassword string) error {
	return s.userRepo.UpdatePassword(userID, newPassword)
}

func (s *authService) GetUserByTelegramID(TelegramID int64) (*models.User, error) {
	return s.userRepo.SearchUserTGID(TelegramID)
}
