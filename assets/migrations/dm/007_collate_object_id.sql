-- +goose Up
CREATE INDEX idx_user_lookup ON tuple (store, _user, relation, object_type, object_id);
DROP INDEX idx_reverse_lookup_user;

-- +goose Down
DROP INDEX idx_user_lookup;
CREATE INDEX idx_reverse_lookup_user ON tuple (store, object_type, relation, _user);
