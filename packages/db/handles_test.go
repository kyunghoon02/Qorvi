package db

import "testing"

func TestNewHandlesAndValidation(t *testing.T) {
	t.Parallel()

	handles := NewHandles(
		NewPostgresConfig("postgres://postgres:postgres@localhost:5432/whalegraph"),
		NewNeo4jConfig("bolt://localhost:7687", "neo4j", "neo4jpassword"),
		NewRedisConfig("localhost:6379", "", 0),
	)

	if err := ValidateHandles(handles); err != nil {
		t.Fatalf("expected valid handles, got %v", err)
	}
}

func TestSplitResponsibilities(t *testing.T) {
	t.Parallel()

	split := SplitResponsibilities()

	if split.Graph != "neo4j" {
		t.Fatalf("expected graph store to be neo4j, got %q", split.Graph)
	}
	if split.Relational != "postgres" {
		t.Fatalf("expected relational store to be postgres, got %q", split.Relational)
	}
}
