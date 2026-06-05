package handlers

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// UpdateHandler - главная функция для обработки сообщений
func (h *Handler) UpdateHandler(update tgbotapi.Update) {
	if update.Message.IsCommand() {
		h.ProcessCommand(update)
	} else {
		h.ProcessMessage(update)
	}
}

// ProcessCommand - обработчик команд
func (h *Handler) ProcessCommand(update tgbotapi.Update) {
	if update.Message == nil || !update.Message.IsCommand() {
		return
	}

	command := update.Message.Command() // /cmd - cmd

	switch command {
	case "start":
		h.Start(update)
	case "help":
		h.Help(update)

	case "create_category":
		h.CreateCategory(update)
	case "categories":
		h.Categories(update)
	case "search_category":
		h.SearchCategory(update)
	case "search_by_category":
		h.SearchByCategory(update)
	case "update_category":
		h.UpdateCategory(update)
	case "delete_category":
		h.DeleteCategory(update)

	case "create_user":
		h.CreateUser(update)
	case "users":
		h.Users(update)
	case "search_user":
		h.SearchUser(update)
	case "update_user":
		h.UpdateUser(update)
	case "delete_user":
		h.DeleteUser(update)

	case "create_product":
		h.CreateProduct(update)
	case "products":
		h.Products(update)
	case "search_product":
		h.SearchProduct(update)
	case "update_product":
		h.UpdateProduct(update)
	case "delete_product":
		h.DeleteProduct(update)

	case "create_order":
		h.CreateOrder(update)
	case "orders":
		h.Orders(update)
	case "delete_order":
		h.DeleteOrder(update)
	case "cart":
		h.Cart(update)

	case "register":
		h.Register(update)
	case "login":
		h.Login(update)
	case "logout":
		h.Logout(update)
	case "token":
		h.handleTokenCommand(update)
	default:
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Неизвестная команда\nПовторите попытку")
		h.Bot.Send(msg)
	}
}

func (h *Handler) ProcessMessage(update tgbotapi.Update) { //обработка написанных сообщений не комманд

	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	text := update.Message.Text
	h.mu.RLock()
	deleteFunc, ok := h.WaitingConfirm[chatID]
	h.mu.RUnlock()
	if ok && deleteFunc != nil {
		if text == "+" {
			if err := deleteFunc(); err != nil {
				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка удаления: %v", err))
				h.Bot.Send(msg)
			} else {
				msg := tgbotapi.NewMessage(chatID, "Успешное удаление!")
				h.Bot.Send(msg)
			}
			delete(h.WaitingConfirm, chatID)
		} else {
			msg := tgbotapi.NewMessage(chatID, "Отмена удаления")
			h.Bot.Send(msg)
			delete(h.WaitingConfirm, chatID)
		}
		return
	}
	h.mu.RLock()
	waiting, ok := h.WaitingProduct[chatID]
	h.mu.RUnlock()
	if ok && waiting {
		products, err := h.ProductRepo.SearchProduct(text)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Ошибка поиска")
			h.Bot.Send(msg)
		} else if len(products) == 0 {
			msg := tgbotapi.NewMessage(chatID, "По запросу: "+text+" товаров не найдено")
			h.Bot.Send(msg)
		} else {
			response := "Результаты поиска по запросу: " + text + "\n\n"
			for _, product := range products {
				response += h.formatProduct(product) + "\n"
			}
			msg := tgbotapi.NewMessage(chatID, response)
			h.Bot.Send(msg)
		}
		h.mu.Lock()
		h.WaitingProduct[chatID] = false
		h.mu.Unlock()
		return
	}
	h.mu.RLock()
	waiting, ok = h.WaitingUser[chatID]
	h.mu.RUnlock()
	if ok && waiting {
		users, err := h.UserRepo.SearchUser(text)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Ошибка поиска")
			h.Bot.Send(msg)
		} else if len(users) == 0 {
			msg := tgbotapi.NewMessage(chatID, "По запросу: "+text+" пользователей не найдено")
			h.Bot.Send(msg)
		} else {
			response := "Результаты поиска по запросу: " + text + "\n\n"
			for _, user := range users {
				response += h.formatUser(user) + "\n"
			}
			msg := tgbotapi.NewMessage(chatID, response)
			h.Bot.Send(msg)
		}
		h.mu.Lock()
		h.WaitingUser[chatID] = false
		h.mu.Unlock()
		return
	}
	h.mu.RLock()
	waiting, ok = h.WaitingCategory[chatID]
	h.mu.RUnlock()
	if ok && waiting {
		categories, err := h.CategoryRepo.SearchCategory(text)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Ошибка поиска")
			h.Bot.Send(msg)
		} else if len(categories) == 0 {
			msg := tgbotapi.NewMessage(chatID, "По запросу: "+text+" категорий не найдено")
			h.Bot.Send(msg)
		} else {
			response := "Результаты поиска по запросу: " + text + "\n\n"
			for _, category := range categories {
				response += h.formatCategory(category) + "\n"
			}
			msg := tgbotapi.NewMessage(chatID, response)
			h.Bot.Send(msg)
		}
		h.mu.Lock()
		h.WaitingCategory[chatID] = false
		h.mu.Unlock()
		return
	}
	msg := tgbotapi.NewMessage(chatID, "Неизвестная команда")
	h.Bot.Send(msg)
}
