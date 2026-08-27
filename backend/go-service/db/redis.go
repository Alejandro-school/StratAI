package db

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"cs2-demo-service/models"

	"github.com/redis/go-redis/v9"
)

var (
	Rdb *redis.Client
	Ctx = context.Background()
)

// MatchDataSnapshot is the exact Redis state needed to compensate a failed
// cross-store publication. TTL is preserved when the key existed.
type MatchDataSnapshot struct {
	Exists bool
	Value  string
	TTL    time.Duration
}

func matchDataKey(matchID string) string {
	namespace := os.Getenv("PIPELINE_NAMESPACE")
	if namespace == "" {
		namespace = "stratai:v2"
	}
	return fmt.Sprintf("%s:match-data:%s", namespace, matchID)
}

// initRedis inicializa la conexión a Redis
func InitRedis() error {
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
		return fmt.Errorf("could not connect to Redis: %w", err)
	}
	return nil
}

// SaveMatchData guarda los datos del match en Redis
func SaveMatchData(matchID string, matchData *models.MatchData) error {
	return SaveMatchDataContext(Ctx, matchID, matchData)
}

// SaveMatchDataContext stores match data with the caller's bounded context.
func SaveMatchDataContext(ctx context.Context, matchID string, matchData *models.MatchData) error {
	if Rdb == nil {
		return fmt.Errorf("redis client not initialized")
	}

	data, err := json.Marshal(matchData)
	if err != nil {
		return fmt.Errorf("failed to marshal match data: %w", err)
	}

	key := matchDataKey(matchID)
	// Guardar con expiración de 30 días (o lo que sea apropiado)
	err = Rdb.Set(ctx, key, data, 30*24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to save match data to redis: %w", err)
	}

	return nil
}

// SnapshotMatchData captures the previous value before the filesystem commit.
func SnapshotMatchData(ctx context.Context, matchID string) (MatchDataSnapshot, error) {
	if Rdb == nil {
		return MatchDataSnapshot{}, fmt.Errorf("redis client not initialized")
	}
	key := matchDataKey(matchID)
	value, err := Rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return MatchDataSnapshot{Exists: false}, nil
	}
	if err != nil {
		return MatchDataSnapshot{}, fmt.Errorf("snapshot match data: %w", err)
	}
	ttl, err := Rdb.PTTL(ctx, key).Result()
	if err != nil {
		return MatchDataSnapshot{}, fmt.Errorf("snapshot match data TTL: %w", err)
	}
	return MatchDataSnapshot{Exists: true, Value: value, TTL: ttl}, nil
}

// RestoreMatchData compensates a failed publication without inventing a prior
// value. A non-expiring key keeps that property; an expired TTL is restored
// with the service's normal retention rather than published indefinitely.
func RestoreMatchData(ctx context.Context, matchID string, snapshot MatchDataSnapshot) error {
	if Rdb == nil {
		return fmt.Errorf("redis client not initialized")
	}
	key := matchDataKey(matchID)
	if !snapshot.Exists {
		if err := Rdb.Del(ctx, key).Err(); err != nil {
			return fmt.Errorf("remove compensated match data: %w", err)
		}
		return nil
	}
	ttl := snapshot.TTL
	if ttl == -1 {
		ttl = 0
	} else if ttl <= 0 {
		ttl = 30 * 24 * time.Hour
	}
	if err := Rdb.Set(ctx, key, snapshot.Value, ttl).Err(); err != nil {
		return fmt.Errorf("restore compensated match data: %w", err)
	}
	return nil
}

// GetMatchData obtiene los datos del match desde Redis
func GetMatchData(matchID string) (*models.MatchData, error) {
	if Rdb == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	key := matchDataKey(matchID)
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
