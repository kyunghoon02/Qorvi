# Ops Admin Runbook

상세 production operator checklist는 [/Users/kh/Github/Qorvi/docs/runbooks/admin-operations-checklist.md](/Users/kh/Github/Qorvi/docs/runbooks/admin-operations-checklist.md)를 따른다. 이 문서는 그중 최소 운영 원칙만 요약한다.

## Purpose

This runbook covers the minimal operations flows for Qorvi admin work: labels, suppression, provider quota monitoring, and audit review.

## Core Rules

1. Every manual override must create an audit event.
2. Suppression should be the default response to a verified false positive.
3. Label changes must be reversible.
4. Provider quota warnings should be reviewed before the window is exhausted.

## Common Tasks

### Update a label

1. Confirm the label name, description, and color.
2. Apply the change through the admin surface.
3. Record who made the change and why.
4. Verify the change appears in the audit log.

### Add a suppression rule

1. Confirm scope and target.
2. Record the reason and the actor.
3. Set an expiration if the suppression should be temporary.
4. Verify downstream alerts or scores stop propagating as expected.

### Review provider quota

1. Check the latest provider snapshot.
2. Compare `used + reserved` against the limit.
3. Escalate when usage reaches warning or critical thresholds.
4. Pause non-essential enrichment before the quota is exhausted.

## Incident Triage

1. Identify whether the issue is labeling, suppression, or provider quota related.
2. Capture the affected wallet, cluster, token, or provider.
3. Apply the smallest reversible fix.
4. Log the action immediately.
5. Recheck the downstream signals after propagation.

## Audit Review

1. Filter by actor, action, and time window.
2. Confirm every suppression and label mutation has a matching audit record.
3. Escalate gaps in audit coverage before release.
