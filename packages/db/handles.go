package db

import "fmt"

func NewPostgresConfig(dsn string) PostgresConfig {
	return PostgresConfig{DSN: dsn}
}

func NewNeo4jConfig(uri, username, password string) Neo4jConfig {
	return Neo4jConfig{
		URI:      uri,
		Username: username,
		Password: password,
	}
}

func NewRedisConfig(address, password string, db int) RedisConfig {
	return RedisConfig{
		Address:  address,
		Password: password,
		DB:       db,
	}
}

func NewHandles(postgres PostgresConfig, neo4j Neo4jConfig, redis RedisConfig) Handles {
	return Handles{
		Postgres: postgres,
		Neo4j:    neo4j,
		Redis:    redis,
	}
}

func ValidateHandles(handles Handles) error {
	if handles.Postgres.DSN == "" {
		return fmt.Errorf("postgres dsn is required")
	}
	if handles.Neo4j.URI == "" {
		return fmt.Errorf("neo4j uri is required")
	}
	if handles.Neo4j.Username == "" {
		return fmt.Errorf("neo4j username is required")
	}
	if handles.Redis.Address == "" {
		return fmt.Errorf("redis address is required")
	}
	return nil
}

func SplitResponsibilities() SplitRoles {
	return SplitRoles{
		Relational: "postgres",
		Graph:      "neo4j",
		Cache:      "redis",
	}
}
