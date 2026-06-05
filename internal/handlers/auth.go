package handlers

import (
	"fmt"
	"project/internal/models"
	"project/internal/utils"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) Register(update tgbotapi.Update) { //Регистрация
	args := update.Message.CommandArguments()

	if args == "" {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"Используйте команду: /register password|UserID для регистрации\nБез указания будет регистрация текущего аккаунта")
		h.Bot.Send(msg)
		return
	}
	data := strings.Split(args, "|")
	var password string
	var TelegramID int64
	var err error
	if len(data) == 1 { //введён только пароль
		password = data[0]
		TelegramID = update.Message.From.ID
	} else if len(data) == 2 { //введён и пароль и юзер
		password = data[0]
		TelegramID, err = strconv.ParseInt(data[1], 10, 64)
		if err != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				"Ошибка: Telegram ID должен быть числом")
			h.Bot.Send(msg)
			return
		}
	} else { //обработка некорректной команды
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"Некорректный формат. Используйте: /register password|TelegramID")
		h.Bot.Send(msg)
		return
	}

	users, err := h.UserRepo.SearchUserTGID(TelegramID)

	if err != nil && !strings.Contains(err.Error(), "user not found") { //ошибка отсутствия юзера
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Ошибка поиска пользователя: %v", err))
		h.Bot.Send(msg)
		return
	}

	if users != nil { //обработка существующего пользователя
		if users.Password != "" { //вход по паролю
			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				"Пользователь уже зарегистрирован. Войдите: /login password|TelegramID")
			h.Bot.Send(msg)
			msgToUser := tgbotapi.NewMessage(TelegramID,
				fmt.Sprintf("Напоминание пароля для аккаунта ID=%d", users.ID))
			h.Bot.Send(msgToUser)
		} else { //обновление пароля
			err = h.UserRepo.UpdatePassword(int(users.ID), password)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID,
					fmt.Sprintf("Ошибка установки пароля: %v", err))
				h.Bot.Send(msg)
				return
			}
			token, err := h.GenerateToken(users)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID,
					fmt.Sprintf("Ошибка генерации токена: %v", err))
				h.Bot.Send(msg)
				return
			}
			h.mu.Lock()
			h.UserTokens[update.Message.Chat.ID] = token
			h.mu.Unlock()
			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				fmt.Sprintf("Пароль установлен для пользователя %s. Сессия активна 10 минут.",
					users.FirstName))
			h.Bot.Send(msg)

		}
	} else { //создание нового пользователя
		username := update.Message.From.UserName
		if username == "" {
			username = strconv.FormatInt(TelegramID, 10)
		}

		NewUser := &models.User{
			TelegramID: TelegramID,
			Username:   username,
			FirstName:  update.Message.From.FirstName,
			Phone:      "",
			Email:      "",
			Role:       "user",
		}
		err = h.UserRepo.CreateUser(NewUser, password)
		if err != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				fmt.Sprintf("Ошибка создания пользователя: %v", err))
			h.Bot.Send(msg)
			return
		}
		token, err := h.GenerateToken(NewUser)
		if err != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				fmt.Sprintf("Ошибка генерации токена: %v", err))
			h.Bot.Send(msg)
			return
		}
		h.mu.Lock()
		h.UserTokens[update.Message.Chat.ID] = token
		h.mu.Unlock()
		msgToUser := tgbotapi.NewMessage(TelegramID,
			fmt.Sprintf("Ваш пароль для аккаунта ID=%d установлен", NewUser.ID))
		h.Bot.Send(msgToUser)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Пользователь %s успешно зарегистрирован. Сессия активна 10 минут.",
				NewUser.FirstName))
		h.Bot.Send(msg)
	}
}

func (h *Handler) Login(update tgbotapi.Update) { //Вход в аккаунт
	args := update.Message.CommandArguments()
	if args == "" {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"Используйте команду:\n/login password|UserID\nКоманда: /login password будет входить в текущий аккаунт")
		h.Bot.Send(msg)
		return
	}
	data := strings.Split(args, "|")
	var password string
	var TelegramID int64
	var err error
	if len(data) == 1 { // вход в текущий аккаунт
		password = data[0]
		TelegramID = update.Message.From.ID
	} else if len(data) == 2 { // вход в указанный аккаунт
		password = data[0]
		TelegramID, err = strconv.ParseInt(data[1], 10, 64)
		if err != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				"Ошибка: Telegram ID должен быть числом")
			h.Bot.Send(msg)

			return
		}
	} else {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"Некорректный формат. Используйте: /login password|TelegramID")
		h.Bot.Send(msg)
		return
	}
	users, err := h.UserRepo.SearchUserTGID(TelegramID)
	if err != nil {
		if strings.Contains(err.Error(), "user not found") {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				"Пользователь не найден. Пройдите регистрацию: /register password|TelegramID")
			h.Bot.Send(msg)
			return
		} else {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				fmt.Sprintf("Ошибка поиска пользователя: %v", err))
			h.Bot.Send(msg)
			return
		}
	}
	if users.Password == "" {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Отсутствует пароль. Введите\n/register password|TelegramID для установки пароля")
		h.Bot.Send(msg)
		return
	}
	if !utils.CheckPasswordHash(password, users.Password) {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный пароль!")
		h.Bot.Send(msg)
		return
	}
	token, err := h.GenerateToken(users)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Ошибка генерации токена: %v", err))
		h.Bot.Send(msg)
		return
	}
	h.mu.Lock()
	h.UserTokens[update.Message.Chat.ID] = token
	h.mu.Unlock()
	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
		fmt.Sprintf("Здравствуйте, %s!\nВаш статус: %s\nID: %d\nСессия активна 10 минут",
			users.FirstName, users.Role, users.ID))
	h.Bot.Send(msg)
}

func (h *Handler) Logout(update tgbotapi.Update) { //выход из аккаунта
	chatID := update.Message.Chat.ID
	h.mu.Lock()
	delete(h.UserTokens, chatID)
	delete(h.SelectProduct, chatID)
	delete(h.SelectCategory, chatID)
	delete(h.BuyingState, chatID)
	delete(h.SelectQuantity, chatID)
	delete(h.WaitingProduct, chatID)
	delete(h.WaitingUser, chatID)
	delete(h.WaitingCategory, chatID)
	delete(h.WaitingConfirm, chatID)
	delete(h.PaginationState, chatID)
	h.mu.Unlock()
	msg := tgbotapi.NewMessage(chatID, "Успешный выход. Вход: /login")
	h.Bot.Send(msg)
}

func (h *Handler) handleTokenCommand(update tgbotapi.Update) { //обновление токена
	chatID := update.Message.Chat.ID
	token := h.GetTokenFromUpdate(update)
	if token == "" {
		msg := tgbotapi.NewMessage(chatID, "Сначала выполните /login")
		h.Bot.Send(msg)
		return
	}
	user, err := h.AuthenticateUser(token, h.UserRepo)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Токен недействителен. Выполните /login")
		h.Bot.Send(msg)
		return
	}
	NewToken, err := h.GenerateToken(user)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Ошибка генерации токена: %v", err))
		h.Bot.Send(msg)
		return
	}
	h.mu.Lock()
	h.UserTokens[update.Message.Chat.ID] = NewToken
	h.mu.Unlock()
	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
		fmt.Sprintf("Ваш новый токен: %s\nДействует 10 минут", NewToken))
	h.Bot.Send(msg)
	msg.ParseMode = "Markdown"
}
