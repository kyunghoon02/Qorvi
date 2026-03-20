package main

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/whalegraph/whalegraph/packages/config"
	"github.com/whalegraph/whalegraph/packages/db"
)

func openWorkerStorageClients(ctx context.Context, env config.WorkerEnv) (*db.StorageClients, error) {
	redisConfig, err := parseWorkerRedisConfig(env.RedisURL)
	if err != nil {
		return nil, err
	}

	return db.OpenStorageClients(ctx, db.NewHandles(
		db.NewPostgresConfig(env.PostgresURL),
		db.NewNeo4jConfig(env.Neo4jURL, env.Neo4jUsername, env.Neo4jPassword),
		redisConfig,
	))
}

func parseWorkerRedisConfig(rawURL string) (db.RedisConfig, error) {
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

	return db.NewRedisConfig(normalizeWorkerRedisAddress(parsed), password, redisDB), nil
}

func normalizeWorkerRedisAddress(parsed *url.URL) string {
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
