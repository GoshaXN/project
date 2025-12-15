package handlers

import (
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) Start(input interface{}) { // кнопка старт
	var ChatID int64
	var msg tgbotapi.MessageConfig
	switch v := input.(type) {
	case tgbotapi.Update:
		ChatID = v.Message.Chat.ID
		msg = tgbotapi.NewMessage(ChatID,
			fmt.Sprintf("%s, добро пожаловать в магазин спортивного питания!\nВаш TG_ID: %s\n\nВыберите нужное действие:",
				v.Message.From.FirstName, v.Message.From.UserName))
	case *tgbotapi.CallbackQuery:
		ChatID = v.Message.Chat.ID
		msg = tgbotapi.NewMessage(ChatID,
			fmt.Sprintf("%s, добро пожаловать в магазин спортивного питания!\nВаш TG_ID: %s\n\nВыберите нужное действие:",
				v.Message.From.FirstName, v.Message.From.UserName))

	default:
		return
	}

	// Очищаем состояния
	delete(h.SelectProduct, ChatID)
	delete(h.SelectCategory, ChatID)
	delete(h.BuyingState, ChatID)
	delete(h.SelectQuantity, ChatID)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Все товары", "products"),
			tgbotapi.NewInlineKeyboardButtonData("Поиск товаров", "search_product"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Все пользователи", "users"),
			tgbotapi.NewInlineKeyboardButtonData("Поиск пользователя", "search_user"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Корзина", "cart"),
			tgbotapi.NewInlineKeyboardButtonData("Категории", "categories"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Все категории", "categories"),
			tgbotapi.NewInlineKeyboardButtonData("Поиск категорий", "search_category"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Помощь по командам", "help"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Заказы", "orders"),
			tgbotapi.NewInlineKeyboardButtonData("Создать заказ", "create_order"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Выбрать товар для покупки", "buyproducts"),
		),
	)
	msg.ReplyMarkup = keyboard
	h.Bot.Send(msg)
}

func (h *Handler) Help(input interface{}) { // кнопка хелп
	var ChatID int64
	var msg tgbotapi.MessageConfig
	switch v := input.(type) {
	case *tgbotapi.Message:
		ChatID = v.Chat.ID
		msg = tgbotapi.NewMessage(ChatID,
			"/start - начало\n/products - все товары\n/categories - все категории\n/search [product/user] [текст] - поиск товаров/пользователей\n/help - помощь\n/users - список пользователей")

	case *tgbotapi.CallbackQuery:
		ChatID = v.Message.Chat.ID
		msg = tgbotapi.NewMessage(ChatID,
			"/start - начало\n/products - все товары\n/categories - все категории\n/search [product/user] [текст] - поиск товаров/пользователей\n/help - помощь\n/users - список пользователей")
	default:
		return
	}
	h.Bot.Send(msg)
}

func (h *Handler) LogUpdate(update tgbotapi.Update) { //функция логирования
	if update.Message != nil {
		fmt.Printf("%s UserName: %s (%d) Message: %s\n",
			time.Now().Format("02.01.2006 15:04:05"), update.Message.From.UserName, update.Message.From.ID, update.Message.Text)
	}
	if update.CallbackQuery != nil {
		fmt.Printf("%s UserName: %s (%d) Callback: %s\n",
			time.Now().Format("02.01.2006 15:04:05"), update.CallbackQuery.From.UserName, update.CallbackQuery.From.ID, update.CallbackQuery.Data)
	}
}
