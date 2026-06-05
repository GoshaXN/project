package handlers

import (
	"project/internal/service"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler struct {
	Bot             *tgbotapi.BotAPI
	productService  service.ProductService
	categoryService service.CategoryService
	userService     service.UserService
	orderService    service.OrderService
	authService     service.AuthService

	mu              sync.RWMutex
	UserTokens      map[int64]string
	PaginationState map[int64]PaginationState
	BuyingState     map[int64]BuyingState
	WaitingProduct  map[int64]bool
	WaitingUser     map[int64]bool
	WaitingCategory map[int64]bool
	WaitingConfirm  map[int64]func() error
	SelectProduct   map[int64]int
	SelectQuantity  map[int64]int
	SelectCategory  map[int64]int
}

type JWTConfig struct {
	SecretKey     string
	TokenDuration time.Duration
}

type PaginationState struct {
	CurrentPage int
	Pages       int
	Type        string
	Count       int
}

type BuyingState struct {
	Total_quantity int
}

func NewHandler(bot *tgbotapi.BotAPI,
	productSvc service.ProductService,
	categorySvc service.CategoryService,
	userSvc service.UserService,
	orderSvc service.OrderService,
	authSvc service.AuthService) *Handler {

	return &Handler{
		Bot:             bot,
		productService:  productSvc,
		categoryService: categorySvc,
		userService:     userSvc,
		orderService:    orderSvc,
		authService:     authSvc,

		UserTokens:      make(map[int64]string),
		PaginationState: make(map[int64]PaginationState),
		BuyingState:     make(map[int64]BuyingState),
		WaitingProduct:  make(map[int64]bool),
		WaitingUser:     make(map[int64]bool),
		WaitingCategory: make(map[int64]bool),
		WaitingConfirm:  make(map[int64]func() error),
		SelectProduct:   make(map[int64]int),
		SelectQuantity:  make(map[int64]int),
		SelectCategory:  make(map[int64]int),
	}
}
