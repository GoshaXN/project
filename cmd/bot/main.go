package main

import (
	"log"
	"project/internal/config"
	"project/internal/db"
	"project/internal/handlers"
	"project/internal/repo"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Panic("Ошибка загрузки конфига", err)
	}
	//инициализация бд, репозиториев
	db, err := db.NewPostgresDB(cfg)
	ProductRepo := repo.NewProductRepo(db)
	CategoryRepo := repo.NewCategoryRepo(db)
	UserRepo := repo.NewUserRepo(db)
	OrderRepo := repo.NewOrderRepo(db)
	if err != nil {
		log.Panic("Ошибка подключения к PG4", err)
	}
	defer db.Close()
	//создание бота
	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Panic("Ошибка создания бота", err)
	}
	bot.Debug = false
	log.Printf("Authorize %s", bot.Self.UserName)

	handlers.HandleUpdates(bot, ProductRepo, CategoryRepo, UserRepo, OrderRepo)
}
