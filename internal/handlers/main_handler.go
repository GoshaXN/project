package handlers

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (h *Handler) MainHandler() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := h.Bot.GetUpdatesChan(u)

	for update := range updates {
		h.LogUpdate(update)

		if update.CallbackQuery != nil {
			h.CallbackHandler(update.CallbackQuery)
		} else if update.Message != nil {
			h.UpdateHandler(update)
		}
	}
}
