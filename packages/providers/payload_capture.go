package providers

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

func buildProviderRawPayloadMetadata(
	provider ProviderName,
	operation string,
	observedAt time.Time,
	identifier string,
	raw []byte,
) map[string]any {
	objectKey := buildProviderRawPayloadObjectKey(provider, operation, observedAt, identifier)
	sum := sha256.Sum256(raw)

	return map[string]any{
		"raw_payload_object_key":   objectKey,
		"raw_payload_sha256":       hex.EncodeToString(sum[:]),
		"raw_payload_size_bytes":   len(raw),
		"raw_payload_content_type": "application/json",
		"raw_payload_body":         string(raw),
		"raw_payload_source":       "http-response",
		"raw_payload_provider":     string(provider),
		"raw_payload_operation":    operation,
	}
}

func buildProviderRawPayloadObjectKey(provider ProviderName, operation string, observedAt time.Time, identifier string) string {
	parts := []string{
		strings.ToLower(strings.TrimSpace(string(provider))),
		strings.ToLower(strings.TrimSpace(operation)),
		observedAt.UTC().Format("2006/01/02"),
		sanitizePayloadIdentifier(identifier),
	}

	return strings.Join(parts, "/")
}

func sanitizePayloadIdentifier(value string) string {
	replacer := strings.NewReplacer("/", "-", ":", "-", " ", "-")
	sanitized := replacer.Replace(strings.TrimSpace(value))
	if sanitized == "" {
		return "payload"
	}

	return sanitized
}

func mergeMetadata(base map[string]any, overlay map[string]any) map[string]any {
	merged := make(map[string]any, len(base)+len(overlay))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overlay {
		merged[key] = value
	}
	return merged
}

func cloneMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}

	return cloned
}

func capturePagePayloadMetadata(
	provider ProviderName,
	operation string,
	observedAt time.Time,
	identifier string,
	raw []byte,
	extra map[string]any,
) map[string]any {
	return mergeMetadata(extra, buildProviderRawPayloadMetadata(provider, operation, observedAt, identifier, raw))
}

func prefixMetadataKeys(metadata map[string]any, prefix string) map[string]any {
	if len(metadata) == 0 {
		return map[string]any{}
	}

	prefixed := make(map[string]any, len(metadata))
	for key, value := range metadata {
		prefixed[prefix+key] = value
	}

	return prefixed
}
