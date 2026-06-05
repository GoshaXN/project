package service

import (
	"fmt"
	"project/internal/models"
	"project/internal/repo"
)

type ProductService interface {
	CreateProduct(product *models.Product) error
	GetAllProducts() ([]models.Product, error)
	GetProductByID(id int) (*models.Product, error)
	SearchProduct(query string) ([]models.Product, error)
	SearchByCategory(query string) ([]models.Product, error)
	UpdateProduct(product *models.Product) error
	DeleteProduct(id int) error
	GetPaginatedProducts(limit, offset int) ([]models.Product, error)
	CountProducts() (int, error)
	GetProductsByCategory(categoryID int, limit, offset int) ([]models.Product, error)
	CountProductsByCategory(categoryID int) (int, error)
	PaginateProductsByCategory(categoryID string, limit, offset int) ([]models.Product, error)
	PaginateProducts(limit, offset int) ([]models.Product, error)
}

type productService struct {
	repo *repo.ProductRepo
}

func NewProductService(repo *repo.ProductRepo) ProductService {
	return &productService{repo: repo}
}

func (s *productService) CreateProduct(product *models.Product) error {
	if product.Name == "" {
		return fmt.Errorf("название товара не может быть пустым")
	}
	if product.Price < 0 {
		return fmt.Errorf("цена не может быть отрицательной")
	}
	if product.Quantity < 0 {
		return fmt.Errorf("количество не может быть отрицательным")
	}
	return s.repo.CreateProduct(product)
}

func (s *productService) GetAllProducts() ([]models.Product, error) {
	return s.repo.AllProducts()
}

func (s *productService) GetProductByID(id int) (*models.Product, error) {
	products, err := s.repo.SearchProduct(fmt.Sprintf("%d", id))
	if err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return nil, fmt.Errorf("товар с ID %d не найден", id)
	}
	return &products[0], nil
}

func (s *productService) SearchProduct(query string) ([]models.Product, error) {
	if query == "" {
		return nil, fmt.Errorf("поисковый запрос не может быть пустым")
	}
	return s.repo.SearchProduct(query)
}

func (s *productService) SearchByCategory(query string) ([]models.Product, error) {
	if query == "" {
		return nil, fmt.Errorf("поисковый запрос не может быть пустым")
	}
	return s.repo.ProductsByCategory(query)
}

func (s *productService) UpdateProduct(product *models.Product) error {
	if product.ID == 0 {
		return fmt.Errorf("ID товара обязателен")
	}
	return s.repo.UpdateProduct(product)
}

func (s *productService) DeleteProduct(id int) error {
	return s.repo.DeleteProduct(id)
}

func (s *productService) GetPaginatedProducts(limit, offset int) ([]models.Product, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.PaginateProducts(limit, offset)
}

func (s *productService) CountProducts() (int, error) {
	return s.repo.CountProducts()
}

func (s *productService) GetProductsByCategory(categoryID int, limit, offset int) ([]models.Product, error) {
	if categoryID <= 0 {
		return nil, fmt.Errorf("некорректный ID категории")
	}
	return s.repo.PaginateProductsByCategory(fmt.Sprintf("%d", categoryID), limit, offset)
}

func (s *productService) CountProductsByCategory(categoryID int) (int, error) {
	if categoryID <= 0 {
		return 0, fmt.Errorf("некорректный ID категории")
	}
	return s.repo.CountProductsByCategory(fmt.Sprintf("%d", categoryID))
}

func (s *productService) PaginateProductsByCategory(categoryID string, limit, offset int) ([]models.Product, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.PaginateProductsByCategory(categoryID, limit, offset)
}

func (s *productService) PaginateProducts(limit, offset int) ([]models.Product, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.PaginateProducts(limit, offset)
}
