package handlers

import (
	"fmt"
	"log"
	"os"

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
	command := update.Message.Command()

	switch command { //что интерфейс получает - свич
	case "search_category":
		h.SearchCategory(update)
		return
	case "search_user":
		h.SearchUser(update)
		return
	case "search_product":
		h.SearchProduct(update)
		return
	case "create_order":
		h.CreateOrder(update)
		return
	case "cart":
		h.Cart(update)
		return
	}

	// остальные через мапу
	if handler, ok := h.commandHandlers[command]; ok {
		handler(update)
	} else {
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
			h.mu.Lock()
			delete(h.WaitingConfirm, chatID)
			h.mu.Unlock()
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
		products, err := h.productService.SearchProduct(text)
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
		users, err := h.userService.SearchUser(text)
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
		categories, err := h.categoryService.SearchCategory(text)
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
	}

	h.mu.RLock()
	pendingProductID, hasPendingPhoto := h.WaitingProductPhoto[chatID]
	h.mu.RUnlock()
	if hasPendingPhoto && update.Message.Photo != nil && len(update.Message.Photo) > 0 {
		fileID := update.Message.Photo[len(update.Message.Photo)-1].FileID
		err := h.productService.UpdateProductPhoto(pendingProductID, fileID)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Ошибка при обновлении фотографии: %v", err))
			h.Bot.Send(msg)
		}
		h.mu.Lock()
		delete(h.WaitingProductPhoto, chatID)
		h.mu.Unlock()
		msg := tgbotapi.NewMessage(chatID, "Фотография обновлена")
		h.Bot.Send(msg)
		return
	}

	h.mu.RLock()
	waitingDefault, ok := h.WaitingDefaultPhoto[chatID]
	h.mu.RUnlock()
	if ok && waitingDefault && update.Message.Photo != nil && len(update.Message.Photo) > 0 {
		fileID := update.Message.Photo[len(update.Message.Photo)-1].FileID
		h.mu.Lock()
		h.DefaultPhotoFileID = fileID
		delete(h.WaitingDefaultPhoto, chatID)
		h.mu.Unlock()
		err := os.WriteFile("default_photo.txt", []byte(fileID), 0644)
		if err != nil {
			log.Printf("Не удалось сохранить default photo: %v", err)
		}
		msg := tgbotapi.NewMessage(chatID, "заглушка сохранена")
		h.Bot.Send(msg)
		return
	}
	// если прислали не фото
	if ok && waitingDefault {
		msg := tgbotapi.NewMessage(chatID, "некорректное сообщение")
		h.Bot.Send(msg)
		h.mu.Lock()
		delete(h.WaitingDefaultPhoto, chatID)
		h.mu.Unlock()
		return
	}

	msg := tgbotapi.NewMessage(chatID, "Неизвестная команда")
	h.Bot.Send(msg)
}
