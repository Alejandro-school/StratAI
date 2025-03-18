package db

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

var (
	Rdb *redis.Client
	Ctx = context.Background()
)

// initRedis inicializa la conexión a Redis
func InitRedis() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	Rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   0,
	})
	_, err := Rdb.Ping(Ctx).Result()
	if err != nil {
		log.Fatalf("No se pudo conectar a Redis: %v", err)
	}
	log.Println("Conectado a Redis en", redisAddr)
}
