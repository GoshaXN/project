package handlers

import (
	"fmt"
	"project/internal/models"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) CreateOrder(input interface{}) {
	var ChatID int64
	var user *models.User
	var err error
	switch v := input.(type) {
	case tgbotapi.Update:
		ChatID = v.Message.Chat.ID
		token := h.GetTokenFromUpdate(v)
		if token == "" {
			msg := tgbotapi.NewMessage(ChatID, "Сначала выполните /login")
			h.Bot.Send(msg)
			return
		}
		user, err = h.authService.AuthenticateUser(token)
		if err != nil {
			msg := tgbotapi.NewMessage(ChatID, "Ошибка аутентификации")
			h.Bot.Send(msg)
			return
		}
	case *tgbotapi.CallbackQuery:
		ChatID = v.Message.Chat.ID
		token := h.GetTokenFromCallback(v)
		if token == "" {
			msg := tgbotapi.NewMessage(ChatID, "Сначала выполните /login")
			h.Bot.Send(msg)
			return
		}
		user, err = h.authService.AuthenticateUser(token)
		if err != nil {
			msg := tgbotapi.NewMessage(ChatID, "Ошибка аутентификации")
			h.Bot.Send(msg)
			return
		}
	default:
		return
	}

	order, err := h.orderService.CreateOrder(int64(user.ID))
	if err != nil {
		msg := tgbotapi.NewMessage(ChatID, fmt.Sprintf("Ошибка создания заказа: %v", err))
		h.Bot.Send(msg)
		return
	}

	response := fmt.Sprintf("Заказ создан\nНомер заказа: #%d", order.ID)
	msg1 := tgbotapi.NewMessage(ChatID, response)
	h.Bot.Send(msg1)

	msg2 := tgbotapi.NewMessage(ChatID, "Где будем искать товары?")
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("По категориям", "buycategories")),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("По товарам", "buyproducts")),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Вывести ассортимент", "products")),
	)
	msg2.ReplyMarkup = keyboard
	h.Bot.Send(msg2)
}

func (h *Handler) Orders(update tgbotapi.Update) { // список всех заказов
	// Проверка авторизации и прав
	_, access := h.AuthenticateCommand(3, update)
	if !access {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Недостаточно прав для совершения команды")
		h.Bot.Send(msg)
		return
	}

	h.ShowPagination(h.Bot, update.Message.Chat.ID, 0, 1,
		h.orderService.CountOrders,
		func(limit, offset int) ([]interface{}, error) {
			orders, err := h.orderService.PaginateOrders(limit, offset)
			if err != nil {
				return nil, err
			}
			return h.ConvertToInterfaceSlice(orders)
		},
		func(data interface{}) string {
			return h.formatOrder(data.(models.Order))
		},
		"заказы",
		"allorders",
		false)
}

func (h *Handler) DeleteOrder(update tgbotapi.Update) { // удаление заказа
	// Проверка авторизации и прав
	user, access := h.AuthenticateCommand(3, update)
	if !access {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Недостаточно прав для совершения команды")
		h.Bot.Send(msg)
		return
	}

	data := strings.Fields(update.Message.CommandArguments())
	if len(data) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Отправьте команду в формате /delete_order order_id")
		h.Bot.Send(msg)
		return
	}
	orderID, err := strconv.Atoi(data[0])
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "ID должно быть числом")
		h.Bot.Send(msg)
		return
	}

	order, err := h.orderService.SearchOrder(orderID)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Ошибка удаления заказа: %v", err))
		h.Bot.Send(msg)
		return
	}

	if order == nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Заказ не найден")
		h.Bot.Send(msg)
		return
	}

	h.mu.Lock()
	h.WaitingConfirm[update.Message.Chat.ID] = func() error {
		return h.orderService.DeleteOrder(orderID)
	}
	h.mu.Unlock()

	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
		fmt.Sprintf("Напишите + если хотите удалить заказ с ID = %d\nПользователь: %s (ID = %d)\nСумма: %.2f\nСтатус: %s",
			order.ID, user.FirstName, user.ID, order.Amount, order.Status))
	h.Bot.Send(msg)

}

func (h *Handler) ConfirmOrder(callback *tgbotapi.CallbackQuery) {
	ChatID := callback.Message.Chat.ID

	token := h.GetTokenFromCallback(callback)
	if token == "" {
		msg := tgbotapi.NewMessage(ChatID, "Сначала выполните /login")
		h.Bot.Send(msg)
		return
	}

	user, err := h.authService.AuthenticateUser(token)
	if err != nil {
		msg := tgbotapi.NewMessage(ChatID, "Ошибка аутентификации")
		h.Bot.Send(msg)
		return
	}

	orderID, err := h.orderService.ConfirmOrder(user.ID)
	if err != nil {
		msg := tgbotapi.NewMessage(ChatID, fmt.Sprintf("Ошибка подтверждения заказа: %v", err))
		h.Bot.Send(msg)
		return
	}
	msg := tgbotapi.NewMessage(ChatID, fmt.Sprintf("Заказ #%d успешно сформирован!", orderID))
	h.Bot.Send(msg)
}

func (h *Handler) Cart(input interface{}) {
	var ChatID int64
	var user *models.User
	var err error
	switch v := input.(type) {
	case tgbotapi.Update:
		ChatID = v.Message.Chat.ID
		token := h.GetTokenFromUpdate(v)
		if token == "" {
			msg := tgbotapi.NewMessage(ChatID, "Сначала выполните /login")
			h.Bot.Send(msg)
			return
		}
		user, err = h.authService.AuthenticateUser(token)
		if err != nil {
			msg := tgbotapi.NewMessage(ChatID, "Ошибка аутентификации")
			h.Bot.Send(msg)
			return
		}
	case *tgbotapi.CallbackQuery:
		ChatID = v.Message.Chat.ID
		token := h.GetTokenFromCallback(v)
		if token == "" {
			msg := tgbotapi.NewMessage(ChatID, "Сначала выполните /login")
			h.Bot.Send(msg)
			return
		}
		user, err = h.authService.AuthenticateUser(token)
		if err != nil {
			msg := tgbotapi.NewMessage(ChatID, "Ошибка аутентификации")
			h.Bot.Send(msg)
			return
		}
	default:
		return
	}

	cart, err := h.orderService.DetailCart(int64(user.ID))
	if err != nil {
		msg := tgbotapi.NewMessage(ChatID, fmt.Sprintf("Ошибка загрузки корзины: %v", err))
		h.Bot.Send(msg)
		return
	}

	if cart == nil {
		msg := tgbotapi.NewMessage(ChatID, "Корзина пуста")
		h.Bot.Send(msg)
		return
	}

	response := h.formatCart(&cart.Order, cart.Items)
	if response == "Пустая корзина" {
		msg := tgbotapi.NewMessage(ChatID, "Пустая корзина")
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Добавить товары", "buyproducts"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Вернуться на главную", "start"),
			),
		)
		msg.ReplyMarkup = keyboard
		h.Bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(ChatID, "Ваш заказ:\n\n"+response)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Подтвердить заказ", "confirm_order"),
			tgbotapi.NewInlineKeyboardButtonData("Вернуться к покупкам", "buyproducts"),
		))
	msg.ReplyMarkup = keyboard
	h.Bot.Send(msg)
}
