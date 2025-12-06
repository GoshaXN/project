package repo

import (
	"database/sql"
	"log"
	"project/internal/models"
)

type OrderRepo struct {
	db *sql.DB
}

func NewOrderRepo(db *sql.DB) *OrderRepo {
	return &OrderRepo{db: db}
}

func (r *OrderRepo) AllOrders() ([]models.Order, error) {
	query := `
	SELECT orders.id, orders.user_id, orders.amount, orders.status, orders.created_at
	FROM orders 
		ORDER BY created_at DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		log.Printf("Ошибка скана: %v", err)
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(
			&order.ID, &order.UserID, &order.Amount, &order.Status, &order.CreatedAt,
		)
		if err != nil {
			log.Printf("Ошибка скана: %v", err)
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (r *OrderRepo) OrderItems(orderID int) ([]models.OrderItem, error) {
	query := `
		SELECT id, order_id, product_id, quantity, price
		FROM order_items 
		WHERE order_id = $1`

	rows, err := r.db.Query(query, orderID)
	if err != nil {
		log.Printf("Ошибка скана: %v", err)
		return nil, err
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		err := rows.Scan(
			&item.ID, &item.OrderID, &item.ProductID, &item.Quantity,
			&item.Price,
		)
		if err != nil {
			log.Printf("Ошибка скана: %v", err)
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *OrderRepo) CreateOrder(userID int64) (*models.Order, error) {
	query := `
        INSERT INTO orders (user_id, status) 
        VALUES ($1, 'new') 
        RETURNING id, user_id, amount, status, created_at`

	var order models.Order
	err := r.db.QueryRow(query, userID).Scan(
		&order.ID, &order.UserID, &order.Amount, &order.Status, &order.CreatedAt,
	)
	return &order, err
}

func (r *OrderRepo) UserOrder(userID int64) ([]models.Order, error) {
	query := `
        SELECT id, user_id, amount, status, created_at
        FROM orders 
        WHERE user_id = $1
        ORDER BY created_at DESC`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(
			&order.ID, &order.UserID, &order.Amount,
			&order.Status, &order.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, nil
}

func (r *OrderRepo) ConfirmOrder(userID int64) (int, error) {
	SearchQuery := `
        SELECT id 
        FROM orders 
        WHERE user_id = $1 AND status = 'new' 
        ORDER BY created_at DESC 
        LIMIT 1`

	var orderID int
	err := r.db.QueryRow(SearchQuery, userID).Scan(&orderID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("нет активных заказов (со статусом 'new')")
		}
		return 0, err
	}

	UpdateQuery := `
        UPDATE orders 
        SET status = 'confirmed' 
        WHERE id = $1`
	_, err = r.db.Exec(UpdateQuery, orderID)
	if err != nil {
		return 0, err
	}

	return orderID, nil
}

func (r *OrderRepo) DetailCart(userID int64) (*models.OrderWithItems, error) {
	query := `
        SELECT id, user_id, amount, status, created_at
        FROM orders 
        WHERE user_id = $1 AND status = 'new'
        ORDER BY id DESC 
        LIMIT 1`

	var order models.Order
	err := r.db.QueryRow(query, userID).Scan(
		&order.ID, &order.UserID, &order.Amount, &order.Status, &order.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Printf("Ошибка корзины: %v", err)
		return nil, err
	}

	items, err := r.OrderItems(order.ID)
	if err != nil {
		log.Printf("Ошибка получения товаров корзины: %v", err)
		return nil, err
	}

	OrderWithItems := &models.OrderWithItems{
		Order: order,
		Items: items,
	}

	return OrderWithItems, nil
}

func (r *OrderRepo) AddItemToCart(orderID, productID int, quantity int, price float64) error {
	query := `
        INSERT INTO order_items (order_id, product_id, quantity, price)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (order_id, product_id) 
        DO UPDATE SET quantity = order_items.quantity + $3`
	_, err := r.db.Exec(query, orderID, productID, quantity, price)
	return err
}
