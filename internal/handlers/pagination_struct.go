package handlers

import (
	"fmt"
	"project/internal/models"
	"strconv"
	"strings"
	_ "time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) Pagination(callback *tgbotapi.CallbackQuery) {

	ChatID := callback.Message.Chat.ID
	MessageID := callback.Message.MessageID
	data := callback.Data

	// Карта обработчиков пагинации
	handlers := map[string]struct {
		AuthRequired   bool
		AdminOnly      bool
		CountFunc      func() (int, error)
		PaginationFunc func(limit, offset int) ([]interface{}, error)
		formatFunc     func(interface{}) string
		title          string
		showKeyboard   bool
	}{
		"products": {
			AuthRequired: false,
			AdminOnly:    false,
			CountFunc:    h.productService.CountProducts,
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				products, err := h.productService.PaginateProducts(limit, offset)
				if err != nil {
					return nil, err
				}
				return h.ConvertToInterfaceSlice(products)
			},
			formatFunc:   func(data interface{}) string { return h.formatProduct(data.(models.Product)) },
			title:        "товары",
			showKeyboard: false,
		},
		"buyproducts": {
			AuthRequired: true,
			AdminOnly:    false,
			CountFunc:    h.productService.CountProducts,
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				products, err := h.productService.PaginateProducts(limit, offset)
				if err != nil {
					return nil, err
				}
				return h.ConvertToInterfaceSlice(products)
			},
			formatFunc:   func(data interface{}) string { return h.formatProduct(data.(models.Product)) },
			title:        "товары",
			showKeyboard: true,
		},
		"users": {
			AuthRequired: true,
			AdminOnly:    true,
			CountFunc:    h.userService.CountUsers,
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				users, err := h.userService.PaginateUsers(limit, offset)
				if err != nil {
					return nil, err
				}
				return h.ConvertToInterfaceSlice(users)
			},
			formatFunc:   func(data interface{}) string { return h.formatUser(data.(models.User)) },
			title:        "пользователи",
			showKeyboard: false,
		},
		"categories": {
			AuthRequired: false,
			AdminOnly:    false,
			CountFunc:    h.categoryService.CountCategories,
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				categories, err := h.categoryService.PaginateCategory(limit, offset)
				if err != nil {
					return nil, err
				}
				return h.ConvertToInterfaceSlice(categories)
			},
			formatFunc:   func(data interface{}) string { return h.formatCategory(data.(models.Category)) },
			title:        "категории",
			showKeyboard: false,
		},
		"orders": {
			AuthRequired: true,
			AdminOnly:    false,
			CountFunc: func() (int, error) {
				UserID := callback.Message.Chat.ID
				users, err := h.userService.SearchUser(fmt.Sprintf("%d", UserID))
				if err != nil || len(users) == 0 {
					return 0, err
				}
				user := users[0]
				return h.orderService.CountUserOrders(int(user.ID))
			},
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				UserID := callback.Message.Chat.ID
				users, err := h.userService.SearchUser(fmt.Sprintf("%d", UserID))
				if err != nil || len(users) == 0 {
					return nil, err
				}
				user := users[0]
				orders, err := h.orderService.PaginateUserOrders(int(user.ID), limit, offset)
				if err != nil {
					return nil, err
				}
				return h.ConvertToInterfaceSlice(orders)
			},
			formatFunc:   func(data interface{}) string { return h.formatOrderPagination(data.(models.Order)) },
			title:        "ваши заказы",
			showKeyboard: false,
		},
		"allorders": {
			AuthRequired: true,
			AdminOnly:    true,
			CountFunc: func() (int, error) {
				return h.orderService.CountOrders()
			},
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				orders, err := h.orderService.PaginateOrders(limit, offset)
				if err != nil {
					return nil, err
				}
				return h.ConvertToInterfaceSlice(orders)
			},
			formatFunc:   func(data interface{}) string { return h.formatOrderPagination(data.(models.Order)) },
			title:        "заказы",
			showKeyboard: false,
		},
		"buycategories": {
			AuthRequired: true,
			AdminOnly:    false,
			CountFunc: func() (int, error) {
				h.mu.RLock()
				categoryID, ok := h.SelectCategory[ChatID]
				h.mu.RUnlock()
				if ok {
					return h.productService.CountProductsByCategory(categoryID)
				}
				return h.categoryService.CountCategories()
			},
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				h.mu.RLock()
				categoryID, ok := h.SelectCategory[ChatID]
				h.mu.RUnlock()
				if ok {
					products, err := h.productService.PaginateProductsByCategory(
						fmt.Sprintf("%d", categoryID), limit, offset)
					if err != nil {
						return nil, err
					}
					return h.ConvertToInterfaceSlice(products)
				}
				categories, err := h.categoryService.PaginateCategory(limit, offset)
				if err != nil {
					return nil, err
				}
				return h.ConvertToInterfaceSlice(categories)
			},
			formatFunc: func(data interface{}) string {
				switch v := data.(type) {
				case models.Category:
					return h.formatCategory(v)
				case models.Product:
					return h.formatProduct(v)
				default:
					return fmt.Sprintf("%v", data)
				}
			},
			title:        "категории и товары",
			showKeyboard: true,
		},
	}

	// Находим обработчик
	for dataType, handler := range handlers {
		if data == dataType ||
			strings.HasPrefix(data, "prev_"+dataType+"_") ||
			strings.HasPrefix(data, "next_"+dataType+"_") ||
			strings.HasPrefix(data, "current_"+dataType+"_") {
			var page int

			if data == dataType {
				page = 1
			} else {
				parts := strings.Split(data, "_")
				currentPage, _ := strconv.Atoi(parts[2])

				switch parts[0] {
				case "current":
					page = currentPage
				case "prev":
					page = currentPage - 1
					if page < 1 {
						page = 1
					}
				case "next":
					page = currentPage + 1
				}
			}

			if handler.AuthRequired {
				if !h.AuthenticateCommand(3, callback) {
					msg := tgbotapi.NewMessage(callback.From.ID, "Недостаточно прав для совершения команды")
					h.Bot.Send(msg)
					return
				}
			}
			h.ShowPagination(h.Bot, ChatID, MessageID, page,
				handler.CountFunc,
				handler.PaginationFunc,
				handler.formatFunc,
				handler.title, dataType, handler.showKeyboard)

			callbackConfig := tgbotapi.NewCallback(callback.ID, "")
			h.Bot.Send(callbackConfig)
			return
		}
	}

	// Если не нашли обработчик
	callbackConfig := tgbotapi.NewCallback(callback.ID, "")
	h.Bot.Send(callbackConfig)
}
