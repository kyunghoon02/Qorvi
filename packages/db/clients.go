package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
)

type StorageClients struct {
	Postgres *pgxpool.Pool
	Neo4j    Neo4jDriver
	Redis    redis.UniversalClient
}

type Neo4jDriver interface {
	NewSession(context.Context, neo4j.SessionConfig) Neo4jSession
	VerifyConnectivity(context.Context) error
	Close(context.Context) error
}

type Neo4jSession interface {
	Run(context.Context, string, map[string]any, ...func(*neo4j.TransactionConfig)) (Neo4jResult, error)
	Close(context.Context) error
}

type Neo4jResult interface {
	Next(context.Context) bool
	Err() error
	Record() *neo4j.Record
}

type neo4jDriverAdapter struct {
	driver neo4j.DriverWithContext
}

type neo4jSessionAdapter struct {
	session neo4j.SessionWithContext
}

type neo4jResultAdapter struct {
	result neo4j.ResultWithContext
}

func OpenPostgresPool(ctx context.Context, cfg PostgresConfig) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}

func OpenNeo4jDriver(ctx context.Context, cfg Neo4jConfig) (Neo4jDriver, error) {
	driver, err := neo4j.NewDriverWithContext(
		cfg.URI,
		neo4j.BasicAuth(cfg.Username, cfg.Password, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("open neo4j driver: %w", err)
	}

	if err := driver.VerifyConnectivity(ctx); err != nil {
		_ = driver.Close(ctx)
		return nil, fmt.Errorf("verify neo4j connectivity: %w", err)
	}

	return neo4jDriverAdapter{driver: driver}, nil
}

func OpenRedisClient(ctx context.Context, cfg RedisConfig) (redis.UniversalClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}

func OpenStorageClients(ctx context.Context, handles Handles) (*StorageClients, error) {
	if err := ValidateHandles(handles); err != nil {
		return nil, err
	}

	postgres, err := OpenPostgresPool(ctx, handles.Postgres)
	if err != nil {
		return nil, err
	}

	neo4jDriver, err := OpenNeo4jDriver(ctx, handles.Neo4j)
	if err != nil {
		postgres.Close()
		return nil, err
	}

	redisClient, err := OpenRedisClient(ctx, handles.Redis)
	if err != nil {
		postgres.Close()
		_ = neo4jDriver.Close(ctx)
		return nil, err
	}

	return &StorageClients{
		Postgres: postgres,
		Neo4j:    neo4jDriver,
		Redis:    redisClient,
	}, nil
}

func (a neo4jDriverAdapter) NewSession(ctx context.Context, config neo4j.SessionConfig) Neo4jSession {
	return neo4jSessionAdapter{session: a.driver.NewSession(ctx, config)}
}

func (a neo4jDriverAdapter) VerifyConnectivity(ctx context.Context) error {
	return a.driver.VerifyConnectivity(ctx)
}

func (a neo4jDriverAdapter) Close(ctx context.Context) error {
	return a.driver.Close(ctx)
}

func (a neo4jSessionAdapter) Run(
	ctx context.Context,
	cypher string,
	params map[string]any,
	configurers ...func(*neo4j.TransactionConfig),
) (Neo4jResult, error) {
	result, err := a.session.Run(ctx, cypher, params, configurers...)
	if err != nil {
		return nil, err
	}

	return neo4jResultAdapter{result: result}, nil
}

func (a neo4jSessionAdapter) Close(ctx context.Context) error {
	return a.session.Close(ctx)
}

func (a neo4jResultAdapter) Next(ctx context.Context) bool {
	return a.result.Next(ctx)
}

func (a neo4jResultAdapter) Err() error {
	return a.result.Err()
}

func (a neo4jResultAdapter) Record() *neo4j.Record {
	return a.result.Record()
}

func (c *StorageClients) Close(ctx context.Context) error {
	if c == nil {
		return nil
	}

	var firstErr error
	if c.Redis != nil {
		if err := c.Redis.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if c.Neo4j != nil {
		if err := c.Neo4j.Close(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if c.Postgres != nil {
		c.Postgres.Close()
	}

	return firstErr
}
