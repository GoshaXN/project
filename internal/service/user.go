package service

import (
	"fmt"
	"project/internal/models"
	"project/internal/repo"
)

type UserService interface {
	CreateUser(user *models.User, password ...string) error
	SearchUserTGID(TelegramID int64) (*models.User, error)
	AllUsers() ([]models.User, error)
	SearchUser(query string) ([]models.User, error)
	UpdateUser(user *models.User, updatePassword ...bool) error
	UpdatePassword(userID int, NewPassword string) error
	DeleteUser(userID int) error
	PaginateUsers(limit, offset int) ([]models.User, error)
	CountUsers() (int, error)
}

type userService struct {
	repo *repo.UserRepo
}

func NewUserService(repo *repo.UserRepo) UserService {
	return &userService{repo: repo}
}

func (s *userService) CreateUser(user *models.User, password ...string) error {
	if !(len(password) > 0 && password[0] != "") {
		return fmt.Errorf("недопустимый пароль")
	}
	return s.repo.CreateUser(user, password...)
}

func (s *userService) SearchUserTGID(TelegramID int64) (*models.User, error) {
	if TelegramID <= 0 {
		return nil, fmt.Errorf("TelegramID должен быть положительным")
	}
	return s.repo.SearchUserTGID(TelegramID)
}

func (s *userService) AllUsers() ([]models.User, error) {
	return s.repo.AllUsers()
}

func (s *userService) SearchUser(query string) ([]models.User, error) {
	if query == "" {
		return nil, fmt.Errorf("query не должно быть пустым")
	}
	return s.repo.SearchUser(query)
}

func (s *userService) UpdateUser(user *models.User, updatePassword ...bool) error {
	return s.repo.UpdateUser(user, updatePassword...)
}

func (s *userService) UpdatePassword(userID int, NewPassword string) error {
	if userID <= 0 {
		return fmt.Errorf("userID не должно быть отрицательным")
	}
	if len(NewPassword) == 0 {
		return fmt.Errorf("пароль не может быть пустым")
	}
	return s.repo.UpdatePassword(userID, NewPassword)
}

func (s *userService) DeleteUser(userID int) error {
	if userID <= 0 {
		return fmt.Errorf("userID не должно быть отрицательным")
	}
	return s.repo.DeleteUser(userID)
}

func (s *userService) PaginateUsers(limit, offset int) ([]models.User, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.PaginateUsers(limit, offset)
}

func (s *userService) CountUsers() (int, error) {
	return s.repo.CountUsers()
}
