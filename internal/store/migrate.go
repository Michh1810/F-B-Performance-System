package store

import (
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func RunMigrations(databaseURL string) {
	m, err := migrate.New(
		"file://migrations",
		databaseURL,
	)

	if err != nil {
		log.Fatalf("migration setup failed: %v", err)
	}

	err = m.Up()

	if err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migration failed: %v", err)
	}

	log.Println("migrations completed")
}