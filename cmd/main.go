package main

import (
	"fmt"
	"log"

	"github.com/go-redis/redis/v9"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/TemurMannonov/mediium_user_service/config"
	"github.com/TemurMannonov/mediium_user_service/storage"
)

func main() {
	cfg := config.Load(".")

	psqlUrl := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Postgres.Host,
		cfg.Postgres.Port,
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.Database,
	)

	psqlConn, err := sqlx.Connect("postgres", psqlUrl)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.Redis.Addr,
	})

	strg := storage.NewStoragePg(psqlConn)
	inMemory := storage.NewInMemoryStorage(rdb)

	strg = strg
	inMemory = inMemory

	log.Print("Server stopped")
}
