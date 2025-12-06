package models

import "time"

type User struct {
	ID         int64     `json:"id"`
	TelegramID int64     `json:"telegram_id"`
	Username   string    `json:"username"`
	FirstName  string    `json:"first_name"`
	Phone      string    `json:"phone"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
	CreatedAt  time.Time `json:"created_at"`
}
