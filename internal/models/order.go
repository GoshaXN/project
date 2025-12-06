package models

import "time"

type Order struct {
	ID        int       `json:"id"`
	UserID    int64     `json:"user_id"`
	Amount    float64   `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type OrderItem struct {
	ID        int     `json:"id"`
	OrderID   int     `json:"order_id"`
	ProductID int     `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type OrderWithItems struct { //В БД она не появится т к содержит абсолютно всю информацию из имеющихся данных: структуррирует данные
	Order Order       `json:"order"` // заказ
	Items []OrderItem `json:"items"` // позиции заказа
}
