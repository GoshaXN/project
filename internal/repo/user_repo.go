package repo

import (
	"database/sql"
	"fmt"
	"log"
	"project/internal/models"
	"project/internal/utils"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {

	return &UserRepo{db: db}
}

func (r *UserRepo) CreateUser(user *models.User, password ...string) error {
	var hashedPassoword string
	var err error
	hashedPassoword, err = utils.HashPassword(password[0])
	if err != nil {
		return fmt.Errorf("err in hash password: %v", err)
	}
	query := `
		INSERT INTO users (telegram_id, username, first_name, phone, email, role, password)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`
	err = r.db.QueryRow( //query для SELECT с 1 строкой
		query, user.TelegramID, user.Username,
		user.FirstName, user.Phone, user.Email,
		user.Role, hashedPassoword).Scan(&user.ID, &user.CreatedAt)

	if err != nil {
		log.Printf("Ошибка создания пользователя: %v", err)
		return err
	}
	return nil
}

func (r *UserRepo) SearchUserTGID(TelegramID int64) (*models.User, error) {
	searchQuery := `
        SELECT id, telegram_id, username, first_name, phone, email, role, password, created_at
        FROM users 
        WHERE telegram_id = $1
        LIMIT 1`

	rows, err := r.db.Query(searchQuery, TelegramID) //query для SELECT с 1 строкой
	if err != nil {
		return nil, fmt.Errorf("err of search: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("user not found")
	}
	var user models.User
	err = rows.Scan(
		&user.ID, &user.TelegramID, &user.Username, &user.FirstName,
		&user.Phone, &user.Email, &user.Role, &user.Password, &user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan error: %w", err)
	}

	return &user, nil
}
func (r *UserRepo) AllUsers() ([]models.User, error) {
	query := `
		SELECT id, telegram_id, username, first_name, phone, email, role, created_at
		FROM users 
		ORDER BY created_at DESC`

	rows, err := r.db.Query(query) //SELECT
	if err != nil {
		log.Printf("Ошибка скана: %v", err)
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() { //идет по строкам и добавляет данные пока они есть. аналог while data
		var user models.User
		err := rows.Scan(
			&user.ID, &user.TelegramID, &user.Username, &user.FirstName,
			&user.Phone, &user.Email, &user.Role, &user.CreatedAt,
		)
		if err != nil {
			log.Printf("Ошибка скана: %v", err)
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *UserRepo) SearchUser(query string) ([]models.User, error) {
	searchQuery := `
        SELECT id, telegram_id, username, first_name, phone, email, role, created_at
        FROM users 
        WHERE id::text = $1 
           OR telegram_id::text = $1
           OR username ILIKE '%' || $1 || '%' 
           OR first_name ILIKE '%' || $1 || '%'
        ORDER BY created_at DESC`

	rows, err := r.db.Query(searchQuery, query)
	if err != nil {
		log.Printf("Ошибка поиска пользователя: %v", err)
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() { //идет по строкам и добавляет данные пока они есть. аналог while data
		var user models.User
		err := rows.Scan(
			&user.ID, &user.TelegramID, &user.Username, &user.FirstName,
			&user.Phone, &user.Email, &user.Role, &user.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *UserRepo) UpdateUser(user *models.User, updatePassword ...bool) error {
	NeedUpdatePassword := len(updatePassword) > 0 && updatePassword[0] && user.Password != ""
	var password interface{}
	if NeedUpdatePassword {
		password = user.Password
	} else {
		password = nil
	}

	query := `
	UPDATE users
	SET telegram_id = $2, username = $3, first_name = $4,
		phone = $5, email = $6, role = $7,
		password = COALESCE($8, password)
	WHERE id = $1`

	args := []interface{}{
		user.ID, user.TelegramID, user.Username, user.FirstName,
		user.Phone, user.Email, user.Role, password,
	}

	_, err := r.db.Exec(query, args...)
	return err
}

func (r *UserRepo) UpdatePassword(userID int, NewPassword string) error {
	hashedPassowrd, err := utils.HashPassword(NewPassword)
	if err != nil {
		return fmt.Errorf("error with hash password: %v", err)
	}
	query := "UPDATE users SET password = $2 WHERE id = $1"
	_, err = r.db.Exec(query, userID, hashedPassowrd)
	if err != nil {
		return fmt.Errorf("error update password: %v", err)
	}
	return nil
}

func (r *UserRepo) DeleteUser(userID int) error {
	query := `WITH deleted_orders AS (DELETE FROM orders WHERE user_id = $1)
				DELETE FROM users WHERE id = $1`
	_, err := r.db.Exec(query, userID) // ISERT/UPDATE/DELETE
	if err != nil {
		log.Printf("Ошибка удаления пользователя: %v", err)
		return err
	}
	return nil
}

func (r *UserRepo) PaginateUsers(limit, offset int) ([]models.User, error) {
	query := `
        SELECT id, telegram_id, username, first_name, phone, email, role, created_at
        FROM users
        ORDER BY created_at ASC, id ASC
        LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(query, limit, offset) //query для SELECT
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() { //идет по строкам и добавляет данные пока они есть. аналог while data
		var user models.User
		err := rows.Scan(
			&user.ID, &user.TelegramID, &user.Username, &user.FirstName,
			&user.Phone, &user.Email, &user.Role, &user.CreatedAt,
		)
		if err != nil {
			log.Printf("Ошибка скана: %v", err)
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *UserRepo) CountUsers() (int, error) { //подсчёт юзеров для пагинации
	query := `SELECT COUNT(*) FROM users`
	var count int
	err := r.db.QueryRow(query).Scan(&count) //query для SELECT c 1 строкой
	return count, err
}
