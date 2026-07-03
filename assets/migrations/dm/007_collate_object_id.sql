-- +goose Up
CREATE INDEX IF NOT EXISTS idx_user_lookup ON tuple (store, _user, relation, object_type, object_id);
DROP INDEX IF EXISTS idx_reverse_lookup_user;

-- +goose Down
DROP INDEX IF EXISTS idx_user_lookup;
CREATE INDEX IF NOT EXISTS idx_reverse_lookup_user ON tuple (store, object_type, relation, _user);
