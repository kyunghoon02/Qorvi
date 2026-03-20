package db

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FilesystemRawPayloadStore struct {
	RootDir string
	Now     func() time.Time
}

type filesystemRawPayloadMetadata struct {
	Descriptor RawPayloadDescriptor `json:"descriptor"`
	StoredAt   time.Time            `json:"stored_at"`
	ByteLength int                  `json:"byte_length"`
}

func NewFilesystemRawPayloadStore(rootDir string) *FilesystemRawPayloadStore {
	return &FilesystemRawPayloadStore{
		RootDir: strings.TrimSpace(rootDir),
		Now:     time.Now,
	}
}

func (s *FilesystemRawPayloadStore) StoreRawPayload(
	ctx context.Context,
	descriptor RawPayloadDescriptor,
	payload []byte,
) error {
	_ = ctx

	if s == nil {
		return fmt.Errorf("raw payload store is nil")
	}

	normalized, err := NormalizeRawPayloadDescriptor(descriptor)
	if err != nil {
		return err
	}
	objectKey, err := normalizeRawPayloadObjectKey(normalized.ObjectKey)
	if err != nil {
		return err
	}
	if normalized.SHA256 == "" {
		normalized.SHA256 = RawPayloadSHA256(payload)
	} else if normalized.SHA256 != RawPayloadSHA256(payload) {
		return fmt.Errorf("payload sha256 mismatch")
	}

	rootDir := strings.TrimSpace(s.RootDir)
	if rootDir == "" {
		return fmt.Errorf("root dir is required")
	}

	payloadPath := filepath.Join(rootDir, objectKey)
	metadataPath := payloadPath + ".meta.json"

	if err := os.MkdirAll(filepath.Dir(payloadPath), 0o755); err != nil {
		return fmt.Errorf("create payload directory: %w", err)
	}

	if err := writeAtomicFile(payloadPath, payload, 0o644); err != nil {
		return err
	}

	metadata := filesystemRawPayloadMetadata{
		Descriptor: normalized,
		StoredAt:   s.now().UTC(),
		ByteLength: len(payload),
	}
	rawMetadata, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal raw payload metadata: %w", err)
	}
	if err := writeAtomicFile(metadataPath, rawMetadata, 0o644); err != nil {
		return err
	}

	return nil
}

func (s *FilesystemRawPayloadStore) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}

	return time.Now()
}

func normalizeRawPayloadObjectKey(objectKey string) (string, error) {
	trimmed := strings.TrimSpace(objectKey)
	for _, segment := range strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '/' || r == '\\'
	}) {
		if segment == ".." {
			return "", fmt.Errorf("object key must not escape root")
		}
	}

	cleaned := filepath.Clean(filepath.FromSlash(trimmed))
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("object key is required")
	}
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("object key must be relative")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("object key must not escape root")
	}

	return cleaned, nil
}

func writeAtomicFile(path string, contents []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".rawpayload-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tempPath := temp.Name()

	defer func() {
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}()

	if _, err := temp.Write(contents); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := temp.Chmod(mode); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}
