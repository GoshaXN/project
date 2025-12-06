package models

import "time"

type Product struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Quantity    int       `json:"quantity"`
	Category_id int       `json:"cateogry_id"`
	Weight      float64   `json:"weight"`
	Flavor      string    `json:"flavor"`
	Brand       string    `json:"brand"`
	Servings    int       `json:"servings"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}
