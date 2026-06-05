package handlers

import (
	"project/internal/repo"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/golang-jwt/jwt/v5"
)

type Handler struct {
	Bot          *tgbotapi.BotAPI
	ProductRepo  *repo.ProductRepo
	CategoryRepo *repo.CategoryRepo
	UserRepo     *repo.UserRepo
	OrderRepo    *repo.OrderRepo

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

	mu sync.RWMutex
}

type JWTConfig struct {
	SecretKey     string
	TokenDuration time.Duration
}

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
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

var (
	jwtConfig = JWTConfig{
		SecretKey:     "secret_key",
		TokenDuration: time.Minute,
	}
)

func NewHandler(bot *tgbotapi.BotAPI, productRepo *repo.ProductRepo,
	categoryRepo *repo.CategoryRepo, userRepo *repo.UserRepo,
	orderRepo *repo.OrderRepo) *Handler {

	return &Handler{
		Bot:          bot,
		ProductRepo:  productRepo,
		CategoryRepo: categoryRepo,
		UserRepo:     userRepo,
		OrderRepo:    orderRepo,

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
