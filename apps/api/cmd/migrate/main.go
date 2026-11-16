package main

import (
	"context"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/database"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
	"log"
	"os"
)

func main() {
	url, err := config.LoadDatabaseURL()
	if err != nil {
		log.Fatal("database configuration error: ", err)
	}
	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		dir = "db/migrations"
	}
	ctx := context.Background()
	if os.Getenv("APP_ENV") == "local" && os.Getenv("MSSQL_AUTO_CREATE_DATABASE") == "true" {
		if err := database.EnsureLocalDatabase(ctx, url); err != nil {
			log.Fatal(err)
		}
	}
	db, err := database.Connect(ctx, url)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Apply(ctx, db, dir); err != nil {
		log.Fatal(err)
	}
	log.Print("migrations applied")
}
