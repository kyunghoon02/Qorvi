package db

type PostgresConfig struct {
	DSN string
}

type Neo4jConfig struct {
	URI      string
	Username string
	Password string
}

type RedisConfig struct {
	Address  string
	Password string
	DB       int
}

type Handles struct {
	Postgres PostgresConfig
	Neo4j    Neo4jConfig
	Redis    RedisConfig
}

type SplitRoles struct {
	Relational string
	Graph      string
	Cache      string
}
