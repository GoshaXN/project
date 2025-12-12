package handlers

import (
	"fmt"
	"log"
	"project/internal/models"
	"project/internal/repo"
	"project/internal/utils"
	"reflect"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/golang-jwt/jwt/v5"
)

type JWTConfig struct {
	SecretKey     string
	TokenDuration time.Duration // длительность жизни токена
}

var (
	jwtConfig = JWTConfig{
		SecretKey:     "secret_key",
		TokenDuration: time.Minute, //1 минуту
	}
	userTokens = make(map[int64]string) // Храним токены в памяти (в продакшене используйте Redis/БД)
)

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type PaginationState struct {
	CurrentPage int
	Pages       int
	Type        string //товары, заказы, юзеры что угодно что надо будет пагинировать
	Count       int
}

type BuyingState struct {
	Total_quantity int
}

type OrderState struct {
	ProductID   int
	ProductName string
	Flavor      string
	Weight      string
	Price       float64
	Step        int //шаги покупки. 1 - товар, 2 - вкус, 3 - размер и тд.
}

const DataOnPage = 5

var paginationState = make(map[int64]PaginationState) //состояние пагинации
var buyingState = make(map[int64]BuyingState)         //состояние покупки
var waitingProduct = make(map[int64]bool)             //состояние поиска товара 2м смс
var waitingUser = make(map[int64]bool)                //состояние поиска юзера 2м смс
var waitingCategory = make(map[int64]bool)            //состояние поиска категории 2м смс
var waitingConfirm = make(map[int64]func() error)     //чат и функция по удалению
var SelectProduct = make(map[int64]int)               //выбранный товар
var SelectQuantity = make(map[int64]int)              //выбранное количество
var SelectCategory = make(map[int64]int)              //выбранная категория

func GenerateToken(user *models.User) (string, error) {
	expirationTime := time.Now().Add(jwtConfig.TokenDuration)

	claims := &Claims{
		UserID:   int64(user.ID),
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   strconv.FormatInt(user.ID, 10),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtConfig.SecretKey))
}

func VerifyToken(tokenString string) (*Claims, error) { //верификация токена
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtConfig.SecretKey), nil
	})
	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func AuthenticateUser(tokenString string, userRepo *repo.UserRepo) (*models.User, error) { //аутентефикация по токену
	claims, err := VerifyToken(tokenString)
	if err != nil {
		return nil, err
	}
	users, err := userRepo.SearchUser(fmt.Sprintf("%d", claims.UserID))
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("user not found")
	}
	return &users[0], nil
}

func AuthMiddleware(handler func(bot *tgbotapi.BotAPI, update tgbotapi.Update, user *models.User, userRepo *repo.UserRepo)) func(bot *tgbotapi.BotAPI, update tgbotapi.Update, userRepo *repo.UserRepo) {
	return func(bot *tgbotapi.BotAPI, update tgbotapi.Update, userRepo *repo.UserRepo) { //аутентефикация в боте
		var user *models.User
		var err error
		token := GetTokenFromUpdate(update)
		if token != "" {
			delete(userTokens, GetChatID(update))
			user, err = AuthenticateUser(token, userRepo)
			if err != nil {
				msg := tgbotapi.NewMessage(GetChatID(update), "Используйте команду /login Для аутентификации")
				bot.Send(msg)
				return
			}
			userTokens[GetChatID(update)] = token
		} else {
			chatID := GetChatID(update)
			users, err := userRepo.SearchUser(fmt.Sprintf("%d", chatID))
			if err != nil || len(users) == 0 {
				msg := tgbotapi.NewMessage(GetChatID(update), "Используйте команду /login Для аутентификации")
				bot.Send(msg)
				return
			}
			user = &users[0]
		}
		handler(bot, update, user, userRepo)
	}
}

func GetTokenFromUpdate(update tgbotapi.Update) string { //извлечение токена из сообщения
	ChatID := GetChatID(update)
	if ChatID > 0 {
		if token, ok := userTokens[ChatID]; ok {
			return token
		}
	}

	if update.Message != nil && update.Message.CommandArguments() != "" {
		args := update.Message.CommandArguments()
		if strings.HasPrefix(args, "token:") {
			return strings.TrimPrefix(args, "token:")
		}
	}
	if update.Message != nil {
		if token, ok := userTokens[update.Message.Chat.ID]; ok {
			return token
		}
	}
	if update.CallbackQuery != nil {
		if token, ok := userTokens[update.CallbackQuery.Message.Chat.ID]; ok {
			return token
		}
	}
	return ""
}

func GetChatID(update tgbotapi.Update) int64 {
	if update.Message != nil {
		return update.Message.Chat.ID
	}
	if update.CallbackQuery != nil {
		return update.CallbackQuery.Message.Chat.ID
	}
	return 0
}

func CheckPermissions(userRepo *repo.UserRepo, token string, status int) error {
	user, err := AuthenticateUser(token, userRepo)
	if status == 1 && (user.Role == "user" || user.Role == "admin") {
		return nil
	} else if status == 2 && (user.Role == "admin") {
		return nil
	} else if status == 0 {
		return nil
	} else {
		return err
	}
}

func CreateBuyingKeyboard(total_quantity int) tgbotapi.InlineKeyboardMarkup { // функция создания клавиатуры для покупки товара
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

func CreateCategoriesKeyboard(CurrentPage, Pages int, data []interface{}) tgbotapi.InlineKeyboardMarkup { //функция создания клавиатуры для выбора категории
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

func ShowBuying(bot *tgbotapi.BotAPI, ChatID int64, MessageID, total_quantity int) { //не используется но аналогия с пагинацией

	buyingState[ChatID] = BuyingState{
		Total_quantity: total_quantity,
	}
	keyboard := CreateBuyingKeyboard(total_quantity)
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

func ShowPagination(bot *tgbotapi.BotAPI, ChatID int64, MessageID int, Page int, //универсальная функция показа данных на страницу с пагинацией
	CountData func() (int, error), //подсчёт страниц. Передается к примеру productRepo.CountProduct
	PaginationFunc func(limit, offset int) ([]interface{}, error), //возрат данных одной страницы
	formatFunc func(interface{}) string, //форматирование(вывод) данных
	title string, paginationType string, showKeyboard bool) {
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
		delete(SelectCategory, ChatID)
		return
	}

	pages := (count + DataOnPage - 1) / DataOnPage
	paginationState[ChatID] = PaginationState{
		CurrentPage: Page,
		Pages:       pages,
		Type:        paginationType,
		Count:       count,
	}

	response := fmt.Sprintf("Все %s\n\n", title)
	for _, item := range data {
		response += formatFunc(item) + "\n"
	}

	keyboard := CreatePaginationKeyboard(Page, pages, paginationType, data, showKeyboard)

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

func CreatePaginationKeyboard(CurrentPage, Pages int, Type string, data []interface{}, showKeyboard bool) tgbotapi.InlineKeyboardMarkup { //создание клавиатуры перелистывания
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

func HandleUpdates(bot *tgbotapi.BotAPI, productRepo *repo.ProductRepo, categoryRepo *repo.CategoryRepo, //мейн функция обработки написанных сообщений
	userRepo *repo.UserRepo, orderRepo *repo.OrderRepo) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.CallbackQuery != nil {
			handleCallback(bot, update.CallbackQuery, productRepo, categoryRepo, userRepo, orderRepo)
			continue
		}

		if update.Message == nil {
			continue
		}

		userID := update.Message.From.ID
		userName := update.Message.From.FirstName
		if update.Message.From.UserName != "" {
			userName = update.Message.From.UserName
		}
		var msg tgbotapi.MessageConfig
		var action string

		if waitingProduct[update.Message.Chat.ID] && !update.Message.IsCommand() { //проверка на ожидание для возможности поиска товара 2м сообщением
			searchQuery := update.Message.Text
			action = "search product 2nd msg"

			products, err := productRepo.SearchProduct(searchQuery)
			if err != nil {
				log.Printf("Ошибка: %v", err)
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка поиска")
			} else if len(products) == 0 {
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "По запросу: "+searchQuery+" товаров не найдено")
			} else {
				response := "Результаты поиска по запросу: " + searchQuery + "\n\n"
				for _, product := range products {
					response += formatProduct(product) + "\n"
				}
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, response)
			}
			waitingProduct[update.Message.Chat.ID] = false // сбрасываем ожидание

		} else if waitingUser[update.Message.Chat.ID] && !update.Message.IsCommand() { //проверка на ожидание для возможности поиска юзера 2м сообщением
			searchQuery := update.Message.Text
			action = "search user 2nd msg"

			users, err := userRepo.SearchUser(searchQuery)
			if err != nil {
				log.Printf("Ошибка: %v", err)
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка поиска")
			} else if len(users) == 0 {
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "По запросу: "+searchQuery+" пользователей не найдено")
			} else {
				response := "Результаты поиска по запросу: " + searchQuery + "\n\n"
				for _, user := range users {
					response += formatUser(user) + "\n"
				}
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, response)
			}
			waitingUser[update.Message.Chat.ID] = false // сбрасываем ожидание

		} else if waitingCategory[update.Message.Chat.ID] && !update.Message.IsCommand() { //проверка на ожидание для возможности поиска категории 2м сообщением
			searchQuery := update.Message.Text
			action = "search category 2nd msg"

			categories, err := categoryRepo.SearchCategory(searchQuery)
			if err != nil {
				log.Printf("Ошибка: %v", err)
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка поиска")
			} else if len(categories) == 0 {
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "По запросу: "+searchQuery+" категорий  не найдено")
			} else {
				response := "Результаты поиска по запросу: " + searchQuery + "\n\n"
				for _, categories := range categories {
					response += formatCategory(categories) + "\n"
				}
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, response)
			}
			waitingCategory[update.Message.Chat.ID] = false // сбрасываем ожидание
		} else if deleteFunc := waitingConfirm[update.Message.Chat.ID]; deleteFunc != nil {
			confirm := update.Message.Text
			if confirm == "+" {
				if err := deleteFunc(); err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка удаления: "+err.Error())
				} else {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Успешное удаление!")
				}
				waitingConfirm[update.Message.Chat.ID] = nil
			} else {
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Отмена удаления")
				waitingConfirm[update.Message.Chat.ID] = nil
			}
		} else if update.Message.IsCommand() { //отбираем все после /

			switch update.Message.Command() {
			case "register":
				args := update.Message.CommandArguments()
				action = "register"

				if args == "" {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						"Используйте команду: /register password|UserID для регистрации")
					break
				}
				data := strings.Split(args, "|")
				var password string
				var TelegramID int64
				var err error
				if len(data) == 1 { //введён только пароль
					password = data[0]
					TelegramID = update.Message.From.ID
				} else if len(data) == 2 { //введён и пароль и юзер
					password = data[0]
					TelegramID, err = strconv.ParseInt(data[1], 10, 64)
					if err != nil {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID,
							"Ошибка: Telegram ID должен быть числом")
						break
					}
				} else { //обработка некорректной команды
					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						"Некорректный формат. Используйте: /register password|TelegramID")
					break
				}
				users, err := userRepo.SearchUserTGID(TelegramID)

				if err != nil && !strings.Contains(err.Error(), "user not found") { //ошибка отсутствия юзера
					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						fmt.Sprintf("Ошибка поиска пользователя: %v", err))
					break
				}

				if users != nil { //обработка существующего пользователя
					if users.Password != "" { //вход по паролю
						msg = tgbotapi.NewMessage(update.Message.Chat.ID,
							"Пользователь уже зарегистрирован. Войдите: /login password|TelegramID")
						msgToUser := tgbotapi.NewMessage(TelegramID,
							fmt.Sprintf("Напоминание пароля для аккаунта ID=%d", users.ID))
						bot.Send(msgToUser)
					} else { //обновление пароля
						err = userRepo.UpdatePassword(int(users.ID), password)
						if err != nil {
							msg = tgbotapi.NewMessage(update.Message.Chat.ID,
								fmt.Sprintf("Ошибка установки пароля: %v", err))
							break
						}
						token, err := GenerateToken(users)
						if err != nil {
							msg = tgbotapi.NewMessage(update.Message.Chat.ID,
								fmt.Sprintf("Ошибка генерации токена: %v", err))
							break
						}
						userTokens[update.Message.Chat.ID] = token
						msg = tgbotapi.NewMessage(update.Message.Chat.ID,
							fmt.Sprintf("Пароль установлен для пользователя %s. Сессия активна 10 минут.",
								users.FirstName))
					}
				} else { //создание нового пользователя
					username := update.Message.From.UserName
					if username == "" {
						username = strconv.FormatInt(TelegramID, 10)
					}

					NewUser := &models.User{
						TelegramID: TelegramID,
						Username:   username,
						FirstName:  update.Message.From.FirstName,
						Phone:      "",
						Email:      "",
						Role:       "user",
					}
					err = userRepo.CreateUser(NewUser, password)
					if err != nil {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID,
							fmt.Sprintf("Ошибка создания пользователя: %v", err))
						break
					}
					token, err := GenerateToken(NewUser)
					if err != nil {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID,
							fmt.Sprintf("Ошибка генерации токена: %v", err))
						break
					}

					userTokens[update.Message.Chat.ID] = token
					msgToUser := tgbotapi.NewMessage(TelegramID,
						fmt.Sprintf("Ваш пароль для аккаунта ID=%d установлен", NewUser.ID))
					bot.Send(msgToUser)

					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						fmt.Sprintf("Пользователь %s успешно зарегистрирован. Сессия активна 10 минут.",
							NewUser.FirstName))
				}
			case "login":
				action = "login"
				args := update.Message.CommandArguments()
				if args == "" {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						"Используйте команду:\n/login password|TelegramID")
					break
				}
				data := strings.Split(args, "|")
				var password string
				var TelegramID int64
				var err error
				if len(data) == 1 { // вход в текущий аккаунт
					password = data[0]
					TelegramID = update.Message.From.ID
				} else if len(data) == 2 { // вход в указанный аккаунт
					password = data[0]
					TelegramID, err = strconv.ParseInt(data[1], 10, 64)
					if err != nil {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID,
							"Ошибка: Telegram ID должен быть числом")
						break
					}
				} else {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						"Некорректный формат. Используйте: /login password|TelegramID")
					break
				}
				user, err := userRepo.SearchUserTGID(TelegramID)
				if err != nil {
					if strings.Contains(err.Error(), "user not found") {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID,
							"Пользователь не найден. Пройдите регистрацию: /register password|TelegramID")
					} else {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID,
							fmt.Sprintf("Ошибка поиска пользователя: %v", err))
					}
					break
				}
				if user.Password == "" {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Отсутствует пароль. Введите\n/register password|TelegramID для установки пароля")
					break
				}
				if !utils.CheckPasswordHash(password, user.Password) {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Неверный пароль!")
					break
				}
				token, err := GenerateToken(user)
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						fmt.Sprintf("Ошибка генерации токена: %v", err))
					break
				}
				userTokens[update.Message.Chat.ID] = token
				msg = tgbotapi.NewMessage(update.Message.Chat.ID,
					fmt.Sprintf("Здравствуйте, %s!\nВаш статус: %s\nID: %d\nСессия активна 10 минут",
						user.FirstName, user.Role, user.ID))
			case "token":
				action = "token"
				token := GetTokenFromUpdate(update)
				var user *models.User
				var err error

				if token != "" { //активный токен
					user, err = AuthenticateUser(token, userRepo)
					if err != nil {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Токен недействителен. Сначала войдите: /login")
						break
					}
				} else { //по отправителю
					TelegramID := update.Message.Chat.ID
					user, err = userRepo.SearchUserTGID(TelegramID)
					if err != nil {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Вы не авторизованы. Сначала войдите: /login")
						break
					}
				}
				NewToken, err := GenerateToken(user)
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						fmt.Sprintf("Ошибка генерации токена: %v", err))
					break
				}
				userTokens[update.Message.Chat.ID] = NewToken
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Ваш новый токен: %s\nДействует 10 минут", NewToken))
				msg.ParseMode = "Markdown"

			case "logout":
				action = "logout"
				delete(userTokens, update.Message.Chat.ID)
				delete(SelectProduct, update.Message.Chat.ID)
				delete(SelectCategory, update.Message.Chat.ID)
				delete(buyingState, update.Message.Chat.ID)
				delete(SelectQuantity, update.Message.Chat.ID)
				delete(waitingProduct, update.Message.Chat.ID)
				delete(waitingUser, update.Message.Chat.ID)
				delete(waitingCategory, update.Message.Chat.ID)
				delete(waitingConfirm, update.Message.Chat.ID)
				delete(paginationState, update.Message.Chat.ID)
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Успешный выхох из программы. Вход: /login")
			case "create_order": //создание заказа
				action = "create_order"
				user1 := strconv.Itoa(int(update.Message.From.ID))
				users, err := userRepo.SearchUser(user1)
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Ошибка поиска пользователя: %v", err))
					break
				}
				if len(users) == 0 {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Пользователь не найден")
					break
				}
				user := &users[0]
				token, err := GenerateToken(user)
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Ошибка генерации токена: %v", err))
					break
				}
				err = CheckPermissions(userRepo, token, 1)
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Недостаточно прав доступа")
					break
				}
				order, err := orderRepo.CreateOrder(user.ID)
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка создания заказа")
				} else {
					response := fmt.Sprintf("Заказ создан\nНомер заказа: %d", order.ID)
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, response)
				}
			case "orders": //все заказы
				action = "orders"
				ShowPagination(bot, update.Message.Chat.ID, 0, 1,
					orderRepo.CountOrders,
					func(limit, offset int) ([]interface{}, error) {
						orders, err := orderRepo.PaginateOrders(limit, offset)
						if err != nil {
							return nil, err
						}
						return convertToInterfaceSlice(orders)
					},
					func(data interface{}) string {
						return formatOrder(data.(models.Order), userRepo)
					},
					"заказы",
					"orders",
					false)
				continue
			case "cart": //корзина(последний заказ)
				action = "cart"
				users, err := userRepo.SearchUser(fmt.Sprintf("%d", update.Message.Chat.ID))
				if err != nil {
					log.Printf("Error loading cart: %v", err)
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка загрузки корзины!")
				} else if users == nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Нет пользователя!")
				} else {
					user := users[0]
					cart, err := orderRepo.DetailCart(int64(user.ID))
					if err != nil {
						log.Printf("Error loading cart: %v", err)
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка загрузки корзины!")
					} else if cart == nil {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Нет заказов!")
					} else {
						response := "Ваша корзина:\n\n"
						response = formatCart(&cart.Order, cart.Items, productRepo)
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, response)
					}
				}
			case "users": //юзеры
				action = "users"
				users, err := userRepo.AllUsers()
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка загрузки юзеров")
				} else {
					response := "Все юзеры\n\n"
					for _, user := range users {
						response += formatUser(user) + "\n"
					}
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, response)
				}
			case "search_user": //поиск юзера
				action = "search_user"
				searchQuery := update.Message.CommandArguments()

				if searchQuery == "" {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Укажите имя пользователя для поиска")
					waitingUser[update.Message.Chat.ID] = true // поднимаем флаг если не будет поиска 1м сообщением
				} else {
					users, err := userRepo.SearchUser(searchQuery)
					if err != nil {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка поиска")
					} else if len(users) == 0 {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, "По запросу: "+searchQuery+" пользователей не найдено")
					} else {
						response := "Результаты поиска по запросу: " + searchQuery + "\n\n"
						for _, user := range users {
							response += formatUser(user) + "\n"
						}
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, response)
					}
				}
			case "create_user": //создание пользователя
				action = "create_user"
				var password string
				data := strings.Split(update.Message.CommandArguments(), "|")

				if len(data) < 6 {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						"Некорректный формат. Используйте\n /create_user telegram_id|telegram_username|first_name|phone|email|role|password\n")
					break
				}
				TelegramID, err := strconv.ParseInt(data[0], 10, 64)
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка: telegram_id должен быть числом")
					break
				}

				user := &models.User{
					TelegramID: TelegramID,
					Username:   data[1],
					FirstName:  data[2],
					Phone:      data[3],
					Email:      data[4],
					Role:       data[5],
				}
				if len(data) > 6 {
					password = data[6]
				}

				err = userRepo.CreateUser(user, password)
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Ошибка создания пользователя: %v", err))
				} else {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						fmt.Sprintf("Создан пользователь\nID: %d\nTelegramID: %d\nНик: %s\nИмя: %s\nТелефон: %v\nПочта: %s\nРоль: %s\nПароль: %s",
							user.ID, user.TelegramID, user.Username, user.FirstName,
							user.Phone, user.Email, user.Role, password))
				}

			case "delete_user": //удаление пользователя
				action = "delete_user"
				data := strings.Fields(update.Message.CommandArguments())
				if len(data) == 0 {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Отправьте команду в формате /delete_user user_id")
					break
				}
				userID, err := strconv.Atoi(data[0])
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "ID должно быть числом")
					break
				}
				users, err := userRepo.SearchUser(fmt.Sprintf("%d", userID))
				if err != nil || len(users) == 0 {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Пользователь не найден")
					break
				}
				waitingConfirm[update.Message.Chat.ID] = func() error { return userRepo.DeleteUser(userID) }
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf(
					"Напишите + если хотите удалить пользователя: %s, %s, ID = %d", users[0].FirstName, users[0].Username, userID))

			case "update_user": //обновление пользователя
				action = "update_user"
				data := strings.Split(update.Message.CommandArguments(), "|")

				if len(data) < 7 {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						"Некорректный формат. Используйте\n /update_user id|telegram_id|telegram_username|first_name|phone|email|role\nНеизменённые поля заполнять символом *")
					break
				}

				users, err := userRepo.SearchUser(data[0])
				if err != nil || len(users) == 0 {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Пользователь не найден")
					break
				}
				user := &users[0]

				NewUser := []*string{&user.Username, &user.FirstName, &user.Phone, &user.Email, &user.Role} //строковые поля обрабатываются

				for i := 2; i < len(data) && i-2 < len(NewUser); i++ {
					if data[i] != "*" {
						*NewUser[i-2] = data[i]
					}
				}

				if data[1] != "*" { //обработка числового значения TG_ID
					TelegramID, _ := strconv.ParseInt(data[1], 10, 64)
					user.TelegramID = TelegramID
				}

				err = userRepo.UpdateUser(user)
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка изменения пользователя: "+err.Error())
					break
				} else {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						fmt.Sprintf("Изменен пользователь\nID: %d\nTelegramID: %d\nНик: %s\nИмя: %s\nТелефон: %v\nПочта: %s\nРоль: %s",
							user.ID, user.TelegramID, user.Username, user.FirstName,
							user.Phone, user.Email, user.Role))
				}
			case "create_product": //содание товара
				action = "create_product"
				data := strings.Split(update.Message.CommandArguments(), "|")

				if len(data) < 10 {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						"Некорректный формат. Используйте\n /create_product name|description|flawor|brand|price|quantity|category_id|weight|servings|is_active\n")
					break
				}

				product := &models.Product{}

				for i, field := range []*string{&product.Name, &product.Description,
					&product.Flavor, &product.Brand} {
					*field = data[i]
				}

				for i, field := range []interface{}{&product.Price, &product.Quantity, &product.Category_id, &product.Weight,
					&product.Servings, &product.IsActive} {
					switch field := field.(type) {
					case *float64:
						*field, _ = strconv.ParseFloat(data[i+4], 64)
					case *int:
						*field, _ = strconv.Atoi(data[i+4])
					case *bool:
						*field, _ = strconv.ParseBool(data[i+4])
					}
				}

				err := productRepo.CreateProduct(product)
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка создания товара: "+err.Error())
				} else {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						fmt.Sprintf("Создан товар\nID: %d\nНазвание: %s\nОписание: %s\nЦена: %.2f\nКоличество: %d\nКатегория ID: %d\nВес: %v\nВкус: %s\nБренд: %s\nПорций: %d\nАктивен: %v",
							product.ID, product.Name, product.Description, product.Price, product.Quantity,
							product.Category_id, product.Weight, product.Flavor, product.Brand, product.Servings,
							product.IsActive))
				}
			case "products": //товары
				action = "products"
				products, err := productRepo.AllProducts()
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка загрузки товаров")
				} else {
					response := "All products \n\n"
					for _, product := range products {
						response += formatProduct(product) + "\n"
					}
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, response)
				}
			case "search_product": //поиск товара
				action = "search_product"
				searchQuery := update.Message.CommandArguments()

				if searchQuery == "" {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Укажите название товара для поиска")
					waitingProduct[update.Message.Chat.ID] = true // поднимаем флаг если не будет поиска 1м сообщением
				} else {
					products, err := productRepo.SearchProduct(searchQuery)
					if err != nil {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка поиска")
					} else if len(products) == 0 {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, "По запросу: "+searchQuery+" товаров не найдено")
					} else {
						response := "Результаты поиска по запросу: " + searchQuery + "\n\n"
						for _, product := range products {
							response += formatProduct(product) + "\n"
						}
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, response)
					}
				}
			case "update_product": //обновление товара
				action = "update_product"
				data := strings.Split(update.Message.CommandArguments(), "|")

				if len(data) < 11 {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						"Некорректный формат. Используйте\n /update_product id|price|quantity|weight|category_id|servings|is_active|name|description|flavor|brand\nНеизменённые поля заполнять символом *")
					break
				}

				products, err := productRepo.SearchProduct(data[0])
				if err != nil || len(products) == 0 {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Товар не найден")
					break
				}
				product := &products[0] //инициализация товара который будет изменться

				for i, field := range []interface{}{&product.Price, &product.Quantity, &product.Weight, &product.Category_id,
					//конструкция для обработки int,float,bool подающегося поля
					&product.Servings, &product.IsActive} {
					switch field := field.(type) {
					case *float64:
						*field, _ = strconv.ParseFloat(data[i+1], 64)
					case *int:
						*field, _ = strconv.Atoi(data[i+1])
					case *bool:
						*field, _ = strconv.ParseBool(data[i+2])
					}
				}

				for i, field := range []*string{&product.Name, &product.Description, //обработка строковых полей вкуса и бренда
					&product.Flavor, &product.Brand} {
					*field = data[i+7]
				}

				err = productRepo.UpdateProduct(product) //внесённые изменения вносятся в товар
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка изменения товара: "+err.Error())
					break
				} else {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						fmt.Sprintf("Изменен товар\nID: %d\nНазвание: %s\nОписание: %s\nЦена: %.2f\nКоличество: %d\nКатегория ID: %d\nВес: %v\nВкус: %s\nБренд: %s\nПорций: %d\nАктивен: %v",
							product.ID, product.Name, product.Description, product.Price, product.Quantity,
							product.Category_id, product.Weight, product.Flavor, product.Brand, product.Servings,
							product.IsActive))
				}
			case "delete_product": //удаление товара
				action = "delete_product"
				data := strings.Fields(update.Message.CommandArguments())
				if len(data) == 0 {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Отправьте команду в формате /delete_product product_id")
					break
				}
				productID, err := strconv.Atoi(data[0])
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "ID должно быть числом")
				}
				product, err := productRepo.SearchProduct(fmt.Sprintf("%d", productID))
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Товар не найден")
					break
				}

				waitingConfirm[update.Message.Chat.ID] = func() error { return productRepo.DeleteProduct(productID) }
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf(
					"Напишите + если хотите удалить товар: %s, ID = %d", product[0].Name, productID))
			case "delete_order": //удаление заказа
				action = "delete_order"
				data := strings.Fields(update.Message.CommandArguments())
				if len(data) == 0 {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Отправьте команду в формате /delete_order order_id")
					break
				}
				orderID, err := strconv.Atoi(data[0])
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "ID должно быть числом")
				}
				order, err := orderRepo.SearchOrder(orderID)
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Заказ не найден")
					break
				}
				waitingConfirm[update.Message.Chat.ID] = func() error { return orderRepo.DeleteOrder(orderID) }
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf(
					"Напишите + если хотите удалить Заказ: ID=%d, ID владельца: %d,", orderID, order.UserID))
			case "create_category": //создание категории
				action = "create_category"
				data := strings.Split(update.Message.CommandArguments(), "|")

				if len(data) < 3 {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						"Некорректный формат. Используйте\n /create_category name|description|is_active\n")
					break
				}
				is_active, err := strconv.ParseBool(data[2])
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "is_active должно быть true/false: "+err.Error())
					break
				}
				category := &models.Category{ //всего 3 поля которые можно изменять. id,created_at автоматические
					Name:        data[0],
					Description: data[1],
					IsActive:    is_active,
				}
				err = categoryRepo.CreateCategory(category)
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка создания категории: "+err.Error())
				} else {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						fmt.Sprintf("Создана категория %s\nID: %d\nОписание: %s",
							category.Name, category.ID, category.Description))
				}
			case "categories": //категории
				action = "categories"
				categories, err := categoryRepo.AllCategories()
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка загрузки категорий")
				} else {
					response := "Все категории:\n\n"
					for _, categories := range categories {
						response += formatCategory(categories)
					}
					response = response + "\n\n"
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, response)
				}
				/*case "search_category": //поиск товаров по категории
				action = "search_by_category"
				category := update.Message.CommandArguments()

				if category == "" {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Укажите название категории для поиска")
					waitingCategory[update.Message.Chat.ID] = true // поднимаем флаг если не будет поиска 1м сообщением
				} else {
					products, err := productRepo.ProductsByCategory(category)
					if err != nil {
						fmt.Printf("error: %v", err)
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка поиска")
					} else if len(products) == 0 {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, "По запросу: "+category+" категорий не найдено")
					} else {
						response := "Результаты поиска по запросу: " + category + "\n\n"
						for _, product := range products {
							response += formatProduct(product) + "\n"
						}
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, response)
					}
				}*/
			case "search_category": //поиск категорий
				action = "search_category"
				searchQuery := update.Message.CommandArguments()

				if searchQuery == "" {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Укажите название категории для поиска")
					waitingCategory[update.Message.Chat.ID] = true // поднимаем флаг если не будет поиска 1м сообщением
				} else {
					categories, err := categoryRepo.SearchCategory(searchQuery)
					if err != nil {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка поиска")
					} else if len(categories) == 0 {
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, "По запросу: "+searchQuery+" категорий не найдено")
					} else {
						response := "Результаты поиска по запросу: " + searchQuery + "\n\n"
						for _, category := range categories {
							response += formatCategory(category) + "\n"
						}
						msg = tgbotapi.NewMessage(update.Message.Chat.ID, response)
					}
				}
			case "update_category": //обновление категории
				action = "update_category"
				data := strings.Split(update.Message.CommandArguments(), "|")

				if len(data) < 4 {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						"Некорректный формат. Используйте\n /update_category id|name|description|is_active\nНеизменённые поля заполнять символом *")
					break
				}

				categories, err := categoryRepo.SearchCategory(data[0])
				if err != nil || len(categories) == 0 {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Категория не найдена")
					break
				}
				category := &categories[0] //категория для изменения

				for i, field := range []*string{&category.Name, &category.Description} { //строковые поля изменяются
					if data[i+1] != "*" {
						*field = data[i+1]
					}
				}

				if data[3] != "*" {
					IsActive, _ := strconv.ParseBool(data[3]) //бул значение
					category.IsActive = IsActive
				}

				err = categoryRepo.UpdateCategory(category)
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка изменения категории: "+err.Error())
					break
				} else {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID,
						fmt.Sprintf("Изменена категория\nID: %d\nИмя: %s\nОписание: %s\nАктивна: %v",
							category.ID, category.Name, category.Description, category.IsActive))
				}

			case "delete_category": //удаление категории
				action = "delete_category"
				data := strings.Fields(update.Message.CommandArguments())
				if len(data) == 0 {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Отправьте команду в формате /delete_category category_id")
					break
				}
				categoryID, err := strconv.Atoi(data[0])
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "ID должно быть числом")
					break
				}
				categories, err := categoryRepo.SearchCategory(fmt.Sprintf("%d", categoryID))
				if err != nil || len(categories) == 0 {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Категория не найдена")
					break
				}
				waitingConfirm[update.Message.Chat.ID] = func() error { return categoryRepo.DeleteCategory(categoryID) }
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf(
					"Напишите + если хотите удалить категорию: %s, %s, ID = %d", categories[0].Name, categories[0].Description, categoryID))
			case "start": //старт команда

				action = "command_start"

				delete(SelectProduct, update.Message.Chat.ID)
				delete(SelectCategory, update.Message.Chat.ID)
				delete(buyingState, update.Message.Chat.ID)
				delete(SelectQuantity, update.Message.Chat.ID)
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("%s, добро пожаловать в магазин спортивного питания!\nВаш TG_ID: %s\n\nВыберите нужное действие:", update.Message.From.FirstName, update.Message.From.UserName))

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

			case "help":
				action = "command help"
				msg = tgbotapi.NewMessage(update.Message.Chat.ID,
					"/start - начало\n/products - все товары\n/categories - все категории\n/search [product/user] [текст] - поиск товаров/пользователей\n/help - помощь\n/users - список пользователей")

			default: //если никакой кейс не сработает
				action = update.Message.Text
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Недоступная команда. повторите попытку")
			}
		} else {
			action = update.Message.Text
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Недоступная команда. повторите попытку")
		}
		/*если непонятно что ввелось можно обработать как введённое сообщение
		if action == "" {
		action = update.Message.Chat.ID
		}*/
		log.Printf("user_id: %d, username: %s, action: %s ", userID, userName, action) //лог введённой команды
		bot.Send(msg)
	}
}

func formatUser(user models.User) string { // вывод юзера
	roleText := "Покупатель"
	if user.Role == "admin" {
		roleText = "admin"
	}
	return fmt.Sprintf("%s: %s(ID=%d)\nТелеграмм: %s (ID=%d)\nИмя: %s\nТелефон: %s\nПочта: %s\nДата регистрации: %s\n",
		roleText, user.Username, user.ID, user.Username, user.TelegramID, user.FirstName,
		user.Phone, user.Email, user.CreatedAt.Format("02.01.2006"))
}
func formatProduct(product models.Product) string { // вывод товара
	return fmt.Sprintf("ID: %d\nНазвание: %s\nОписание: %s\nЦена: %.2f\nКоличество: %d\nКатегория ID: %d\nВес: %.2f\nВкус: %s\nБренд: %s\nПорций: %d\nАктивен: %v\nСоздан: %s\n\n",
		product.ID, product.Name, product.Description, product.Price, product.Quantity,
		product.Category_id, product.Weight, product.Flavor, product.Brand, product.Servings,
		product.IsActive, product.CreatedAt.Format("02.01.2006 15:04"))
}

func formatCategory(category models.Category) string { //вывод категории
	return fmt.Sprintf(" Категория: %s (%v) \nОписание: %s\nАктивность: %v\n\n",
		category.Name, category.ID, category.Description, category.IsActive)
}

func formatOrder(order models.Order, userRepo *repo.UserRepo) string { //вывод заказа
	users, err := userRepo.SearchUser(fmt.Sprintf("%d", order.UserID))
	if err != nil {
		return fmt.Sprintf("Пользователь: %d не найден", order.UserID)
	}
	//fmt.Printf("users: %v\n", users)
	user := users[0]
	return fmt.Sprintf("Заказ #%d\nПользователь: %s (%d)\nСумма: %.2f\nСтатус: %s\nДата создания: %s\n",
		order.ID, user.FirstName, order.UserID, order.Amount, order.Status, order.CreatedAt.Format("02.01.2006 15:04"))
}

func formatOrderPagination(order models.Order) string {
	return fmt.Sprintf("Заказ #%d\nПользователь ID: %d\nСумма: %.2f руб.\nСтатус: %s\nДата создания: %s\n",
		order.ID, order.UserID, order.Amount, order.Status,
		order.CreatedAt.Format("02.01.2006 15:04"))
}

func FormatUserPaginaasdasdtion(user models.User) string {
	return "hello"
}

func formatCart(order *models.Order, items []models.OrderItem, productRepo *repo.ProductRepo) string { //вывод корзины с товарами
	var response string
	if order == nil {
		return "Нет заказов!"
	}
	if len(items) == 0 {
		return "Пустая корзина"
	}

	total := 0.0
	for _, item := range items {
		sum := item.Price * float64(item.Quantity)
		total += sum
		product, err := productRepo.SearchProduct(fmt.Sprintf("%d", item.ProductID))
		productName := "Товар"
		var flavor string
		if err == nil && product != nil {
			productName = product[0].Name
			flavor = product[0].Flavor
		}

		response += fmt.Sprintf("Товар: %s (%s) %dшт. - %.2f руб.\n",
			productName, flavor, item.Quantity, sum)
	}

	response += fmt.Sprintf("\nОбщая сумма: %.2f руб.", total)
	response += fmt.Sprintf("\nНомер заказа: #%d", order.ID)
	response += fmt.Sprintf("\nСтатус заказа: #%s", order.Status)

	return response
}

func convertToInterfaceSlice(slice interface{}) ([]interface{}, error) { //конвертация любого слайса в []interface{}
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
func handleCallback(bot *tgbotapi.BotAPI, callback *tgbotapi.CallbackQuery, productRepo *repo.ProductRepo, //мейн функция обработки нажатий на кнопки
	categoryRepo *repo.CategoryRepo, userRepo *repo.UserRepo, orderRepo *repo.OrderRepo) {

	ChatID := callback.Message.Chat.ID
	MessageID := callback.Message.MessageID
	data := callback.Data
	var msg tgbotapi.MessageConfig
	var action string
	if strings.HasPrefix(data, "category_") { //data - то какое значение под собой содержит та или иная кнопка
		ID := strings.TrimPrefix(data, "category_")
		categoryID, err := strconv.Atoi(ID) //конвертация строки в инт (аналог Int в питоне)
		if err != nil {
			log.Printf("Ошибка конвертации: %v", err)
			return
		}
		SelectCategory[ChatID] = categoryID

		action = fmt.Sprintf("select_category_%s", data) //аналог ф строки конвертирующей int->str
		categories, err := categoryRepo.SearchCategory(fmt.Sprintf("%d", categoryID))
		var categoryName string
		if err == nil && len(categories) > 0 {
			categoryName = categories[0].Name
		} else {
			categoryName = fmt.Sprintf("Категория %d", categoryID)
		}
		ShowPagination(bot, ChatID, MessageID, 1, //1 = начальная страница
			func() (int, error) {
				return productRepo.CountProductsByCategory(fmt.Sprintf("%d", categoryID))
			},
			func(limit, offset int) ([]interface{}, error) {
				products, err := productRepo.PaginateProductsByCategory(fmt.Sprintf("%d", categoryID), limit, offset)
				if err != nil {
					return nil, err
				}
				return convertToInterfaceSlice(products)
			},
			func(data interface{}) string { return formatProduct(data.(models.Product)) },
			fmt.Sprintf("Товары категории: %s", categoryName),
			"buycategories",
			true)

		callbackConfig := tgbotapi.NewCallback(callback.ID, "")
		bot.Send(callbackConfig)
		log.Printf("user_id: %d, username: %s, action: %s", callback.From.ID, callback.From.FirstName, action)
		return
	}
	if strings.HasPrefix(data, "product_") { //нажатие по кнопке с ID в товарах
		ID := strings.TrimPrefix(data, "product_")
		productID, err := strconv.Atoi(ID)
		if err != nil {
			log.Printf("Ошибка конвертации: %v", err)
			return
		}

		action = fmt.Sprintf("select_product_%s", data)
		products, err := productRepo.SearchProduct(fmt.Sprintf("%d", productID))
		if err != nil || len(products) == 0 {
			msg = tgbotapi.NewMessage(ChatID, "Товар не найден")
			bot.Send(msg)
			return
		}

		product := products[0]
		SelectProduct[ChatID] = productID
		response := fmt.Sprintf("Выбран товар: %s (%s)\nЦена: %.2f руб.\nВыберите количество:", product.Name, product.Flavor, product.Price)
		keyboard := CreateBuyingKeyboard(1) //создает клавиатуру покупки
		editMsg := tgbotapi.NewEditMessageText(ChatID, MessageID, response)
		editMsg.ReplyMarkup = &keyboard
		bot.Send(editMsg)

		callbackConfig := tgbotapi.NewCallback(callback.ID, "")
		bot.Send(callbackConfig)
		log.Printf("user_id: %d, username: %s, action: %s", callback.From.ID, callback.From.FirstName, action)
		return
	}
	if strings.HasPrefix(data, "buying_") { //после выбора товара выбор количества
		parts := strings.Split(data, "_")
		if len(parts) < 3 {
			log.Printf("Неверный формат покупки: %s", data)
		}
		var total_quantity int

		total_quantity, _ = strconv.Atoi(parts[2])

		switch parts[1] {
		case "add":
			total_quantity += 1
			action = "buying_add_" + fmt.Sprintf("%d", total_quantity)

		case "del":
			total_quantity -= 1
			action = "buying_del_" + fmt.Sprintf("%d", total_quantity)
		case "quantity":
			action = "buying_quantity_" + fmt.Sprintf("%d", total_quantity)
		}
		SelectQuantity[ChatID] = total_quantity
		var response string
		if productID, ok := SelectProduct[ChatID]; ok && productID > 0 {
			products, err := productRepo.SearchProduct(fmt.Sprintf("%d", productID))
			if err == nil && len(products) > 0 {
				product := products[0]
				response = fmt.Sprintf("Выбран товар: %s\nЦена: %.2f руб.\n\nК покупке: %d",
					product.Name, product.Price, total_quantity)
			} else {
				response = fmt.Sprintf("К покупке: %d", total_quantity)
			}
		} else {
			response = fmt.Sprintf("К покупке: %d", total_quantity)
		}

		keyboard := CreateBuyingKeyboard(total_quantity)
		msg1 := tgbotapi.NewEditMessageText(ChatID, MessageID, response)
		msg1.ReplyMarkup = &keyboard
		bot.Send(msg1)
		callbackConfig := tgbotapi.NewCallback(callback.ID, "")
		bot.Send(callbackConfig)
		log.Printf("user_id: %d, username: %s, action: %s, quantity: %d",
			callback.From.ID, callback.From.FirstName, action, total_quantity)
		return
	}
	if data == "confirm" || data == "cancell" {

		productID, hasProduct := SelectProduct[ChatID]

		if data == "confirm" && hasProduct { //обработка добавления товара в корзину с укаанным количеством
			action = "confirm_purchase"
			var quantity int = 1
			if quantity < SelectQuantity[ChatID] {
				quantity = SelectQuantity[ChatID]
			}
			fmt.Printf("quantity: %d", quantity)
			users, err := userRepo.SearchUser(fmt.Sprintf("%d", ChatID))
			if err != nil || len(users) == 0 {
				msg = tgbotapi.NewMessage(ChatID, "Пользователь не найден")
			} else {
				user := users[0]
				products, err := productRepo.SearchProduct(fmt.Sprintf("%d", productID))
				if err != nil || len(products) == 0 {
					msg = tgbotapi.NewMessage(ChatID, "Товар не найден")
				} else {
					product := products[0]

					cart, err := orderRepo.DetailCart(int64(user.ID))
					if err != nil {
						msg = tgbotapi.NewMessage(ChatID, "Ошибка при работе с корзиной: "+err.Error())
					} else if cart == nil {
						order, err := orderRepo.CreateOrder(int64(user.ID))
						if err != nil {
							msg = tgbotapi.NewMessage(ChatID, "Ошибка создания заказа: "+err.Error())
						} else {
							err := orderRepo.AddItemToCart(order.ID, productID, quantity, product.Price)
							if err != nil {
								msg = tgbotapi.NewMessage(ChatID, "Ошибка добавления товара в корзину: "+err.Error())
							} else {
								msg = tgbotapi.NewMessage(ChatID,
									fmt.Sprintf("Товар добавлен в корзину\n\nЗаказ: #%d\nТовар: %s\nЦена товара: %.2f руб.\nКоличество: %d\nЦена: %.2f руб.",
										order.ID, product.Name, product.Price, quantity,
										product.Price*float64(quantity)))
							}
						}
					} else {
						err := orderRepo.AddItemToCart(cart.Order.ID, productID, quantity, product.Price) //добавление товара в существующую корзину
						if err != nil {
							msg = tgbotapi.NewMessage(ChatID, "Ошибка добавления товара в корзину: "+err.Error())
						} else {
							updatedCart, err := orderRepo.DetailCart(int64(user.ID))
							if err != nil {
								msg = tgbotapi.NewMessage(ChatID, "Ошибка получения обновленной корзины: "+err.Error())
							} else {
								var totalSum float64 //обновление суммы
								for _, item := range updatedCart.Items {
									totalSum += item.Price * float64(item.Quantity)
								}
								msg1 := tgbotapi.NewMessage(ChatID,
									fmt.Sprintf("Товар добавлен в корзину\n\nЗаказ: #%d\nТовар: %s (%s)\nЦена товара: %.2f руб.\nКоличество: %d\nСумма за товар: %.2f руб.\nСумма заказа: %.2f руб.",
										cart.Order.ID, product.Name, product.Flavor, product.Price, quantity,
										product.Price*float64(quantity), totalSum))
								delete(SelectProduct, ChatID) //очищается выбранный товар
								delete(buyingState, ChatID)   //очищается состояние покупки
								answermsg := tgbotapi.NewMessage(ChatID, "Хотите выбрать ещё товары?")
								keyboard := tgbotapi.NewInlineKeyboardMarkup(
									tgbotapi.NewInlineKeyboardRow(
										tgbotapi.NewInlineKeyboardButtonData("Да", "buyproducts"),
										tgbotapi.NewInlineKeyboardButtonData("Нет", "cart"),
									),
								)
								answermsg.ReplyMarkup = keyboard
								bot.Send(msg1)
								bot.Send(answermsg)
							}
						}
					}
				}
			}
			editMsg := tgbotapi.NewEditMessageReplyMarkup(
				ChatID,
				MessageID,
				tgbotapi.NewInlineKeyboardMarkup(),
			)
			bot.Send(editMsg)

		} else if data == "cancell" { //обработка кнопи отмены
			action = "cancel_purchase"
			delete(SelectProduct, ChatID)
			delete(SelectCategory, ChatID)
			delete(buyingState, ChatID)
			delete(SelectQuantity, ChatID)

			msg = tgbotapi.NewMessage(ChatID, "Отмена")

			editMsg := tgbotapi.NewEditMessageReplyMarkup(
				ChatID,
				MessageID,
				tgbotapi.NewInlineKeyboardMarkup(),
			)
			bot.Send(editMsg)
		}

		if msg.Text != "" {
			bot.Send(msg)
		}

		callbackConfig := tgbotapi.NewCallback(callback.ID, "")
		bot.Send(callbackConfig)
		log.Printf("user_id: %d, username: %s, action: %s",
			callback.From.ID, callback.From.FirstName, action)
		return
	}

	if bot == nil || callback == nil || productRepo == nil || //обработка возможных ошибок из
		categoryRepo == nil || userRepo == nil || orderRepo == nil {
		log.Printf("Один из параметров nil")
		return
	}

	handlers := map[string]struct { //структура, которая принимает значения (функции) чтобы для каждого случая был персональный вывод. уменьшает написание кода, упрощает добавление
		CountFunc      func() (int, error)                            //функция подсчёта товаров для пагинации
		PaginationFunc func(limit, offset int) ([]interface{}, error) //пагинационная функция с лимитом данных и отступом offset
		formatFunc     func(interface{}) string
		title          string
		showKeyboard   bool
	}{
		"products": {
			CountFunc: productRepo.CountProducts,
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				products, err := productRepo.PaginateProducts(limit, offset)
				if err != nil {
					return nil, err
				}
				return convertToInterfaceSlice(products)
			},
			formatFunc:   func(data interface{}) string { return formatProduct(data.(models.Product)) },
			title:        "товары",
			showKeyboard: false,
		},
		"buyproducts": {
			CountFunc: productRepo.CountProducts,
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				products, err := productRepo.PaginateProducts(limit, offset)
				if err != nil {
					return nil, err
				}
				return convertToInterfaceSlice(products)
			},
			formatFunc:   func(data interface{}) string { return formatProduct(data.(models.Product)) },
			title:        "товары",
			showKeyboard: true,
		},
		"users": {
			CountFunc: userRepo.CountUsers,
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				users, err := userRepo.PaginateUser(limit, offset)
				if err != nil {
					return nil, err
				}
				return convertToInterfaceSlice(users)
			},
			formatFunc:   func(data interface{}) string { return formatUser(data.(models.User)) },
			title:        "пользователи",
			showKeyboard: false,
		},
		"categories": {
			CountFunc: categoryRepo.CountCategories,
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				categories, err := categoryRepo.PaginateCategory(limit, offset)
				if err != nil {
					return nil, err
				}
				return convertToInterfaceSlice(categories)
			},
			formatFunc:   func(data interface{}) string { return formatCategory(data.(models.Category)) },
			title:        "категории",
			showKeyboard: false,
		},
		"orders": {
			CountFunc: func() (int, error) {
				UserID := callback.Message.Chat.ID
				users, err := userRepo.SearchUser(fmt.Sprintf("%d", UserID))
				if err != nil || len(users) == 0 {
					return 0, err
				}
				user := users[0]
				return orderRepo.CountUserOrders(int(user.ID))
			},
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				UserID := callback.Message.Chat.ID
				users, err := userRepo.SearchUser(fmt.Sprintf("%d", UserID))
				if err != nil || len(users) == 0 {
					return nil, err
				}
				user := users[0]
				orders, err := orderRepo.PaginateUserOrders(int(user.ID), limit, offset)
				if err != nil {
					return nil, err
				}
				return convertToInterfaceSlice(orders)
			},
			formatFunc:   func(data interface{}) string { return formatOrderPagination(data.(models.Order)) },
			title:        "ваши заказы",
			showKeyboard: false,
		},
		"buycategories": {
			CountFunc: func() (int, error) {
				if categoryID, ok := SelectCategory[ChatID]; ok { // если выбрана категория то показываем товары категории
					return productRepo.CountProductsByCategory(fmt.Sprintf("%d", categoryID))
				}
				return categoryRepo.CountCategories() // иначе список категорий
			},
			PaginationFunc: func(limit, offset int) ([]interface{}, error) {
				if categoryID, ok := SelectCategory[ChatID]; ok {
					products, err := productRepo.PaginateProductsByCategory(
						fmt.Sprintf("%d", categoryID), limit, offset)
					if err != nil {
						return nil, err
					}
					return convertToInterfaceSlice(products)
				}
				categories, err := categoryRepo.PaginateCategory(limit, offset)
				if err != nil {
					return nil, err
				}
				return convertToInterfaceSlice(categories)
			},
			formatFunc: func(data interface{}) string {
				switch v := data.(type) {
				case models.Category:
					return formatCategory(v)
				case models.Product:
					return formatProduct(v)
				default:
					return fmt.Sprintf("%v", data)
				}
			},
			title:        "категории и товары",
			showKeyboard: true,
		},
	}
	for dataType, handler := range handlers { //обработка перелистывания страниц
		if data == dataType ||
			strings.HasPrefix(data, "prev_"+dataType+"_") ||
			strings.HasPrefix(data, "next_"+dataType+"_") ||
			strings.HasPrefix(data, "current_"+dataType+"_") {
			var page int

			if data == dataType { // изначально выводим 1 страницу
				page = 1

				action = "callback_" + dataType
			} else { //а потом считаем
				parts := strings.Split(data, "_")
				currentPage, _ := strconv.Atoi(parts[2])

				switch parts[0] {
				case "current":
					page = currentPage
					action = "pagination_current_" + dataType
				case "prev":
					page = currentPage - 1
					if page < 1 {
						page = 1
					}
					action = "pagination_prev_" + dataType
				case "next":
					page = currentPage + 1
					action = "pagination_next_" + dataType
				}
			}

			ShowPagination(bot, ChatID, MessageID, page,
				handler.CountFunc,
				handler.PaginationFunc,
				handler.formatFunc,
				handler.title, dataType, dataType == "buyproducts" || dataType == "buycategories") //условие == || чтобы выводить доп клавиатуру выбора товара/категории

			callbackConfig := tgbotapi.NewCallback(callback.ID, "")
			bot.Send(callbackConfig)
			log.Printf("user_id: %d, username: %s, action: %s_page_%d", callback.From.ID, callback.From.FirstName, action, page)

		}

	}

	if data == "products" || data == "users" || data == "buyproducts" || data == "buycategories" || data == "orders" ||
		strings.HasPrefix(data, "prev_") || strings.HasPrefix(data, "next_") || strings.HasPrefix(data, "current_") {
		//пропускаем обработку пагинации во избежание возникновения ошибок ибо оно обработано уже
	} else {
		switch data {
		case "search_product": //поиск товара
			action = "callback_search_product"
			msg1 := tgbotapi.NewMessage(ChatID, "Укажите название товара для поиска")
			bot.Send(msg1)
			waitingProduct[ChatID] = true
			product := callback.Message.CommandArguments()

			if product == "" {
				msg = tgbotapi.NewMessage(ChatID, "Укажите название категории для поиска")
				waitingProduct[callback.Message.Chat.ID] = true // поднимаем флаг если не будет поиска 1м сообщением
			} else {
				products, err := productRepo.SearchProduct(product)
				if err != nil {
					fmt.Printf("error: %v", err)
					msg = tgbotapi.NewMessage(ChatID, "Ошибка поиска")
				} else if len(products) == 0 {
					msg = tgbotapi.NewMessage(ChatID, "По запросу: "+product+" товаров не найдено")
				} else {
					response := "Результаты поиска по запросу: " + product + "\n\n"
					for _, product := range products {
						response += formatProduct(product) + "\n"
					}
					msg = tgbotapi.NewMessage(ChatID, response)
				}
			}
		case "search_category": //поиск категорий
			action = "search_category"
			waitingCategory[ChatID] = true // поднимаем флаг поиска
			msg1 := tgbotapi.NewMessage(ChatID, "Введите запрос поиска категории:")
			bot.Send(msg1)
		case "create_order": //создание корзины
			action = "create_order"

			users, err := userRepo.SearchUser(fmt.Sprintf("%d", callback.Message.Chat.ID))
			if err != nil || users == nil {
				msg = tgbotapi.NewMessage(ChatID, "Ошибка создания заказа. Пользователь не найден")
			} else {
				user := users[0]
				order, err := orderRepo.CreateOrder(int64(user.ID))
				if err != nil {
					msg = tgbotapi.NewMessage(ChatID, "Ошибка создания заказа.")
				} else {
					response := fmt.Sprintf("Заказ создан\nНомер заказа: %d", order.ID)
					msg1 := tgbotapi.NewMessage(ChatID, response)
					bot.Send(msg1)

					msg2 := tgbotapi.NewMessage(callback.Message.Chat.ID, "Где будем искать товары?") //варианты добавления товаров в корзину
					keyboard := tgbotapi.NewInlineKeyboardMarkup(
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("По категориям", "buycategories")),
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("По товарам", "buyproducts")),
						tgbotapi.NewInlineKeyboardRow(
							tgbotapi.NewInlineKeyboardButtonData("Вывести ассортимент", "products")),
					)
					msg2.ReplyMarkup = keyboard
					bot.Send(msg2)
				}
			}
		case "confirm_order": //подтверждение заказа
			action = "confirm_order"
			user, err := userRepo.SearchUser(fmt.Sprintf("%d", callback.Message.Chat.ID))
			userID := user[0].ID
			if err != nil {
				msg = tgbotapi.NewMessage(ChatID, "Нет пользователя!")
			} else {
				orderID, err := orderRepo.ConfirmOrder(userID)
				if err != nil {
					msg = tgbotapi.NewMessage(ChatID, "Ошибка подтверждения заказа")
				} else {
					msg = tgbotapi.NewMessage(ChatID, fmt.Sprintf("Заказ #%d успешно сформирован!", orderID))
				}
			}
		case "cart": //последний активный заказ
			action = "cart"
			users, err := userRepo.SearchUser(fmt.Sprintf("%d", callback.Message.Chat.ID))
			if err != nil {
				log.Printf("Error loading cart: %v", err)
				msg = tgbotapi.NewMessage(ChatID, "Ошибка загрузки корзины!")
			} else if users == nil {
				msg = tgbotapi.NewMessage(ChatID, "Нет пользователя!")
			} else {
				user := users[0]
				cart, err := orderRepo.DetailCart(int64(user.ID))
				if err != nil {
					log.Printf("Error loading cart: %v", err)
					msg = tgbotapi.NewMessage(ChatID, "Ошибка загрузки корзины!")
				} else if cart == nil {
					msg = tgbotapi.NewMessage(ChatID, "Нет заказов!")
				} else {
					response := "Ваш заказ:\n\n"
					if formatCart(&cart.Order, cart.Items, productRepo) == "Пустая корзина" {
						msg1 := tgbotapi.NewMessage(ChatID, "Пустая корзина")
						keyboard := tgbotapi.NewInlineKeyboardMarkup(
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("Добавить товары", "buyproducts"),
							),
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("Вернуться на главную", "start"),
							),
						)
						msg1.ReplyMarkup = keyboard
						bot.Send(msg1)
						return
					} else {
						response += formatCart(&cart.Order, cart.Items, productRepo)
						msg = tgbotapi.NewMessage(ChatID, response)
						keyboard := tgbotapi.NewInlineKeyboardMarkup(
							tgbotapi.NewInlineKeyboardRow(
								tgbotapi.NewInlineKeyboardButtonData("Подтвердить заказ", "confirm_order"),
								tgbotapi.NewInlineKeyboardButtonData("Вернуться к покупкам", "buyproducts"),
							))
						msg.ReplyMarkup = keyboard
					}
				}
			}
		case "help":
			action = "command help"
			msg = tgbotapi.NewMessage(ChatID,
				"/start - начало\n/products - все товары\n/categories - все категории\n/search [product/user] [текст] - поиск товаров/пользователей\n/help - помощь\n/users - список пользователей")
		case "start": //старт команда
			action = "command start"
			delete(SelectProduct, ChatID)
			delete(SelectCategory, ChatID)
			delete(buyingState, ChatID)
			delete(SelectQuantity, ChatID)
			msg = tgbotapi.NewMessage(ChatID, "Добро пожаловать в магазин спортивного питания!\n\nВыберите нужное действие:")

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
					tgbotapi.NewInlineKeyboardButtonData("Мои заказы", "orders"),
					tgbotapi.NewInlineKeyboardButtonData("Создать заказ", "create_order"),
				),
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Выбрать товар для покупки", "buyproducts"),
				),
			)
			msg.ReplyMarkup = keyboard

		default:
		}
		bot.Send(msg)
		callbackConfig := tgbotapi.NewCallback(callback.ID, "")
		bot.Send(callbackConfig)
		log.Printf("user_id: %d, username: %s, action: %s ", callback.From.ID, callback.From.FirstName, action)
	}
}
