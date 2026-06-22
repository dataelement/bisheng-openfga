-- +goose Up
-- DaMeng (DM) optimizer bug: "ORDER BY ulid + LIMIT N" abandons existing indexes
-- and falls back to a full table scan. Appending ulid to the tail of each read
-- index lets the optimizer satisfy the ORDER BY directly from the index, avoiding
-- the post-fetch sort and full scan. See pkg/storage/dm/dm.go read() / ReadPage().

-- Read by object: WHERE store, object_type, object_id ORDER BY ulid LIMIT N (hot path).
CREATE INDEX idx_tuple_read_object ON tuple (store, object_type, object_id, ulid);

-- Read by user + relation: WHERE store, _user, relation ORDER BY ulid LIMIT N.
CREATE INDEX idx_tuple_read_user ON tuple (store, _user, relation, ulid);

-- Read by user only: WHERE store, _user ORDER BY ulid LIMIT N.
CREATE INDEX idx_tuple_read_user_only ON tuple (store, _user, ulid);

-- +goose Down
DROP INDEX idx_tuple_read_object;
DROP INDEX idx_tuple_read_user;
DROP INDEX idx_tuple_read_user_only;
