package handlers

import (
	"fmt"
	"project/internal/models"
	"strconv"
	"strings"

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
			CountFunc:    h.ProductRepo.CountProducts,
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				products, err := h.ProductRepo.PaginateProducts(limit, offset)
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
			CountFunc:    h.ProductRepo.CountProducts,
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				products, err := h.ProductRepo.PaginateProducts(limit, offset)
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
			CountFunc:    h.UserRepo.CountUsers,
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				users, err := h.UserRepo.PaginateUser(limit, offset)
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
			CountFunc:    h.CategoryRepo.CountCategories,
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				categories, err := h.CategoryRepo.PaginateCategory(limit, offset)
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
				users, err := h.UserRepo.SearchUser(fmt.Sprintf("%d", UserID))
				if err != nil || len(users) == 0 {
					return 0, err
				}
				user := users[0]
				return h.OrderRepo.CountUserOrders(int(user.ID))
			},
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				UserID := callback.Message.Chat.ID
				users, err := h.UserRepo.SearchUser(fmt.Sprintf("%d", UserID))
				if err != nil || len(users) == 0 {
					return nil, err
				}
				user := users[0]
				orders, err := h.OrderRepo.PaginateUserOrders(int(user.ID), limit, offset)
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
				return h.OrderRepo.CountOrders()
			},
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				orders, err := h.OrderRepo.PaginateOrders(limit, offset)
				if err != nil {
					return nil, err
				}
				return h.ConvertToInterfaceSlice(orders)
			},
			formatFunc:   func(data interface{}) string { return h.formatOrderPagination(data.(models.Order)) },
			title:        "все заказы",
			showKeyboard: false,
		},
		"buycategories": {
			AuthRequired: true,
			AdminOnly:    false,
			CountFunc: func() (int, error) {
				if categoryID, ok := h.SelectCategory[ChatID]; ok {
					return h.ProductRepo.CountProductsByCategory(fmt.Sprintf("%d", categoryID))
				}
				return h.CategoryRepo.CountCategories()
			},
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				if categoryID, ok := h.SelectCategory[ChatID]; ok {
					products, err := h.ProductRepo.PaginateProductsByCategory(
						fmt.Sprintf("%d", categoryID), limit, offset)
					if err != nil {
						return nil, err
					}
					return h.ConvertToInterfaceSlice(products)
				}
				categories, err := h.CategoryRepo.PaginateCategory(limit, offset)
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
				token := h.GetTokenFromCallback(callback)
				if token == "" {
					msg := tgbotapi.NewMessage(ChatID, "Сначала выполните /login")
					h.Bot.Send(msg)
					continue
				}

				user, err := h.AuthenticateUser(token, h.UserRepo)
				if err != nil {
					msg := tgbotapi.NewMessage(ChatID, "Токен недействителен. Выполните /login")
					h.Bot.Send(msg)
					continue
				}
				if handler.AdminOnly && user.Role != "admin" {
					msg := tgbotapi.NewMessage(ChatID, "Доступ только для администраторов")
					h.Bot.Send(msg)
					continue
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
