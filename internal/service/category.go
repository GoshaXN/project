package service

import (
	"fmt"
	"project/internal/models"
	"project/internal/repo"
)

type CategoryService interface {
	CreateCategory(category *models.Category) error
	Categories() ([]models.Category, error)
	SearchCategory(query string) ([]models.Category, error)
	UpdateCategory(category *models.Category) error
	DeleteCategory(categoryID int) error
	CountCategories() (int, error)
	PaginateCategory(limit, offset int) ([]models.Category, error)
}

type categoryService struct {
	repo *repo.CategoryRepo
}

func NewCategoryService(repo *repo.CategoryRepo) CategoryService {
	return &categoryService{repo: repo}
}

func (s *categoryService) CreateCategory(category *models.Category) error {
	if category.Name == "" {
		return fmt.Errorf("название категории не может быть пустым")
	}
	return s.repo.CreateCategory(category)
}

func (s *categoryService) Categories() ([]models.Category, error) {
	return s.repo.AllCategories()
}

func (s *categoryService) SearchCategory(query string) ([]models.Category, error) {
	if query == "" {
		return nil, fmt.Errorf("поисковый запрос не может быть пустым")
	}
	return s.repo.SearchCategory(query)
}

func (s *categoryService) UpdateCategory(category *models.Category) error {
	if category.ID == 0 {
		return fmt.Errorf("ID категории обязателен")
	}
	return s.repo.UpdateCategory(category)
}

func (s *categoryService) DeleteCategory(categoryID int) error {
	return s.repo.DeleteCategory(categoryID)
}

func (s *categoryService) CountCategories() (int, error) {
	return s.repo.CountCategories()
}

func (s *categoryService) PaginateCategory(limit, offset int) ([]models.Category, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.PaginateCategory(limit, offset)
}
