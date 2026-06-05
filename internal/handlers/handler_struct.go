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

	mu                   sync.RWMutex
	UserTokens           map[int64]string
	PaginationState      map[int64]PaginationState
	DefaultPhotoFileID   string
	PhotoPaginationState map[int64]*PhotoPaginationState
	BuyingState          map[int64]BuyingState
	WaitingProduct       map[int64]bool
	WaitingUser          map[int64]bool
	WaitingCategory      map[int64]bool
	WaitingConfirm       map[int64]func() error
	WaitingProductPhoto  map[int64]int
	WaitingDefaultPhoto  map[int64]bool
	SelectProduct        map[int64]int
	SelectQuantity       map[int64]int
	SelectCategory       map[int64]int

	commandHandlers map[string]func(tgbotapi.Update)
}

type PhotoPaginationState struct {
	TextMessageID int
	PhotoMessage  []int
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

	h := &Handler{
		Bot:             bot,
		productService:  productSvc,
		categoryService: categorySvc,
		userService:     userSvc,
		orderService:    orderSvc,
		authService:     authSvc,

		UserTokens:           make(map[int64]string),
		PaginationState:      make(map[int64]PaginationState),
		PhotoPaginationState: make(map[int64]*PhotoPaginationState),
		DefaultPhotoFileID:   "",
		BuyingState:          make(map[int64]BuyingState),
		WaitingProduct:       make(map[int64]bool),
		WaitingUser:          make(map[int64]bool),
		WaitingCategory:      make(map[int64]bool),
		WaitingConfirm:       make(map[int64]func() error),
		WaitingProductPhoto:  make(map[int64]int),
		WaitingDefaultPhoto:  make(map[int64]bool),
		SelectProduct:        make(map[int64]int),
		SelectQuantity:       make(map[int64]int),
		SelectCategory:       make(map[int64]int),
	}

	h.commandHandlers = map[string]func(tgbotapi.Update){
		"start":              func(u tgbotapi.Update) { h.Start(u) },
		"help":               func(u tgbotapi.Update) { h.Help(u) },
		"create_category":    h.CreateCategory,
		"categories":         h.Categories,
		"search_by_category": h.SearchByCategory,
		"skip_photo":         h.SkipPhoto,
		"update_category":    h.UpdateCategory,
		"delete_category":    h.DeleteCategory,
		"create_user":        h.CreateUser,
		"users":              h.Users,
		"update_user":        h.UpdateUser,
		"delete_user":        h.DeleteUser,
		"create_product":     h.CreateProduct,
		"products":           h.Products,
		"update_product":     h.UpdateProduct,
		"delete_product":     h.DeleteProduct,
		"orders":             h.Orders,
		"delete_order":       h.DeleteOrder,
		"register":           h.Register,
		"login":              h.Login,
		"logout":             h.Logout,
		"token":              h.handleTokenCommand,
		"set_default_photo":  h.SetDefaultPhoto,
		"update_photo":       h.UpdateProductPhoto,
	}

	return h
}
