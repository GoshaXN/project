package handlers

import (
	"project/internal/models"
	"project/internal/repo"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) AuthMiddleware(handler func(bot *tgbotapi.BotAPI, update tgbotapi.Update, // аутентефикация юзера и если с токеном - выполняет переданную функцию
	user *models.User, userRepo *repo.UserRepo)) func(bot *tgbotapi.BotAPI, update tgbotapi.Update, userRepo *repo.UserRepo) {

	return func(bot *tgbotapi.BotAPI, update tgbotapi.Update, userRepo *repo.UserRepo) { //аутентефикация в боте
		token := h.GetTokenFromUpdate(update)
		if token == "" {
			msg := tgbotapi.NewMessage(h.GetChatID(update), "Используйте команду /login")
			h.Bot.Send(msg)
			return
		}
		user, err := h.authService.AuthenticateUser(token) //получение юзера из бд по токену
		if err != nil {
			msg := tgbotapi.NewMessage(h.GetChatID(update), "Токен недействителен /login")
			h.Bot.Send(msg)
			return
		}
		h.mu.Lock()
		h.UserTokens[h.GetChatID(update)] = token
		h.mu.Unlock()
		handler(bot, update, user, userRepo) //вызов обработчика
	}
}

func (h *Handler) GetTokenFromUpdate(update tgbotapi.Update) string { //извлечение токена из сообщения
	ChatID := h.GetChatID(update)
	if ChatID > 0 {
		h.mu.RLock()
		token, ok := h.UserTokens[ChatID]
		h.mu.RUnlock()
		if ok {
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
		h.mu.RLock()
		token, ok := h.UserTokens[update.Message.Chat.ID]
		h.mu.RUnlock()
		if ok {
			return token
		}
	}
	if update.CallbackQuery != nil {
		h.mu.RLock()
		token, ok := h.UserTokens[update.CallbackQuery.Message.Chat.ID]
		h.mu.RUnlock()
		if ok {
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

func (h *Handler) AuthenticateCommand(sec_level /*1 - all, 2 - auth, 3 - admin */ int, input interface{}) (*models.User, bool) {

	var ChatID int64
	var update tgbotapi.Update

	switch v := input.(type) {
	case tgbotapi.Update:
		ChatID = h.GetChatID(v)
		update = v
	case *tgbotapi.CallbackQuery:
		ChatID = v.Message.Chat.ID
		update = tgbotapi.Update{CallbackQuery: v}
	}

	if sec_level < 2 {
		return nil, true
	}

	token := h.GetTokenFromUpdate(update)
	if token == "" {
		msg := tgbotapi.NewMessage(ChatID, "Сначала выполните логин /login")
		h.Bot.Send(msg)
		return nil, false
	}

	user, err := h.authService.AuthenticateUser(token)
	if err != nil {
		msg := tgbotapi.NewMessage(ChatID, "Токен недействителен /login")
		h.Bot.Send(msg)
		return nil, false
	}

	if sec_level == 3 && user.Role != "admin" {
		msg := tgbotapi.NewMessage(ChatID, "У вас нет прав для выполнения команды")
		h.Bot.Send(msg)
		return nil, false
	}
	return user, true
}
