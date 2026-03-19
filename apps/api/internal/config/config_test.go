package config

import "testing"

func TestParseRedisConfig(t *testing.T) {
	t.Parallel()

	cfg, err := parseRedisConfig("redis://:secret@cache.internal:6381/2")
	if err != nil {
		t.Fatalf("expected redis config, got %v", err)
	}

	if cfg.Address != "cache.internal:6381" {
		t.Fatalf("unexpected redis address %q", cfg.Address)
	}
	if cfg.Password != "secret" {
		t.Fatalf("unexpected redis password %q", cfg.Password)
	}
	if cfg.DB != 2 {
		t.Fatalf("unexpected redis db %d", cfg.DB)
	}
}

func TestParseRedisConfigDefaultsPortAndDB(t *testing.T) {
	t.Parallel()

	cfg, err := parseRedisConfig("redis://cache.internal")
	if err != nil {
		t.Fatalf("expected redis config, got %v", err)
	}

	if cfg.Address != "cache.internal:6379" {
		t.Fatalf("unexpected redis address %q", cfg.Address)
	}
	if cfg.DB != 0 {
		t.Fatalf("unexpected redis db %d", cfg.DB)
	}
}
