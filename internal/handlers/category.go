package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"project/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) CreateCategory(update tgbotapi.Update) { //Создание категории
	access := h.AuthenticateCommand(3, update)
	if !access {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Недостаточно прав для совершения команды")
		h.Bot.Send(msg)
		return
	}

	// остальная логика из telegram.go
	data := strings.Split(update.Message.CommandArguments(), "|")

	if len(data) < 3 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"Некорректный формат. Используйте\n /create_category name|description|is_active\n")
		h.Bot.Send(msg)
		return
	}

	isActive, err := strconv.ParseBool(data[2])
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("is_active должно быть true/false: %v", err))
		h.Bot.Send(msg)
		return
	}

	category := &models.Category{
		Name:        data[0],
		Description: data[1],
		IsActive:    isActive,
	}

	err = h.CategoryRepo.CreateCategory(category)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Ошибка создания категории: %v", err))
		h.Bot.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID,
		fmt.Sprintf("Создана категория %s\nID: %d\nОписание: %s",
			category.Name, category.ID, category.Description))
	h.Bot.Send(msg)
}

func (h *Handler) Categories(update tgbotapi.Update) { // список категорий

	h.ShowPagination(h.Bot, update.Message.Chat.ID, 0, 1,
		h.CategoryRepo.CountCategories,
		func(limit, offset int) ([]interface{}, error) {
			orders, err := h.CategoryRepo.PaginateCategory(limit, offset)
			if err != nil {
				return nil, err
			}
			return h.ConvertToInterfaceSlice(orders)
		},
		func(data interface{}) string {
			return h.formatCategory(data.(models.Category))
		},
		"категории",
		"categories",
		false)

}

func (h *Handler) SearchCategory(input interface{}) { // поиск категории

	var ChatID int64
	var searchQuery string
	switch v := input.(type) {
	case tgbotapi.Update:
		ChatID = v.Message.Chat.ID
		searchQuery = strings.TrimSpace(v.Message.CommandArguments()) //TrimSpace - удаляет пробелы в начале и в конце строки
		if searchQuery == "" {
			h.mu.Lock()
			h.WaitingCategory[ChatID] = true
			h.mu.Unlock()
			msg := tgbotapi.NewMessage(ChatID, "Укажите название категории для поиска")
			h.Bot.Send(msg)
			return
		}
	case *tgbotapi.CallbackQuery:
		ChatID = v.Message.Chat.ID
		callbackID := v.ID
		h.mu.Lock()
		h.WaitingCategory[ChatID] = true
		h.mu.Unlock()
		msg := tgbotapi.NewMessage(ChatID, "Укажите название категории для поиска")
		h.Bot.Send(msg)

		callbackConfig := tgbotapi.NewCallback(callbackID, "") //отправка этого конфига нужна для того чтобы кнопка не была нажата
		h.Bot.Send(callbackConfig)
		return
	default:
		return
	}

	categories, err := h.CategoryRepo.SearchCategory(searchQuery)
	if err != nil {
		msg := tgbotapi.NewMessage(ChatID, "Ошибка поиска")
		h.Bot.Send(msg)
		return
	}

	if len(categories) == 0 {
		msg := tgbotapi.NewMessage(ChatID, "По запросу: "+searchQuery+" категорий не найдено")
		h.Bot.Send(msg)
		return
	}

	response := "Результаты поиска по запросу: " + searchQuery + "\n\n"
	for _, category := range categories {
		response += h.formatCategory(category) + "\n"
	}

	msg := tgbotapi.NewMessage(ChatID, response)
	h.Bot.Send(msg)
}

func (h *Handler) SearchByCategory(update tgbotapi.Update) { // поиск товаров по категории

	searchQuery := update.Message.CommandArguments()

	if searchQuery == "" {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Укажите название категории для поиска")
		h.mu.Lock()
		h.WaitingCategory[update.Message.Chat.ID] = true
		h.mu.Unlock()
		h.Bot.Send(msg)
		return
	}

	products, err := h.ProductRepo.ProductsByCategory(searchQuery)
	if err != nil {
		fmt.Printf("error: %v", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка поиска")
		h.Bot.Send(msg)
		return
	} else if len(products) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "По запросу: "+searchQuery+" категорий не найдено")
		h.Bot.Send(msg)
		return
	} else {
		response := "Результаты поиска по запросу: " + searchQuery + "\n\n"
		for _, product := range products {
			response += h.formatProduct(product) + "\n"
		}
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, response)
		h.Bot.Send(msg)

	}
}

func (h *Handler) UpdateCategory(update tgbotapi.Update) { // обновление категории

	access := h.AuthenticateCommand(3, update)
	if !access {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Недостаточно прав для совершения команды")
		h.Bot.Send(msg)
		return
	}

	data := strings.Split(update.Message.CommandArguments(), "|")

	if len(data) < 4 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"Некорректный формат. Используйте\n /update_category id|name|description|is_active\nНеизменённые поля заполнять символом *")
		h.Bot.Send(msg)
		return
	}

	categories, err := h.CategoryRepo.SearchCategory(data[0])
	if err != nil || len(categories) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Категория не найдена")
		h.Bot.Send(msg)
		return
	}
	category := &categories[0] //категория для изменения

	for i, field := range []*string{&category.Name, &category.Description} { //строковые поля изменяются
		if data[i+1] != "*" {
			*field = data[i+1]
		}
	}

	if data[3] != "*" {
		IsActive, _ := strconv.ParseBool(data[3]) //бул значение
		category.IsActive = IsActive
	}

	err = h.CategoryRepo.UpdateCategory(category)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Ошибка изменения категории: %v", err))
		h.Bot.Send(msg)
		return
	} else {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Изменена категория\nID: %d\nИмя: %s\nОписание: %s\nАктивна: %v",
				category.ID, category.Name, category.Description, category.IsActive))
		h.Bot.Send(msg)
	}
}

func (h *Handler) DeleteCategory(update tgbotapi.Update) { // удаление категории

	access := h.AuthenticateCommand(3, update)
	if !access {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Недостаточно прав для совершения команды")
		h.Bot.Send(msg)
		return
	}

	data := strings.Fields(update.Message.CommandArguments())
	if len(data) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Отправьте команду в формате /delete_category category_id")
		h.Bot.Send(msg)
		return
	}
	categoryID, err := strconv.Atoi(data[0])
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "ID должно быть числом")
		h.Bot.Send(msg)
		return
	}
	categories, err := h.CategoryRepo.SearchCategory(fmt.Sprintf("%d", categoryID))
	if err != nil || len(categories) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Категория не найдена")
		h.Bot.Send(msg)
		return

	}
	h.mu.Lock()
	h.WaitingConfirm[update.Message.Chat.ID] = func() error { return h.CategoryRepo.DeleteCategory(categoryID) }
	h.mu.Unlock()
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf(
		"Напишите + если хотите удалить категорию: %s, %s, ID = %d", categories[0].Name, categories[0].Description, categoryID))
	h.Bot.Send(msg)
}
