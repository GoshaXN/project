package main

import (
	"log"
	"os"
	"project/internal/config"
	"project/internal/db"
	"project/internal/handlers"
	"project/internal/repo"
	"project/internal/service"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Ошибка загрузки конфига: %v", err)
	}
	//инициализация бд, репозиториев
	db, err := db.NewPostgresDB(cfg)
	if err != nil {
		log.Printf("Ошибка подключения к PG4: %v", err)
	}
	defer db.Close()

	productRepo := repo.NewProductRepo(db)
	categoryRepo := repo.NewCategoryRepo(db)
	userRepo := repo.NewUserRepo(db)
	orderRepo := repo.NewOrderRepo(db)

	ProductService := service.NewProductService(productRepo)
	CategoryService := service.NewCategoryService(categoryRepo)
	UserService := service.NewUserService(userRepo)
	OrderService := service.NewOrderService(orderRepo, ProductService)

	jwtSecret := "sfdkgfksdfnm,"
	tokenDuration := 10 * time.Minute

	AuthService := service.NewAuthService(userRepo, jwtSecret, tokenDuration)

	//создание бота
	Bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Printf("Ошибка создания бота: %v", err)
	}
	Bot.Debug = false
	log.Printf("Authorize %s", Bot.Self.UserName)

	handler := handlers.NewHandler(Bot, ProductService, CategoryService, UserService, OrderService, AuthService) // инициализация обработчика
	log.Printf("Bot Started")
	data, err := os.ReadFile("default_photo.txt")
	if err == nil {
		handler.DefaultPhotoFileID = string(data)
		log.Println("Загружено дефолтное фото из файла")
	}
	handler.MainHandler() // запуск обработчика
}
