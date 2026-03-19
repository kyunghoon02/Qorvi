package ops

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

func BuildAuditEvent(action AuditAction, actor, target, note string) (AuditEvent, error) {
	event := AuditEvent{
		ID:        fmt.Sprintf("audit_%d", time.Now().UTC().UnixNano()),
		Action:    action,
		Actor:     strings.TrimSpace(actor),
		Target:    strings.TrimSpace(target),
		Note:      strings.TrimSpace(note),
		CreatedAt: time.Now().UTC(),
	}
	if err := ValidateAuditEvent(event); err != nil {
		return AuditEvent{}, err
	}
	return event, nil
}

func ValidateAuditEvent(event AuditEvent) error {
	if event.Action == "" {
		return errors.New("audit action is required")
	}
	if strings.TrimSpace(event.Actor) == "" {
		return errors.New("audit actor is required")
	}
	if strings.TrimSpace(event.Target) == "" {
		return errors.New("audit target is required")
	}
	if event.CreatedAt.IsZero() {
		return errors.New("audit timestamp is required")
	}
	return nil
}
