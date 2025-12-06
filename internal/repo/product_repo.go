package repo

import (
	"database/sql"
	"log"
	"project/internal/models"
	"strconv"
)

type ProductRepo struct {
	db *sql.DB
}

func NewProductRepo(db *sql.DB) *ProductRepo {

	return &ProductRepo{db: db}
}

func (r *ProductRepo) CreateProduct(product *models.Product) error {
	query := `
		INSERT INTO products (name, description, price, quantity, category_id, 
			weight, flavor, brand, servings, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`
	err := r.db.QueryRow(
		query, product.Name, product.Description,
		product.Price, product.Quantity, product.Category_id,
		product.Weight, product.Flavor, product.Brand,
		product.Servings, product.IsActive).Scan(&product.ID, &product.CreatedAt)

	if err != nil {
		log.Printf("Ошибка создания товара: %v", err)
		return err
	}
	return nil
}

func (r *ProductRepo) AllProducts() ([]models.Product, error) {
	query := `SELECT id, name, description, price, quantity, category_id, 
        weight, flavor, brand, servings, is_active, created_at
        FROM products 
        WHERE is_active = true
        ORDER BY id`

	rows, err := r.db.Query(query) //query для SELECT
	if err != nil {
		log.Printf("Ошибка скана: %v", err)
		return nil, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() { //идет по строкам и добавляет данные пока они есть. аналог while data
		var product models.Product
		err := rows.Scan(
			&product.ID, &product.Name, &product.Description, &product.Price, &product.Quantity,
			&product.Category_id, &product.Weight, &product.Flavor, &product.Brand, &product.Servings,
			&product.IsActive, &product.CreatedAt,
		)
		if err != nil {
			log.Printf("Ошибка скана: %v", err)
			return nil, err
		}
		products = append(products, product)
	}
	return products, nil
}

func (r *ProductRepo) ProductsByCategory(category interface{}) ([]models.Product, error) {
	query := `
        SELECT products.id, products.name, products.description, products.price, products.quantity, products.category_id, 
               products.weight, products.flavor, products.brand, products.servings, products.is_active, products.created_at
        FROM products 
        JOIN categories ON products.category_id = categories.id
        WHERE products.is_active = true 
          AND (products.category_id::text ILIKE $1 OR categories.name ILIKE '%' || $1 || '%')
        ORDER BY products.id`

	rows, err := r.db.Query(query, category) //query для SELECT
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() { //идет по строкам и добавляет данные пока они есть. аналог while data
		var product models.Product
		err := rows.Scan(
			&product.ID, &product.Name, &product.Description, &product.Price, &product.Quantity,
			&product.Category_id, &product.Weight, &product.Flavor, &product.Brand, &product.Servings,
			&product.IsActive, &product.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	return products, nil
}

func (r *ProductRepo) SearchProduct(query string) ([]models.Product, error) {
	searchQuery := `
	SELECT id, name, description, price, quantity, category_id, 
		weight, flavor, brand, servings, is_active, created_at
	FROM products 	
	WHERE name ILIKE '%' || $1 || '%' 
	OR description ILIKE '%' || $1 || '%' 
	OR flavor ILIKE '%' || $1 || '%' 
	OR weight ILIKE $1 
	OR id::text ILIKE $1
	ORDER BY id`
	rows, err := r.db.Query(searchQuery, query) //query для SELECT
	if err != nil {
		log.Printf("Ошибка: %v", err)
		return nil, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() { //идет по строкам и добавляет данные пока они есть. аналог while data
		var product models.Product
		err := rows.Scan(
			&product.ID, &product.Name, &product.Description, &product.Price, &product.Quantity,
			&product.Category_id, &product.Weight, &product.Flavor, &product.Brand, &product.Servings,
			&product.IsActive, &product.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	return products, nil
}

func (r *ProductRepo) UpdateProduct(product *models.Product) error {
	query := `
		update products 
		set name = $2, description = $3, price = $4, quantity = $5,
		category_id = $6, weight = $7, flavor = $8, brand = $9, 
		servings = $10, is_active = $11
		WHERE id = $1`
	_, err := r.db.Exec( //Exec для INSERT/UPDATE/DELETE
		query, product.ID, product.Name, product.Description,
		product.Price, product.Quantity, product.Category_id,
		product.Weight, product.Flavor, product.Brand,
		product.Servings, product.IsActive,
	)
	if err != nil {
		log.Printf("Ошибка обновления товара: %v", err)
		return err
	}
	return nil
}

func (r *ProductRepo) DeleteProduct(productID int) error {
	query := `
        WITH deleted_order_items AS (DELETE FROM order_items WHERE product_id = $1)
        DELETE FROM products WHERE id = $1`
	_, err := r.db.Exec(query, productID) //Exec для INSERT/UPDATE/DELETE
	if err != nil {
		log.Printf("Ошибка удаления товара: %v", err)
		return err
	}
	return nil
}

func (r *ProductRepo) PaginateProduct(limit, offset int) ([]models.Product, error) {
	query := `
        SELECT id, name, description, price, quantity, category_id, weight, flavor, servings, is_active, created_at
        FROM products
        WHERE is_active = true
        ORDER BY created_at ASC, id ASC
        LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(query, limit, offset) //query для SELECT
	if err != nil {
		return nil, err
	}
	defer rows.Close() //предотвращает утечку соединений

	var products []models.Product
	for rows.Next() { //идет по строкам и добавляет данные пока они есть. аналог while data
		var product models.Product
		err := rows.Scan(
			&product.ID, &product.Name, &product.Description, &product.Price, &product.Quantity,
			&product.Category_id, &product.Weight, &product.Flavor, &product.Servings,
			&product.IsActive, &product.CreatedAt,
		)
		if err != nil {
			log.Printf("Ошибка скана: %v", err)
			return nil, err
		}
		products = append(products, product)
	}
	return products, nil
}

func (r *ProductRepo) CountProducts() (int, error) { //подсчёт продуктов для пагинации
	query := `SELECT COUNT(*) FROM products WHERE is_active = true`
	var count int
	err := r.db.QueryRow(query).Scan(&count) //query для SELECT с 1 строкой
	return count, err
}

func (r *ProductRepo) PaginateProductsByCategory(categoryID string, limit, offset int) ([]models.Product, error) {
	id, err := strconv.Atoi(categoryID)
	if err != nil {
		return nil, err
	}
	query := `
        SELECT id, name, description, price, quantity, category_id, weight, flavor, servings, is_active, created_at
        FROM products
        WHERE is_active = true AND category_id = $1
        ORDER BY created_at ASC, id ASC
        LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(query, id, limit, offset) //query для SELECT
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var products []models.Product
	for rows.Next() { //идет по строкам и добавляет данные пока они есть. аналог while data
		var product models.Product
		err := rows.Scan(
			&product.ID, &product.Name, &product.Description, &product.Price, &product.Quantity,
			&product.Category_id, &product.Weight, &product.Flavor, &product.Servings,
			&product.IsActive, &product.CreatedAt,
		)
		if err != nil {
			log.Printf("Ошибка скана: %v", err)
			return nil, err
		}
		products = append(products, product)
	}
	return products, nil
}

func (r *ProductRepo) CountProductsByCategory(categoryID string) (int, error) {
	id, err := strconv.Atoi(categoryID)
	if err != nil {
		return 0, err
	}
	var count int
	query := `SELECT COUNT(*) FROM products WHERE is_active = true AND category_id = $1`
	err = r.db.QueryRow(query, id).Scan(&count) //query для SELECT с одной строкой
	return count, err
}
