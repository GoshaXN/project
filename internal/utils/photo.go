package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func SavePhoto(bot *tgbotapi.BotAPI, fileID string, productID int) (string, error) {
	file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return "", fmt.Errorf("failed to get file: %w", err)
	}
	if err := os.MkdirAll("uploads", 0755); err != nil {
		return "", fmt.Errorf("failed to create uploads directory: %w", err)
	}
	ext := filepath.Ext(file.FilePath)
	if ext == "" {
		ext = ".jpg"
	}
	filename := fmt.Sprintf("uploads/product_%d_%d%s", productID, time.Now().UnixNano(), ext)

	url, err := bot.GetFileDirectURL(file.FileID)
	if err != nil {
		return "", err
	}
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	out, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return filename, nil
}
