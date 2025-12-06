package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"project/internal/config"
	"project/internal/db"

	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Panic("Ошибка конфига", err)
	}

	db, err := db.NewPostgresDB(cfg)
	if err != nil {
		log.Panic("Ошибка подключения к PG4", err)
	}
	defer db.Close()

	projectRoot, err := getProjectRoot()
	if err != nil {
		log.Panic("Ошибка в корневой папке проекта", err)
	}

	migrations := []string{
		"001_create_categories.sql",
		"002_create_products.sql",
		"003_create_users.sql",
		"004_create_orders.sql",
		"100_data.sql",
	}

	successes := 0
	for _, migration := range migrations {
		migrationPath := filepath.Join(projectRoot, "migrations", migration)
		if err := Migrations(db, migrationPath); err != nil {
			log.Printf("Error: %s in migration: %v", err, migration)
		} else {
			log.Printf("Succesfull migration: %s", migration)
			successes++
		}
	}
	log.Printf("Succesfull migrations %d from %d", successes, len(migrations))
}

func Migrations(db *sql.DB, filepath string) error {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	_, err = db.Exec(string(content))
	return err
}

func getProjectRoot() (string, error) {
	// Ищем корень проекта по наличию go.mod
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd, nil
		}

		parent := filepath.Dir(wd)
		if parent == wd {
			return "", os.ErrNotExist
		}
		wd = parent
	}
}
