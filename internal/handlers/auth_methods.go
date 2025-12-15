package handlers

import (
	"fmt"
	"project/internal/models"
	"project/internal/repo"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/golang-jwt/jwt/v5"
)

func (h *Handler) GenerateToken(user *models.User) (string, error) {
	expirationTime := time.Now().Add(jwtConfig.TokenDuration)

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
	return token.SignedString([]byte(jwtConfig.SecretKey))
}

func (h *Handler) VerifyToken(tokenString string) (*Claims, error) { //верификация токена: преобразовывает в данные и проверяет на валидность
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtConfig.SecretKey), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func (h *Handler) AuthenticateUser(tokenString string, userRepo *repo.UserRepo) (*models.User, error) { //аутентефикация по токену
	claims, err := h.VerifyToken(tokenString) //получение юзера из бд по токену
	if err != nil {
		return nil, err
	}
	users, err := userRepo.SearchUser(fmt.Sprintf("%d", claims.UserID))
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("user not found")
	}
	return &users[0], nil
}

func (h *Handler) AuthMiddleware(handler func(bot *tgbotapi.BotAPI, update tgbotapi.Update, // аутентефикация юзера и если с токеном - выполняет переданную функцию
	user *models.User, userRepo *repo.UserRepo)) func(bot *tgbotapi.BotAPI, update tgbotapi.Update, userRepo *repo.UserRepo) {

	return func(bot *tgbotapi.BotAPI, update tgbotapi.Update, userRepo *repo.UserRepo) { //аутентефикация в боте
		token := h.GetTokenFromUpdate(update)
		if token == "" {
			msg := tgbotapi.NewMessage(h.GetChatID(update), "Используйте команду /login")
			h.Bot.Send(msg)
			return
		}
		user, err := h.AuthenticateUser(token, userRepo) //получение юзера из бд по токену
		if err != nil {
			msg := tgbotapi.NewMessage(h.GetChatID(update), "Токен недействителен /login")
			h.Bot.Send(msg)
			return
		}
		h.UserTokens[h.GetChatID(update)] = token

		handler(bot, update, user, userRepo) //вызов обработчика
	}
}
func (h *Handler) GetTokenFromUpdate(update tgbotapi.Update) string { //извлечение токена из сообщения
	ChatID := h.GetChatID(update)
	if ChatID > 0 {
		if token, ok := h.UserTokens[ChatID]; ok {
			return token
		}
	}

	if update.Message != nil && update.Message.CommandArguments() != "" {
		args := update.Message.CommandArguments()
		if strings.HasPrefix(args, "token:") {
			return strings.TrimPrefix(args, "token:")
		}
	}
	if update.Message != nil {
		if token, ok := h.UserTokens[update.Message.Chat.ID]; ok {
			return token
		}
	}
	if update.CallbackQuery != nil {
		if token, ok := h.UserTokens[update.CallbackQuery.Message.Chat.ID]; ok {
			return token
		}
	}
	return ""
}

func (h *Handler) GetTokenFromCallback(callback *tgbotapi.CallbackQuery) string { //извлечение токена из коллбека через фейк апдейт
	update := tgbotapi.Update{CallbackQuery: callback} //  создается фейковый апдейт который содержит коллбек
	return h.GetTokenFromUpdate(update)
}

func (h *Handler) GetChatID(update tgbotapi.Update) int64 {
	if update.Message != nil {
		return update.Message.Chat.ID
	}
	if update.CallbackQuery != nil {
		return update.CallbackQuery.Message.Chat.ID
	}
	return 0
}

func (h *Handler) CheckPermissions(userRepo *repo.UserRepo, token string, status int) error {
	user, err := h.AuthenticateUser(token, userRepo)
	if status == 1 && (user.Role == "user" || user.Role == "admin") {
		return nil
	} else if status == 2 && (user.Role == "admin") {
		return nil
	} else if status == 0 {
		return nil
	} else {
		return err
	}
}
