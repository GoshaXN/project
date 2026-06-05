package handlers

import (
	"fmt"
	"log"
	"project/internal/models"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// В callback_purchase.go
func (h *Handler) Purchase(callback *tgbotapi.CallbackQuery) {
	data := callback.Data
	ChatID := callback.Message.Chat.ID
	MessageID := callback.Message.MessageID

	if strings.HasPrefix(data, "category_") { //data - то какое значение под собой содержит та или иная кнопка
		ID := strings.TrimPrefix(data, "category_")
		categoryID, err := strconv.Atoi(ID) //конвертация строки в инт (аналог Int в питоне)
		if err != nil {
			log.Printf("Ошибка конвертации: %v", err)
			return
		}
		h.SelectCategory[ChatID] = categoryID

		//action = fmt.Sprintf("select_category_%s", data) //аналог ф строки конвертирующей int->str
		categories, err := h.categoryService.SearchCategory(fmt.Sprintf("%d", categoryID))
		var categoryName string
		if err == nil && len(categories) > 0 {
			categoryName = categories[0].Name
		} else {
			categoryName = fmt.Sprintf("Категория %d", categoryID)
		}
		h.ShowPagination(h.Bot, ChatID, MessageID, 1, //1 = начальная страница
			func() (int, error) {
				return h.productService.CountProductsByCategory(categoryID)
			},
			func(limit, offset int) ([]interface{}, error) {
				products, err := h.productService.PaginateProductsByCategory(fmt.Sprintf("%d", categoryID), limit, offset)
				if err != nil {
					return nil, err
				}
				return h.ConvertToInterfaceSlice(products)
			},
			func(data interface{}) string { return h.formatProduct(data.(models.Product)) },
			fmt.Sprintf("Товары категории: %s", categoryName),
			"buycategories",
			true)

		callbackConfig := tgbotapi.NewCallback(callback.ID, "")
		h.Bot.Send(callbackConfig)
		//log.Printf("user_id: %d, username: %s, action: %s", callback.From.ID, callback.From.FirstName, action)
		return
	}
	if strings.HasPrefix(data, "product_") { //нажатие по кнопке с ID в товарах
		ID := strings.TrimPrefix(data, "product_")
		productID, err := strconv.Atoi(ID)
		if err != nil {
			log.Printf("Ошибка конвертации: %v", err)
			return
		}

		//action = fmt.Sprintf("select_product_%s", data)
		products, err := h.productService.SearchProduct(fmt.Sprintf("%d", productID))
		if err != nil || len(products) == 0 {
			msg := tgbotapi.NewMessage(ChatID, "Товар не найден")
			h.Bot.Send(msg)
			return
		}

		product := products[0]
		h.mu.Lock()
		h.SelectProduct[ChatID] = productID
		h.mu.Unlock()

		if product.Photo != "" {
			photoMsg := tgbotapi.NewPhoto(ChatID, tgbotapi.FileID(product.Photo))
			photoMsg.Caption = fmt.Sprintf("Выбран товар: %s (%s)\nЦена: %.2f руб.\nВыберите количество:", product.Name, product.Flavor, product.Price)
			photoMsg.ReplyMarkup = h.CreateBuyingKeyboard(1)
			h.Bot.Send(photoMsg)
		} else {
			msg := tgbotapi.NewMessage(ChatID, fmt.Sprintf("Выбран товар: %s (%s)\nЦена: %.2f руб.\nВыберите количество:", product.Name, product.Flavor, product.Price))
			msg.ReplyMarkup = h.CreateBuyingKeyboard(1)
			h.Bot.Send(msg)
		}

		callbackConfig := tgbotapi.NewCallback(callback.ID, "")
		h.Bot.Send(callbackConfig)
		return
	}
	if strings.HasPrefix(data, "buying_") { //после выбора товара выбор количества
		parts := strings.Split(data, "_")
		if len(parts) < 3 {
			log.Printf("Неверный формат покупки: %s", data)
			return
		}
		var total_quantity int

		total_quantity, _ = strconv.Atoi(parts[2])

		switch parts[1] {
		case "add":
			total_quantity += 1
			//action = "buying_add_" + fmt.Sprintf("%d", total_quantity)
		case "del":
			total_quantity -= 1
			//action = "buying_del_" + fmt.Sprintf("%d", total_quantity)
		case "quantity":
			//action = "buying_quantity_" + fmt.Sprintf("%d", total_quantity)
		}
		h.mu.Lock()
		h.SelectQuantity[ChatID] = total_quantity
		h.mu.Unlock()
		var response string
		h.mu.RLock()
		productID, ok := h.SelectProduct[ChatID]
		h.mu.RUnlock()
		if ok && productID > 0 {
			products, err := h.productService.SearchProduct(fmt.Sprintf("%d", productID))
			if err == nil && len(products) > 0 {
				product := products[0]
				response = fmt.Sprintf("Выбран товар: %s\nЦена: %.2f руб.\n\nК покупке: %d",
					product.Name, product.Price, total_quantity)
			} else {
				response = fmt.Sprintf("К покупке: %d", total_quantity)
			}
		} else {
			response = fmt.Sprintf("К покупке: %d", total_quantity)
		}

		keyboard := h.CreateBuyingKeyboard(total_quantity)
		msg1 := tgbotapi.NewEditMessageText(ChatID, MessageID, response)
		msg1.ReplyMarkup = &keyboard
		h.Bot.Send(msg1)
		callbackConfig := tgbotapi.NewCallback(callback.ID, "")
		h.Bot.Send(callbackConfig)
		//log.Printf("user_id: %d, username: %s, action: %s, quantity: %d",	callback.From.ID, callback.From.FirstName, action, total_quantity)
		return
	}
	if data == "confirm" || data == "cancell" {
		h.mu.RLock()
		productID, hasProduct := h.SelectProduct[ChatID]
		h.mu.RUnlock()
		if data == "confirm" && hasProduct {
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

			var quantity int = 1
			if stored, ok := h.SelectQuantity[ChatID]; ok {
				quantity = stored
			}

			order, product, err := h.orderService.AddToCart(user.ID, productID, quantity)
			if err != nil {
				msg := tgbotapi.NewMessage(ChatID, fmt.Sprintf("Ошибка: %v", err))
				h.Bot.Send(msg)
				return
			}

			h.mu.Lock()
			delete(h.SelectProduct, ChatID)
			delete(h.SelectCategory, ChatID)
			delete(h.BuyingState, ChatID)
			delete(h.SelectQuantity, ChatID)
			h.mu.Unlock()

			totalSum := float64(quantity) * product.Price
			msgText := fmt.Sprintf("Товар добавлен в корзину\n\nЗаказ: #%d\nТовар: %s (%s)\nЦена товара: %.2f руб.\nКоличество: %d\nСумма за товар: %.2f руб.",
				order.ID, product.Name, product.Flavor, product.Price, quantity, totalSum)
			h.Bot.Send(tgbotapi.NewMessage(ChatID, msgText))

			keyboard := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Да", "buyproducts"),
					tgbotapi.NewInlineKeyboardButtonData("Нет", "cart"),
				),
			)
			msg2 := tgbotapi.NewMessage(ChatID, "Хотите выбрать ещё товары?")
			msg2.ReplyMarkup = keyboard
			h.Bot.Send(msg2)

			editMsg := tgbotapi.NewEditMessageReplyMarkup(ChatID, MessageID, tgbotapi.NewInlineKeyboardMarkup())
			h.Bot.Send(editMsg)

		} else if data == "cancell" { //обработка кнопи отмены
			//action = "cancel_purchase"
			h.mu.Lock()
			delete(h.SelectProduct, ChatID)
			delete(h.SelectCategory, ChatID)
			delete(h.BuyingState, ChatID)
			delete(h.SelectQuantity, ChatID)
			h.mu.Unlock()
			msg := tgbotapi.NewMessage(ChatID, "Отмена")
			h.Bot.Send(msg)

			editMsg := tgbotapi.NewEditMessageReplyMarkup(
				ChatID,
				MessageID,
				tgbotapi.NewInlineKeyboardMarkup(),
			)
			h.Bot.Send(editMsg)
		}

		callbackConfig := tgbotapi.NewCallback(callback.ID, "")
		h.Bot.Send(callbackConfig)
		//log.Printf("user_id: %d, username: %s, action: %s", callback.From.ID, callback.From.FirstName, action)
		return
	}
}
