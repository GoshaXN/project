package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"project/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) CreateProduct(update tgbotapi.Update) { // создание товара
	// Проверка авторизации и прав
	access := h.AuthenticateCommand(3, update)
	if !access {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Недостаточно прав для совершения команды")
		h.Bot.Send(msg)
		return
	}

	data := strings.Split(update.Message.CommandArguments(), "|")

	if len(data) < 10 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"Некорректный формат. Используйте\n /create_product name|description|flawor|brand|price|quantity|category_id|weight|servings|is_active\n")
		h.Bot.Send(msg)
		return
	}

	product := &models.Product{}

	for i, field := range []*string{&product.Name, &product.Description,
		&product.Flavor, &product.Brand} {
		*field = data[i]
	}

	for i, field := range []interface{}{&product.Price, &product.Quantity, &product.Category_id, &product.Weight,
		&product.Servings, &product.IsActive} {
		switch field := field.(type) {
		case *float64:
			*field, _ = strconv.ParseFloat(data[i+4], 64)
		case *int:
			*field, _ = strconv.Atoi(data[i+4])
		case *bool:
			*field, _ = strconv.ParseBool(data[i+4])
		}
	}

	err := h.ProductRepo.CreateProduct(product)
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Ошибка создания товара: %v", err))
		h.Bot.Send(msg)
		return
	} else {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Создан товар\nID: %d\nНазвание: %s\nОписание: %s\nЦена: %.2f\nКоличество: %d\nКатегория ID: %d\nВес: %v\nВкус: %s\nБренд: %s\nПорций: %d\nАктивен: %v",
				product.ID, product.Name, product.Description, product.Price, product.Quantity,
				product.Category_id, product.Weight, product.Flavor, product.Brand, product.Servings,
				product.IsActive))
		h.Bot.Send(msg)
	}
}

func (h *Handler) Products(update tgbotapi.Update) { // список товаров

	h.ShowPagination(h.Bot, update.Message.Chat.ID, 0, 1,
		h.ProductRepo.CountProducts,
		func(limit, offset int) ([]interface{}, error) {
			orders, err := h.ProductRepo.PaginateProducts(limit, offset)
			if err != nil {
				return nil, err
			}
			return h.ConvertToInterfaceSlice(orders)
		},
		func(data interface{}) string {
			return h.formatProduct(data.(models.Product))
		},
		"товары",
		"products",
		false)
}

func (h *Handler) SearchProduct(input interface{}) { // поиск товаров
	var ChatID int64
	var searchQuery string
	log.Printf("start")
	switch v := input.(type) {
	case tgbotapi.Update:
		ChatID = v.Message.Chat.ID
		searchQuery = strings.TrimSpace(v.Message.CommandArguments()) //TrimSpace удаляет пробелы в начале и конце строки
		if searchQuery == "" {
			h.mu.Lock()
			h.WaitingProduct[ChatID] = true
			h.mu.Unlock()
			msg := tgbotapi.NewMessage(ChatID, "Укажите название товара для поиска")
			h.Bot.Send(msg)
			return
		}
	case *tgbotapi.CallbackQuery:
		log.Printf("continue")
		ChatID = v.Message.Chat.ID
		callbackID := v.ID
		h.mu.Lock()
		h.WaitingProduct[ChatID] = true
		h.mu.Unlock()
		msg := tgbotapi.NewMessage(ChatID, "Укажите название товара для поиска")
		h.Bot.Send(msg)

		callbackConfig := tgbotapi.NewCallback(callbackID, "") //отправка этого конфига нужна для того чтобы кнопка не была нажата
		h.Bot.Send(callbackConfig)
		return
	default:
		return
	}
	log.Printf("end")

	products, err := h.ProductRepo.SearchProduct(searchQuery)
	if err != nil {
		msg := tgbotapi.NewMessage(ChatID, "Ошибка поиска")
		h.Bot.Send(msg)
		return
	}

	if len(products) == 0 {
		msg := tgbotapi.NewMessage(ChatID, "По запросу: "+searchQuery+" товаров не найдено")
		h.Bot.Send(msg)
		return
	}

	response := "Результаты поиска по запросу: " + searchQuery + "\n\n"
	for _, product := range products {
		response += h.formatProduct(product) + "\n"
	}
	msg := tgbotapi.NewMessage(ChatID, response)
	h.Bot.Send(msg)
}

func (h *Handler) UpdateProduct(update tgbotapi.Update) { // изменение товара
	// Проверка авторизации и прав
	token := h.GetTokenFromUpdate(update)
	if token == "" {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Сначала выполните /login")
		h.Bot.Send(msg)
		return
	}

	user, err := h.AuthenticateUser(token, h.UserRepo)
	if err != nil || user.Role != "admin" {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Доступ только для администраторов")
		h.Bot.Send(msg)
		return
	}

	data := strings.Split(update.Message.CommandArguments(), "|")

	if len(data) < 11 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"Некорректный формат. Используйте\n /update_product id|price|quantity|weight|category_id|servings|is_active|name|description|flavor|brand\nНеизменённые поля заполнять символом *")
		h.Bot.Send(msg)

		return
	}

	products, err := h.ProductRepo.SearchProduct(data[0])
	if err != nil || len(products) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Товар не найден")
		h.Bot.Send(msg)

		return
	}
	product := &products[0] //инициализация товара который будет изменться

	for i, field := range []interface{}{&product.Price, &product.Quantity, &product.Weight, &product.Category_id,
		//конструкция для обработки int,float,bool подающегося поля
		&product.Servings, &product.IsActive} {
		switch field := field.(type) {
		case *float64:
			*field, _ = strconv.ParseFloat(data[i+1], 64)
		case *int:
			*field, _ = strconv.Atoi(data[i+1])
		case *bool:
			*field, _ = strconv.ParseBool(data[i+2])
		}
	}

	for i, field := range []*string{&product.Name, &product.Description, //обработка строковых полей вкуса и бренда
		&product.Flavor, &product.Brand} {
		*field = data[i+7]
	}

	err = h.ProductRepo.UpdateProduct(product) //внесённые изменения вносятся в товар
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Ошибка изменения товара: %v", err))
		h.Bot.Send(msg)

		return
	} else {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Изменен товар\nID: %d\nНазвание: %s\nОписание: %s\nЦена: %.2f\nКоличество: %d\nКатегория ID: %d\nВес: %v\nВкус: %s\nБренд: %s\nПорций: %d\nАктивен: %v",
				product.ID, product.Name, product.Description, product.Price, product.Quantity,
				product.Category_id, product.Weight, product.Flavor, product.Brand, product.Servings,
				product.IsActive))
		h.Bot.Send(msg)

	}

}

func (h *Handler) DeleteProduct(update tgbotapi.Update) { // удаление товара
	// Проверка авторизации и прав
	token := h.GetTokenFromUpdate(update)
	if token == "" {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Сначала выполните /login")
		h.Bot.Send(msg)
		return
	}

	user, err := h.AuthenticateUser(token, h.UserRepo)
	if err != nil || user.Role != "admin" {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Доступ только для администраторов")
		h.Bot.Send(msg)
		return
	}

	data := strings.Fields(update.Message.CommandArguments())
	if len(data) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Отправьте команду в формате /delete_product product_id")
		h.Bot.Send(msg)

		return
	}
	productID, err := strconv.Atoi(data[0])
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "ID должно быть числом")
		h.Bot.Send(msg)

		return
	}
	product, err := h.ProductRepo.SearchProduct(fmt.Sprintf("%d", productID))
	if err != nil || len(product) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Товар не найден")
		h.Bot.Send(msg)

		return
	}
	h.mu.Lock()
	h.WaitingConfirm[update.Message.Chat.ID] = func() error { return h.ProductRepo.DeleteProduct(productID) }
	h.mu.Unlock()
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf(
		"Напишите + если хотите удалить товар: %s, ID = %d", product[0].Name, productID))
	h.Bot.Send(msg)

}
