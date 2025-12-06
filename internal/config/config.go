package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken  string
	DBHost    string
	DBPort    string
	DBUser    string
	DBPass    string
	DBName    string
	DBSSLMode string
}

func Load() (*Config, error) {
	_, filename, _, _ := runtime.Caller(0) // корневая папка проекта
	rootDir := filepath.Join(filepath.Dir(filename), "..", "..")

	envPath := filepath.Join(rootDir, ".env") // загрузка .env из корневой папки
	err := godotenv.Load(envPath)
	if err != nil {
		return nil, err
	}

	return &Config{ //подключение бд
		BotToken:  os.Getenv("BOT_TOKEN"),
		DBHost:    os.Getenv("DB_HOST"),
		DBPort:    os.Getenv("DB_PORT"),
		DBUser:    os.Getenv("DB_USER"),
		DBPass:    os.Getenv("DB_PASSWORD"),
		DBName:    os.Getenv("DB_NAME"),
		DBSSLMode: os.Getenv("DB_SSLMODE"),
	}, nil
}
