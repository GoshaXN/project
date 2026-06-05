package service

import (
	"fmt"
	"project/internal/models"
	"project/internal/repo"
)

type OrderService interface {
	CreateOrder(userID int64) (*models.Order, error)
	Orders() ([]models.Order, error)
	UserOrder(userID int64) ([]models.Order, error)
	ConfirmOrder(userID int64) (int, error)
	DetailCart(userID int64) (*models.OrderWithItems, error)
	AddItemToCart(orderID, productID int, quantity int, price float64) error
	SearchOrder(orderID int) (*models.Order, error)
	PaginateOrders(limit, offset int) ([]models.Order, error)
	CountOrders() (int, error)
	PaginateUserOrders(UserID, limit, offset int) ([]models.Order, error)
	CountUserOrders(UserID int) (int, error)
	DeleteOrder(orderID int) error
}

type orderService struct {
	repo *repo.OrderRepo
}

func NewOrderService(repo *repo.OrderRepo) OrderService {
	return &orderService{repo: repo}
}

func (s *orderService) CreateOrder(userID int64) (*models.Order, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("ID пользователя обязателен")
	}
	return s.repo.CreateOrder(userID)
}

func (s *orderService) Orders() ([]models.Order, error) {
	return s.repo.AllOrders()
}

func (s *orderService) UserOrder(userID int64) ([]models.Order, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("ID пользователя обязателен")
	}
	return s.repo.UserOrder(userID)
}

func (s *orderService) ConfirmOrder(userID int64) (int, error) {
	if userID <= 0 {
		return 0, fmt.Errorf("ID пользователя обязателен")
	}
	return s.repo.ConfirmOrder(userID)
}

func (s *orderService) DetailCart(userID int64) (*models.OrderWithItems, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("ID пользователя обязателен")
	}
	return s.repo.DetailCart(userID)
}

func (s *orderService) AddItemToCart(orderID, productID int, quantity int, price float64) error {
	if orderID <= 0 {
		return fmt.Errorf("ID пользователя обязателен")
	}
	if productID <= 0 {
		return fmt.Errorf("ID товара обязателен")
	}
	if quantity <= 0 {
		return fmt.Errorf("количество должно быть больше 0")
	}
	if price <= 0 {
		return fmt.Errorf("цена должна быть больше 0")
	}
	return s.repo.AddItemToCart(orderID, productID, quantity, price)
}

func (s *orderService) SearchOrder(orderID int) (*models.Order, error) {
	if orderID <= 0 {
		return nil, fmt.Errorf("ID заказа обязателен")
	}
	return s.repo.SearchOrder(orderID)
}

func (s *orderService) PaginateOrders(limit, offset int) ([]models.Order, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.PaginateOrders(limit, offset)
}

func (s *orderService) CountOrders() (int, error) {
	return s.repo.CountOrders()
}

func (s *orderService) PaginateUserOrders(userID, limit, offset int) ([]models.Order, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	if userID <= 0 {
		return nil, fmt.Errorf("ID пользователя обязателен")
	}
	return s.repo.PaginateUserOrders(userID, limit, offset)
}

func (s *orderService) CountUserOrders(userID int) (int, error) {
	if userID <= 0 {
		return 0, fmt.Errorf("ID пользователя обязателен")
	}
	return s.repo.CountUserOrders(userID)
}

func (s *orderService) DeleteOrder(orderID int) error {
	if orderID <= 0 {
		return fmt.Errorf("ID заказа обязателен")
	}
	return s.repo.DeleteOrder(orderID)
}
