module github.com/flowintel/flowintel/apps/api

go 1.24.4

require (
	github.com/flowintel/flowintel/packages/config v0.0.0
	github.com/flowintel/flowintel/packages/db v0.0.0
	github.com/flowintel/flowintel/packages/domain v0.0.0
	github.com/flowintel/flowintel/packages/intelligence v0.0.0
	github.com/flowintel/flowintel/packages/providers v0.0.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.7.6 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/neo4j/neo4j-go-driver/v5 v5.28.2 // indirect
	github.com/redis/go-redis/v9 v9.12.1 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/text v0.24.0 // indirect
)

replace github.com/flowintel/flowintel/packages/config => ../../packages/config

replace github.com/flowintel/flowintel/packages/db => ../../packages/db

replace github.com/flowintel/flowintel/packages/domain => ../../packages/domain

replace github.com/flowintel/flowintel/packages/intelligence => ../../packages/intelligence

replace github.com/flowintel/flowintel/packages/providers => ../../packages/providers
