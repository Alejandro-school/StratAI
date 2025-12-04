package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"cs2-demo-service/models"

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

// SaveMatchData guarda los datos del match en Redis
func SaveMatchData(matchID string, matchData *models.MatchData) error {
	if Rdb == nil {
		return fmt.Errorf("redis client not initialized")
	}

	data, err := json.Marshal(matchData)
	if err != nil {
		return fmt.Errorf("failed to marshal match data: %w", err)
	}

	key := fmt.Sprintf("match_data:%s", matchID)
	// Guardar con expiración de 30 días (o lo que sea apropiado)
	err = Rdb.Set(Ctx, key, data, 30*24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to save match data to redis: %w", err)
	}

	return nil
}

// GetMatchData obtiene los datos del match desde Redis
func GetMatchData(matchID string) (*models.MatchData, error) {
	if Rdb == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	key := fmt.Sprintf("match_data:%s", matchID)
	data, err := Rdb.Get(Ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get match data from redis: %w", err)
	}

	var matchData models.MatchData
	err = json.Unmarshal([]byte(data), &matchData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal match data: %w", err)
	}

	return &matchData, nil
}
