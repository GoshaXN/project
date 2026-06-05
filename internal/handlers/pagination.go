package handlers

import (
	"fmt"
	"project/internal/models"
	"reflect"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) ShowPagination(bot *tgbotapi.BotAPI, ChatID int64, MessageID int, Page int, //универсальная функция показа данных на страницу с пагинацией
	CountData func() (int, error), //подсчёт страниц. Передается к примеру productRepo.CountProduct
	PaginationFunc func(limit, offset int) ([]interface{}, error), //возрат данных одной страницы
	formatFunc func(interface{}) string, //форматирование(вывод) данных
	title string, paginationType string, showKeyboard bool) {
	DataOnPage := 5
	offset := (Page - 1) * DataOnPage
	count, err := CountData()
	if err != nil {
		fmt.Printf("error: %v", err)
		msg := tgbotapi.NewMessage(ChatID, "Ошибка подсчёта данных")
		bot.Send(msg)
		return
	}

	data, err := PaginationFunc(DataOnPage, offset)
	if err != nil {
		msg := tgbotapi.NewMessage(ChatID, "Ошибка загрузки данных")
		bot.Send(msg)
		return
	}

	if len(data) == 0 {
		msg := tgbotapi.NewEditMessageText(ChatID, MessageID, "Нет данных!")
		bot.Send(msg)
		h.mu.Lock()
		delete(h.SelectCategory, ChatID)
		h.mu.Unlock()
		return
	}

	pages := (count + DataOnPage - 1) / DataOnPage
	h.mu.Lock()
	h.PaginationState[ChatID] = PaginationState{
		CurrentPage: Page,
		Pages:       pages,
		Type:        paginationType,
		Count:       count,
	}
	h.mu.Unlock()

	response := fmt.Sprintf("Все %s\n\n", title)
	for _, item := range data {
		response += formatFunc(item) + "\n"
	}

	keyboard := h.CreatePaginationKeyboard(Page, pages, paginationType, data, showKeyboard)

	if MessageID != 0 {
		msg := tgbotapi.NewEditMessageText(ChatID, MessageID, response)
		msg.ReplyMarkup = &keyboard
		bot.Send(msg)
	} else {
		msg := tgbotapi.NewMessage(ChatID, response)
		msg.ReplyMarkup = keyboard
		bot.Send(msg)
	}
}

func (h *Handler) ShowPaginationWithPhotos(
	bot *tgbotapi.BotAPI, chatID int64, oldTextMsgID int, page int, countData func() (int, error), paginationFunc func(limit, offset int) ([]interface{}, error),
	formatFunc func(interface{}) string,
	title string, paginationType string, showKeyboard bool,
) {

	const perPage = 5
	offset := (page - 1) * perPage

	total, err := countData()
	if err != nil || total == 0 {
		msg := tgbotapi.NewMessage(chatID, "нет данных")
		bot.Send(msg)
		return
	}

	data, err := paginationFunc(perPage, offset)
	if err != nil || len(data) == 0 {
		msg := tgbotapi.NewMessage(chatID, "ошибка загрузки")
		bot.Send(msg)
		return
	}

	pages := (total + perPage - 1) / perPage
	response := fmt.Sprintf("%s (страница %d/%d)\n\n", title, page, pages)
	for _, item := range data {
		response += formatFunc(item)
	}

	var textMsgID int
	if oldTextMsgID == 0 {
		sendMsg := tgbotapi.NewMessage(chatID, response)
		sendMsg.ReplyMarkup = h.CreatePaginationKeyboard(page, pages, paginationType, data, showKeyboard)
		sent, _ := bot.Send(sendMsg)
		textMsgID = sent.MessageID
	} else {
		editMsg := tgbotapi.NewEditMessageText(chatID, oldTextMsgID, response)
		editMsg.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: h.CreatePaginationKeyboard(page, pages, paginationType, data, showKeyboard).InlineKeyboard,
		}

		bot.Send(editMsg)
		textMsgID = oldTextMsgID
	}

	h.mu.Lock()
	if state, ok := h.PhotoPaginationState[chatID]; ok {
		for _, msgID := range state.PhotoMessage {
			del := tgbotapi.NewDeleteMessage(chatID, msgID)
			bot.Send(del)
		}
	}

	var media []interface{}
	for _, item := range data {
		product := item.(models.Product)
		fileID := product.Photo
		if fileID == "" && h.DefaultPhotoFileID != "" {
			fileID = h.DefaultPhotoFileID
		}
		if fileID != "" {
			media = append(media, tgbotapi.NewInputMediaPhoto(tgbotapi.FileID(fileID)))
		}
	}

	var photoIDs []int
	if len(media) > 0 {
		mediaGroup := tgbotapi.NewMediaGroup(chatID, media)
		sentPhotos, _ := bot.SendMediaGroup(mediaGroup)
		for _, p := range sentPhotos {
			photoIDs = append(photoIDs, p.MessageID)
		}
	}

	h.PhotoPaginationState[chatID] = &PhotoPaginationState{
		TextMessageID: textMsgID,
		PhotoMessage:  photoIDs,
	}
	h.mu.Unlock()
}
func (h *Handler) CreatePaginationKeyboard(CurrentPage, Pages int, Type string, data []interface{}, showKeyboard bool) tgbotapi.InlineKeyboardMarkup { //создание клавиатуры перелистывания
	var rows [][]tgbotapi.InlineKeyboardButton

	var nav []tgbotapi.InlineKeyboardButton

	if CurrentPage > 1 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("← Назад",
			fmt.Sprintf("prev_%s_%d", Type, CurrentPage)))
	}
	currentpage := fmt.Sprintf("%d/%d", CurrentPage, Pages)
	nav = append(nav, tgbotapi.NewInlineKeyboardButtonData(currentpage,
		fmt.Sprintf("current_%s_%d", Type, CurrentPage)))
	if CurrentPage < Pages {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("Вперед →",
			fmt.Sprintf("next_%s_%d", Type, CurrentPage)))
	}

	if len(nav) > 0 {
		rows = append(rows, nav)
	}

	if showKeyboard { // если showKeyboard = true то ряды айдишников товаров
		if Type == "buyproducts" {
			var currentRow []tgbotapi.InlineKeyboardButton
			for i, item := range data {
				product := item.(models.Product)
				if i > 0 && i%5 == 0 { //кнопок в ряду
					rows = append(rows, currentRow)
					currentRow = []tgbotapi.InlineKeyboardButton{}
				}
				currentRow = append(currentRow, tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("ID%d", product.ID),
					fmt.Sprintf("product_%d", product.ID)))
			}
			if len(currentRow) > 0 {
				rows = append(rows, currentRow)
			}
		}
		if Type == "buycategories" {
			var currentRow []tgbotapi.InlineKeyboardButton
			for i, item := range data {
				var buttonText, callbackData string
				switch v := item.(type) {
				case models.Category: //работа с категориями
					buttonText = fmt.Sprintf("%d", v.ID)
					callbackData = fmt.Sprintf("category_%d", v.ID)
				case models.Product: //работа с товарами
					buttonText = fmt.Sprintf("%d", v.ID)
					callbackData = fmt.Sprintf("product_%d", v.ID)
				default:
					continue // пропускаем неизвестный тип
				}
				if i > 0 && i%5 == 0 { // кнопок в ряду
					rows = append(rows, currentRow)
					currentRow = []tgbotapi.InlineKeyboardButton{}
				}
				currentRow = append(currentRow, tgbotapi.NewInlineKeyboardButtonData(
					buttonText,
					callbackData))
			}
			if len(currentRow) > 0 {
				rows = append(rows, currentRow)
			}
		}
	}

	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("Главная", "start"),
	})

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (h *Handler) CreateBuyingKeyboard(total_quantity int) tgbotapi.InlineKeyboardMarkup { // функция создания клавиатуры для покупки товара
	var rows [][]tgbotapi.InlineKeyboardButton

	var nav []tgbotapi.InlineKeyboardButton

	if total_quantity > 1 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("-",
			fmt.Sprintf("buying_del_%d", total_quantity)))
	}
	quantity := fmt.Sprintf("%d", total_quantity)
	nav = append(nav, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v", quantity),
		fmt.Sprintf("buying_quantity_%d", total_quantity)))
	nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("+",
		fmt.Sprintf("buying_add_%d", total_quantity)))
	if len(nav) > 0 {
		rows = append(rows, nav)
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("Отмена", "cancell"),
		tgbotapi.NewInlineKeyboardButtonData("Подтвердить", "confirm"),
	})
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (h *Handler) CreateCategoriesKeyboard(CurrentPage, Pages int, data []interface{}) tgbotapi.InlineKeyboardMarkup { //функция создания клавиатуры для выбора категории
	var rows [][]tgbotapi.InlineKeyboardButton

	var nav []tgbotapi.InlineKeyboardButton
	Type := "buycategories"
	if CurrentPage > 1 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("← Назад",
			fmt.Sprintf("prev_%s_%d", Type, CurrentPage)))
	}
	currentpage := fmt.Sprintf("%d/%d", CurrentPage, Pages)
	nav = append(nav, tgbotapi.NewInlineKeyboardButtonData(currentpage,
		fmt.Sprintf("current_%s_%d", Type, CurrentPage)))
	if CurrentPage < Pages {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("Вперед →",
			fmt.Sprintf("next_%s_%d", Type, CurrentPage)))
	}

	if len(nav) > 0 {
		rows = append(rows, nav)
	}
	rows = append(rows, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("Главная", "start"),
	})

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (h *Handler) ShowBuying(bot *tgbotapi.BotAPI, ChatID int64, MessageID, total_quantity int) { //не используется но аналогия с пагинацией
	h.mu.Lock()
	h.BuyingState[ChatID] = BuyingState{
		Total_quantity: total_quantity,
	}
	h.mu.Unlock()
	keyboard := h.CreateBuyingKeyboard(total_quantity)
	response := fmt.Sprintf("К покупке: %d", total_quantity)
	if MessageID != 0 {
		msg := tgbotapi.NewEditMessageText(ChatID, MessageID, response)
		msg.ReplyMarkup = &keyboard
		bot.Send(msg)
	} else {
		msg := tgbotapi.NewMessage(ChatID, response)
		msg.ReplyMarkup = keyboard
		bot.Send(msg)
	}
}

func (h *Handler) ConvertToInterfaceSlice(slice interface{}) ([]interface{}, error) { //конвертация любого слайса в []interface{}
	v := reflect.ValueOf(slice) // используется рефлексия для работы с любым типом слайса
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("convertToInterfaceSlice: expected slice, got %T", slice)
	}
	result := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		result[i] = v.Index(i).Interface()
	}
	return result, nil
}
