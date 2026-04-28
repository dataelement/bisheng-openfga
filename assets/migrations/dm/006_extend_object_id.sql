-- +goose Up
ALTER TABLE tuple MODIFY object_id VARCHAR(255);

-- +goose Down
ALTER TABLE tuple MODIFY object_id VARCHAR(128);
