package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"project/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) CreateUser(update tgbotapi.Update) { // создание юзера
	// Проверка авторизации и прав
	access := h.AuthenticateCommand(3, update)
	if !access {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Недостаточно прав для совершения команды")
		h.Bot.Send(msg)
		return
	}

	var password string
	data := strings.Split(update.Message.CommandArguments(), "|")

	if len(data) < 6 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"Некорректный формат. Используйте\n /create_user telegram_id|telegram_username|first_name|phone|email|role|password\n")
		h.Bot.Send(msg)
		return
	}
	TelegramID, err := strconv.ParseInt(data[0], 10, 64)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка: telegram_id должен быть числом")
		h.Bot.Send(msg)
		return
	}

	NewUser := &models.User{
		TelegramID: TelegramID,
		Username:   data[1],
		FirstName:  data[2],
		Phone:      data[3],
		Email:      data[4],
		Role:       data[5],
	}
	if len(data) > 6 {
		password = data[6]
	}

	err = h.userService.CreateUser(NewUser, password)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Ошибка создания пользователя: %v", err))
		h.Bot.Send(msg)
		return
	} else {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Создан пользователь\nID: %d\nTelegramID: %d\nНик: %s\nИмя: %s\nТелефон: %v\nПочта: %s\nРоль: %s\nПароль: %s",
				NewUser.ID, NewUser.TelegramID, NewUser.Username, NewUser.FirstName,
				NewUser.Phone, NewUser.Email, NewUser.Role, password))
		h.Bot.Send(msg)
	}
}

func (h *Handler) Users(update tgbotapi.Update) { // список юзеров
	// Проверка авторизации и прав
	token := h.GetTokenFromUpdate(update)
	if token == "" {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Сначала выполните /login")
		h.Bot.Send(msg)
		return
	}

	h.ShowPagination(h.Bot, update.Message.Chat.ID, 0, 1,
		h.userService.CountUsers,
		func(limit, offset int) ([]interface{}, error) {
			orders, err := h.userService.PaginateUsers(limit, offset)
			if err != nil {
				return nil, err
			}
			return h.ConvertToInterfaceSlice(orders)
		},
		func(data interface{}) string {
			return h.formatUser(data.(models.User))
		},
		"юзеры",
		"users",
		false)
}

func (h *Handler) SearchUser(input interface{}) { //поиск юзера
	var ChatID int64
	var searchQuery string
	switch v := input.(type) {
	case tgbotapi.Update:
		ChatID = v.Message.Chat.ID
		update := v
		searchQuery = strings.TrimSpace(v.Message.CommandArguments()) //TrimSpace удаляет пробелы в начале и конце строки
		// Проверка авторизации и прав
		access := h.AuthenticateCommand(3, update)
		if !access {
			msg := tgbotapi.NewMessage(ChatID, "Недостаточно прав для совершения команды")
			h.Bot.Send(msg)
			return
		}
		if searchQuery == "" {
			h.mu.Lock()
			h.WaitingUser[ChatID] = true
			h.mu.Unlock()
			msg := tgbotapi.NewMessage(ChatID, "Укажите имя юзера для поиска")
			h.Bot.Send(msg)
			return
		}
	case *tgbotapi.CallbackQuery:
		ChatID = v.Message.Chat.ID
		callbackID := v.ID
		// Проверка авторизации и прав
		access := h.AuthenticateCommand(3, callbackID)
		if !access {
			msg := tgbotapi.NewMessage(ChatID, "Недостаточно прав для совершения команды")
			h.Bot.Send(msg)
			return
		}
		h.mu.Lock()
		h.WaitingUser[ChatID] = true
		h.mu.Unlock()
		msg := tgbotapi.NewMessage(ChatID, "Укажите имя юзера для поиска")
		h.Bot.Send(msg)
		callbackConfig := tgbotapi.NewCallback(callbackID, "") //отправка этого конфига нужна для того чтобы кнопка не была нажата
		h.Bot.Send(callbackConfig)
		return
	default:
		return
	}
	users, err := h.userService.SearchUser(searchQuery)
	if err != nil {
		msg := tgbotapi.NewMessage(ChatID, "Ошибка поиска")
		h.Bot.Send(msg)
		return
	} else if len(users) == 0 {
		msg := tgbotapi.NewMessage(ChatID, "По запросу: "+searchQuery+" пользователей не найдено")
		h.Bot.Send(msg)
		return
	} else {
		response := "Результаты поиска по запросу: " + searchQuery + "\n\n"
		for _, user := range users {
			response += h.formatUser(user) + "\n"
		}
		msg := tgbotapi.NewMessage(ChatID, response)
		h.Bot.Send(msg)
	}
}

func (h *Handler) UpdateUser(update tgbotapi.Update) { //изменение юзера
	// Проверка авторизации и прав
	access := h.AuthenticateCommand(3, update)
	if !access {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Отказано в доступе")
		h.Bot.Send(msg)
		return
	}
	data := strings.Split(update.Message.CommandArguments(), "|")

	if len(data) < 7 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"Некорректный формат. Используйте\n /update_user id|telegram_id|telegram_username|first_name|phone|email|role\nНеизменённые поля заполнять символом *")
		h.Bot.Send(msg)
		return
	}

	users, err := h.userService.SearchUser(data[0])
	if err != nil || len(users) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Пользователь не найден")
		h.Bot.Send(msg)
		return
	}
	OldUser := &users[0]

	NewUser := []*string{&OldUser.Username, &OldUser.FirstName, &OldUser.Phone, &OldUser.Email, &OldUser.Role} //строковые поля обрабатываются

	for i := 2; i < len(data) && i-2 < len(NewUser); i++ {
		if data[i] != "*" {
			*NewUser[i-2] = data[i]
		}
	}

	if data[1] != "*" { //обработка числового значения TG_ID
		TelegramID, _ := strconv.ParseInt(data[1], 10, 64)
		OldUser.TelegramID = TelegramID
	}

	err = h.userService.UpdateUser(OldUser)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Ошибка изменения пользователя: %v", err))
		h.Bot.Send(msg)
		return
	} else {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Изменен пользователь\nID: %d\nTelegramID: %d\nНик: %s\nИмя: %s\nТелефон: %v\nПочта: %s\nРоль: %s",
				OldUser.ID, OldUser.TelegramID, OldUser.Username, OldUser.FirstName,
				OldUser.Phone, OldUser.Email, OldUser.Role))
		h.Bot.Send(msg)
	}
}

func (h *Handler) DeleteUser(update tgbotapi.Update) { // удаление пользователя
	// Проверка авторизации и прав
	access := h.AuthenticateCommand(3, update)
	if !access {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Недостаточно прав для совершения команды")
		h.Bot.Send(msg)
		return
	}

	data := strings.Fields(update.Message.CommandArguments())
	if len(data) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Отправьте команду в формате /delete_user user_id")
		h.Bot.Send(msg)
		return
	}
	userID, err := strconv.Atoi(data[0])
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "ID должно быть числом")
		h.Bot.Send(msg)
		return
	}
	users, err := h.userService.SearchUser(fmt.Sprintf("%d", userID))
	if err != nil || len(users) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Пользователь не найден")
		h.Bot.Send(msg)
		return
	}
	h.mu.Lock()
	h.WaitingConfirm[update.Message.Chat.ID] = func() error { return h.userService.DeleteUser(userID) }
	h.mu.Unlock()
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf(
		"Напишите + если хотите удалить пользователя: %s, %s, ID = %d", users[0].FirstName, users[0].Username, userID))
	h.Bot.Send(msg)
}
