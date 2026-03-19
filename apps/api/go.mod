module github.com/whalegraph/whalegraph/apps/api

go 1.24.4

require (
	github.com/whalegraph/whalegraph/packages/config v0.0.0
	github.com/whalegraph/whalegraph/packages/db v0.0.0
	github.com/whalegraph/whalegraph/packages/domain v0.0.0
	github.com/whalegraph/whalegraph/packages/intelligence v0.0.0
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

replace github.com/whalegraph/whalegraph/packages/config => ../../packages/config

replace github.com/whalegraph/whalegraph/packages/db => ../../packages/db

replace github.com/whalegraph/whalegraph/packages/domain => ../../packages/domain

replace github.com/whalegraph/whalegraph/packages/intelligence => ../../packages/intelligence
