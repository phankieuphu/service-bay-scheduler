-- Links a customer profile to its identity-service user (§3a, §5a of
-- docs/architecture-design.md). Set by the identity.user-events consumer;
-- NULL for staff-created customers that have never signed up.

-- +goose Up
ALTER TABLE customer ADD COLUMN IF NOT EXISTS user_id uuid;
CREATE UNIQUE INDEX IF NOT EXISTS idx_customer_user_id ON customer (user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_customer_user_id;
ALTER TABLE customer DROP COLUMN IF EXISTS user_id;
