package repo

import (
	"database/sql"
	"log"
	"project/internal/models"
)

type CategoryRepo struct {
	db *sql.DB
}

func NewCategoryRepo(db *sql.DB) *CategoryRepo {
	return &CategoryRepo{db: db}
}

func (r *CategoryRepo) CreateCategory(category *models.Category) error {
	query := `
		INSERT INTO categories (name, description, is_active)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`
	err := r.db.QueryRow(
		query, category.Name, category.Description,
		category.IsActive).Scan(&category.ID, &category.CreatedAt)

	if err != nil {
		log.Printf("Ошибка создания категории: %v", err)
		return err
	}
	return nil
}

func (r *CategoryRepo) AllCategories() ([]models.Category, error) {
	query := `SELECT id, name, description, created_at, is_active
		FROM categories 
		WHERE is_active = true
		ORDER BY id`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var category models.Category
		err := rows.Scan(
			&category.ID, &category.Name, &category.Description,
			&category.CreatedAt, &category.IsActive,
		)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, nil
}

func (r *CategoryRepo) SearchCategory(query string) ([]models.Category, error) {
	searchQuery := `
	SELECT id, name, description, created_at, is_active
	FROM categories 	
	WHERE name ILIKE '%' || $1 || '%' 
	OR description ILIKE '%' || $1 || '%'  
	OR id::text = $1
	ORDER BY id`
	rows, err := r.db.Query(searchQuery, query)
	if err != nil {
		log.Panic("Ошибка: ", err)
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var category models.Category
		err := rows.Scan(
			&category.ID, &category.Name, &category.Description,
			&category.CreatedAt, &category.IsActive,
		)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, nil
}

func (r *CategoryRepo) UpdateCategory(category *models.Category) error {
	query := `
		update categories
		set name = $2, description = $3, is_active = $4
		WHERE id = $1`
	_, err := r.db.Exec(
		query, category.ID, category.Name, category.Description,
		category.IsActive,
	)
	if err != nil {
		log.Printf("Ошибка обновления категории: %v", err)
		return err
	}
	return nil
}

func (r *CategoryRepo) DeleteCategory(categoryID int) error {
	query := `
        WITH deleted_order_items AS (DELETE FROM order_items WHERE product_id = $1)
        DELETE FROM products WHERE id = $1`
	_, err := r.db.Exec(query, categoryID)
	if err != nil {
		log.Printf("Ошибка удаления категории: %v", err)
		return err
	}
	return nil
}

func (r *CategoryRepo) PaginateCategory(limit, offset int) ([]models.Category, error) {
	query := `
	SELECT id, name, description, created_at, is_active
        FROM categories
        WHERE is_active = true
        ORDER BY created_at ASC, id ASC
        LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var category models.Category
		err := rows.Scan(
			&category.ID, &category.Name, &category.Description,
			&category.CreatedAt, &category.IsActive,
		)
		if err != nil {
			log.Printf("Ошибка скана: %v", err)
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, nil
}

func (r *CategoryRepo) CountCategories() (int, error) { //подсчёт категорий для пагинации
	query := `SELECT COUNT(*) FROM categories WHERE is_active = true`
	var count int
	err := r.db.QueryRow(query).Scan(&count)
	return count, err
}
