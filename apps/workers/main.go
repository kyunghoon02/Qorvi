package main

import (
	"fmt"
	"log"

	"github.com/whalegraph/whalegraph/packages/config"
)

func main() {
	env, err := config.ParseWorkerEnvFromOS()
	if err != nil {
		log.Fatalf("worker env validation failed: %v", err)
	}

	fmt.Println(buildStartupMessage(env))
}

func buildStartupMessage(env config.WorkerEnv) string {
	return fmt.Sprintf(
		"WhaleGraph workers ready (env=%s, postgres=%s, redis=%s)",
		env.NodeEnv,
		env.PostgresURL,
		env.RedisURL,
	)
}
