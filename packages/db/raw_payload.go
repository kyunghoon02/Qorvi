package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type RawPayloadDescriptor struct {
	Provider    string
	Operation   string
	ContentType string
	ObjectKey   string
	SHA256      string
	ObservedAt  time.Time
}

type RawPayloadStore interface {
	StoreRawPayload(context.Context, RawPayloadDescriptor, []byte) error
}

func BuildRawPayloadObjectKey(provider, operation string, observedAt time.Time, identifier string) string {
	parts := []string{
		strings.ToLower(strings.TrimSpace(provider)),
		strings.ReplaceAll(strings.ToLower(strings.TrimSpace(operation)), " ", "-"),
		observedAt.UTC().Format("2006/01/02"),
		strings.TrimSpace(identifier),
	}

	return strings.Join(parts, "/")
}

func NormalizeRawPayloadDescriptor(descriptor RawPayloadDescriptor) (RawPayloadDescriptor, error) {
	descriptor.Provider = strings.TrimSpace(descriptor.Provider)
	descriptor.Operation = strings.TrimSpace(descriptor.Operation)
	descriptor.ContentType = strings.TrimSpace(descriptor.ContentType)
	descriptor.ObjectKey = strings.TrimSpace(descriptor.ObjectKey)
	descriptor.SHA256 = strings.TrimSpace(descriptor.SHA256)

	if descriptor.Provider == "" {
		return RawPayloadDescriptor{}, fmt.Errorf("provider is required")
	}
	if descriptor.Operation == "" {
		return RawPayloadDescriptor{}, fmt.Errorf("operation is required")
	}
	if descriptor.ContentType == "" {
		return RawPayloadDescriptor{}, fmt.Errorf("content type is required")
	}
	if descriptor.ObjectKey == "" {
		return RawPayloadDescriptor{}, fmt.Errorf("object key is required")
	}
	if descriptor.ObservedAt.IsZero() {
		return RawPayloadDescriptor{}, fmt.Errorf("observed_at is required")
	}

	return descriptor, nil
}

func RawPayloadSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
