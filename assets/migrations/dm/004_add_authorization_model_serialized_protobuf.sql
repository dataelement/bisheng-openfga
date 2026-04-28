-- +goose Up
ALTER TABLE authorization_model ADD serialized_protobuf BLOB;

-- +goose Down
ALTER TABLE authorization_model DROP serialized_protobuf;
