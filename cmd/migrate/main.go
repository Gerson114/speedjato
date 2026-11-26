package main

import (
	"context"
	"github.com/joho/godotenv"
	database "lava/external/db"
	"log"
	"os"
	"time"
)

func main() {
	if os.Getenv("APP_ENV") != "production" {
		_ = godotenv.Load()
	}
	db, err := database.ConnectDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := database.Migrar(ctx, db); err != nil {
		log.Fatal(err)
	}
	log.Print("Migrações aplicadas.")
}
