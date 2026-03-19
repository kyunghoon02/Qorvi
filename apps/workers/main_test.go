package main

import (
	"strings"
	"testing"

	"github.com/whalegraph/whalegraph/packages/config"
)

func TestBuildStartupMessage(t *testing.T) {
	t.Parallel()

	message := buildStartupMessage(config.WorkerEnv{
		NodeEnv:     "development",
		PostgresURL: "postgres://postgres:postgres@localhost:5432/whalegraph",
		RedisURL:    "redis://localhost:6379",
	})

	if !strings.Contains(message, "WhaleGraph workers ready") {
		t.Fatalf("unexpected startup message %q", message)
	}
}
