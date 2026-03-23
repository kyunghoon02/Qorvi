package providers

import (
	"os"
	"path/filepath"
	"testing"
)

func readProviderFixture(t *testing.T, segments ...string) string {
	t.Helper()

	path := filepath.Join(append([]string{"testdata"}, segments...)...)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) returned error: %v", path, err)
	}

	return string(body)
}
