package handlers

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CallbackHandler - главная функция для обработки колбэков
func (h *Handler) CallbackHandler(callback *tgbotapi.CallbackQuery) {
	data := callback.Data

	if strings.HasPrefix(data, "prev_") ||
		strings.HasPrefix(data, "next_") ||
		strings.HasPrefix(data, "current_") ||
		strings.HasPrefix(data, "orders") ||
		strings.HasPrefix(data, "users") ||
		strings.HasPrefix(data, "products") ||
		data == "buycategories" ||
		data == "buyproducts" ||
		strings.HasPrefix(data, "categories") {
		h.Pagination(callback)
		return
	}

	if strings.HasPrefix(data, "category_") ||
		strings.HasPrefix(data, "product_") ||
		strings.HasPrefix(data, "buying_") ||
		data == "confirm" || data == "cancell" {
		h.Purchase(callback)
		return
	}
	if strings.HasPrefix(data, "search_category") {
		h.SearchCategory(callback)
	}
	if strings.HasPrefix(data, "search_product") {
		h.SearchProduct(callback)
	}
	if strings.HasPrefix(data, "search_user") {
		h.SearchUser(callback)
	}

	if data == "create_order" {
		h.CreateOrder(callback)
		return
	}
	if data == "confirm_order" {
		h.ConfirmOrder(callback)
		return
	}
	if data == "cart" {
		h.Cart(callback)
		return
	}
	if data == "start" {
		h.Start(callback)
	}
	if data == "help" {
		h.Help(callback)
	}
}
