package db

import "testing"

func TestRedisURLSharesAuthenticationDatabaseAndTLS(t *testing.T) {
	t.Setenv("REDIS_URL", "rediss://worker:secret@redis.example:6380/3")
	t.Setenv("REDIS_ADDR", "ignored:6379")
	options, err := redisOptions()
	if err != nil {
		t.Fatal(err)
	}
	if options.Addr != "redis.example:6380" || options.Username != "worker" || options.Password != "secret" || options.DB != 3 || options.TLSConfig == nil {
		t.Fatal("REDIS_URL must preserve server, authentication, database and TLS")
	}
}

func TestInvalidRedisURLDoesNotLeakCredentials(t *testing.T) {
	t.Setenv("REDIS_URL", "https://worker:secret@redis.example")
	_, err := redisOptions()
	if err == nil || err.Error() != "invalid REDIS_URL" {
		t.Fatal("invalid configuration must be rejected without exposing its value")
	}
}

func TestRedisAddressCompatibility(t *testing.T) {
	t.Setenv("REDIS_URL", "")
	t.Setenv("REDIS_ADDR", "127.0.0.1:6381")
	options, err := redisOptions()
	if err != nil || options.Addr != "127.0.0.1:6381" {
		t.Fatal("explicit legacy address must remain supported")
	}
}
