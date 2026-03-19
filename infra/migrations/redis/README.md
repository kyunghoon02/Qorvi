# Redis Bootstrap Notes

Redis는 별도 마이그레이션이 없고, 다음 keyspace 규약만 고정한다.

- `dedup:*` - ingest idempotency key
- `rate-limit:*` - rate limit counters
- `queue:*` - job queue metadata
- `cache:*` - hot cache entries
