-- +goose Up
ALTER TABLE tuple ADD condition_name VARCHAR(256);
ALTER TABLE tuple ADD condition_context BLOB;
ALTER TABLE changelog ADD condition_name VARCHAR(256);
ALTER TABLE changelog ADD condition_context BLOB;

-- +goose Down
ALTER TABLE tuple DROP condition_name;
ALTER TABLE tuple DROP condition_context;
ALTER TABLE changelog DROP condition_name;
ALTER TABLE changelog DROP condition_context;
