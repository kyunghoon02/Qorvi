package config

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	sharedconfig "github.com/qorvi/qorvi/packages/config"
	"github.com/qorvi/qorvi/packages/db"
)

type Config struct {
	Host                  string
	Port                  string
	API                   sharedconfig.APIEnv
	StorageHandles        db.Handles
	WalletSummaryCacheTTL time.Duration
}

func Load() (Config, error) {
	env, err := sharedconfig.ParseAPIEnvFromOS()
	if err != nil {
		return Config{}, err
	}

	redisConfig, err := parseRedisConfig(env.RedisURL)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Host: env.APIHost,
		Port: strconv.Itoa(env.APIPort),
		API:  env,
		StorageHandles: db.NewHandles(
			db.NewPostgresConfig(env.PostgresURL),
			db.NewNeo4jConfig(env.Neo4jURL, env.Neo4jUsername, env.Neo4jPassword),
			redisConfig,
		),
		WalletSummaryCacheTTL: 5 * time.Minute,
	}, nil
}

func parseRedisConfig(rawURL string) (db.RedisConfig, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return db.RedisConfig{}, fmt.Errorf("parse redis url: %w", err)
	}

	if parsed.Host == "" {
		return db.RedisConfig{}, fmt.Errorf("redis url must include host")
	}

	password, _ := parsed.User.Password()

	redisDB := 0
	if parsed.Path != "" && parsed.Path != "/" {
		value, err := strconv.Atoi(parsed.Path[1:])
		if err != nil {
			return db.RedisConfig{}, fmt.Errorf("redis url database must be an integer")
		}
		redisDB = value
	}

	return db.NewRedisConfig(normalizeRedisAddress(parsed), password, redisDB), nil
}

func normalizeRedisAddress(parsed *url.URL) string {
	if parsed == nil {
		return ""
	}

	if _, _, err := net.SplitHostPort(parsed.Host); err == nil {
		return parsed.Host
	}

	switch parsed.Scheme {
	case "rediss":
		return net.JoinHostPort(parsed.Hostname(), "6380")
	default:
		return net.JoinHostPort(parsed.Hostname(), "6379")
	}
}
