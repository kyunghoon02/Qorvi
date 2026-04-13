CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_action_created_at
  ON audit_logs (actor_user_id, action, created_at DESC);
