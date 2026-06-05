package handlers

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CallbackHandler - главная функция для обработки колбэков
func (h *Handler) CallbackHandler(callback *tgbotapi.CallbackQuery) {
	data := callback.Data

	if h.isPagination(data) {
		h.Pagination(callback)
		return
	}

	if h.isPurchase(data) {
		h.Purchase(callback)
		return
	}

	if strings.HasPrefix(data, "search_") {
		h.handleSearch(callback, data)
		return
	}

	h.handleExact(callback, data)
}

func (h *Handler) isPagination(data string) bool {
	return strings.HasPrefix(data, "prev_") ||
		strings.HasPrefix(data, "next_") ||
		strings.HasPrefix(data, "current_") ||
		strings.HasPrefix(data, "orders") ||
		strings.HasPrefix(data, "users") ||
		strings.HasPrefix(data, "products") ||
		data == "buycategories" ||
		data == "buyproducts" ||
		strings.HasPrefix(data, "categories")
}

func (h *Handler) isPurchase(data string) bool {
	return strings.HasPrefix(data, "category_") ||
		strings.HasPrefix(data, "product_") ||
		strings.HasPrefix(data, "buying_") ||
		data == "confirm" ||
		data == "cancell"
}

func (h *Handler) handleSearch(callback *tgbotapi.CallbackQuery, data string) {
	switch {
	case strings.HasPrefix(data, "search_category"):
		h.SearchCategory(callback)
	case strings.HasPrefix(data, "search_product"):
		h.SearchProduct(callback)
	case strings.HasPrefix(data, "search_user"):
		h.SearchUser(callback)
	}
}

func (h *Handler) handleExact(callback *tgbotapi.CallbackQuery, data string) {
	switch data {
	case "create_order":
		h.CreateOrder(callback)
	case "confirm_order":
		h.ConfirmOrder(callback)
	case "cart":
		h.Cart(callback)
	case "start":
		h.Start(callback)
	case "help":
		h.Help(callback)
	}
}
