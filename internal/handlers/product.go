package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"project/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) CreateProduct(update tgbotapi.Update) { // создание товара
	// Проверка авторизации и прав
	_, access := h.AuthenticateCommand(3, update)
	if !access {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Недостаточно прав для совершения команды")
		h.Bot.Send(msg)
		return
	}

	args := strings.Split(update.Message.CommandArguments(), "|")

	if len(args) < 10 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			"Некорректный формат. Используйте\n /create_product name|description|flawor|brand|price|quantity|category_id|weight|servings|is_active\n")
		h.Bot.Send(msg)
		return
	}

	product := &models.Product{}

	for i, field := range []*string{&product.Name, &product.Description,
		&product.Flavor, &product.Brand} {
		*field = args[i]
	}

	for i, field := range []interface{}{&product.Price, &product.Quantity, &product.Category_id, &product.Weight,
		&product.Servings, &product.IsActive} {
		switch field := field.(type) {
		case *float64:
			*field, _ = strconv.ParseFloat(args[i+4], 64)
		case *int:
			*field, _ = strconv.Atoi(args[i+4])
		case *bool:
			*field, _ = strconv.ParseBool(args[i+4])
		}
	}

	if err := h.productService.CreateProduct(product); err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Ошибка создания товара: %v", err))
		h.Bot.Send(msg)
		return
	} else {
		h.mu.Lock()
		h.WaitingProductPhoto[update.Message.Chat.ID] = product.ID
		h.mu.Unlock()
		msg := tgbotapi.NewMessage(update.Message.Chat.ID,
			fmt.Sprintf("Создан товар\nID: %d\nНазвание: %s\nОписание: %s\nЦена: %.2f\nКоличество: %d\nКатегория ID: %d\nВес: %v\nВкус: %s\nБренд: %s\nПорций: %d\nАктивен: %v\nОтпрваьте фото для товара или /skip_photo для отмены действия",
				product.ID, product.Name, product.Description, product.Price, product.Quantity,
				product.Category_id, product.Weight, product.Flavor, product.Brand, product.Servings,
				product.IsActive))
		h.Bot.Send(msg)
	}
}

func (h *Handler) Products(update tgbotapi.Update) { // список товаров с фото
	h.ShowPaginationWithPhotos(h.Bot, update.Message.Chat.ID, 0, 1,
		h.productService.CountProducts,
		func(limit, offset int) ([]interface{}, error) {
			products, err := h.productService.GetPaginatedProducts(limit, offset)
			if err != nil {
				return nil, err
			}
			return h.ConvertToInterfaceSlice(products)
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

	products, err := h.productService.SearchProduct(searchQuery)
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

	header := tgbotapi.NewMessage(ChatID, fmt.Sprintf("🔍 Результаты поиска по запросу: %s\nНайдено: %d товаров\n\n", searchQuery, len(products)))
	h.Bot.Send(header)

	for _, product := range products {
		fileID := product.Photo
		if fileID == "" {
			fileID = h.DefaultPhotoFileID
		}
		text := h.formatProduct(product)
		if fileID != "" {
			photoMsg := tgbotapi.NewPhoto(ChatID, tgbotapi.FileID(fileID))
			photoMsg.Caption = text
			h.Bot.Send(photoMsg)
		} else {
			h.Bot.Send(tgbotapi.NewMessage(ChatID, text))
		}
	}
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
	products, err := h.productService.SearchByCategory(searchQuery)
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

func (h *Handler) UpdateProduct(update tgbotapi.Update) { // изменение товара
	// проверка авторизации и прав
	_, access := h.AuthenticateCommand(3, update)
	if !access {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Недостаточно прав для совершения команды")
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

	products, err := h.productService.SearchProduct(data[0])
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

	err = h.productService.UpdateProduct(product) //внесённые изменения вносятся в товар
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

func (h *Handler) UpdateProductPhoto(update tgbotapi.Update) {
	_, access := h.AuthenticateCommand(3, update)
	if !access {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Недостаточно прав для совершения команды")
		h.Bot.Send(msg)
		return
	}
	args := strings.Fields(update.Message.CommandArguments())
	if len(args) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "используйте команду: /update_photo productID")
		h.Bot.Send(msg)
		return
	}
	productID, err := strconv.Atoi(args[0])
	if err != nil {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "ID должно быть числом")
		h.Bot.Send(msg)
	}
	h.mu.Lock()
	h.WaitingProductPhoto[update.Message.Chat.ID] = productID
	h.mu.Unlock()
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Отправьте новое фото для товара")
	h.Bot.Send(msg)
}

func (h *Handler) DeleteProduct(update tgbotapi.Update) { // удаление товара
	// Проверка авторизации и прав
	_, access := h.AuthenticateCommand(3, update)
	if !access {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Недостаточно прав для совершения команды")
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
	product, err := h.productService.SearchProduct(fmt.Sprintf("%d", productID))
	if err != nil || len(product) == 0 {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Товар не найден")
		h.Bot.Send(msg)
		return
	}
	h.mu.Lock()
	h.WaitingConfirm[update.Message.Chat.ID] = func() error { return h.productService.DeleteProduct(productID) }
	h.mu.Unlock()
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf(
		"Напишите + если хотите удалить товар: %s, ID = %d", product[0].Name, productID))
	h.Bot.Send(msg)

}

func (h *Handler) SkipPhoto(update tgbotapi.Update) {
	h.mu.Lock()
	delete(h.WaitingProductPhoto, update.Message.Chat.ID)
	h.mu.Unlock()
}
