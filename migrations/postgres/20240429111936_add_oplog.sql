-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS boo_operation_logs (
  id           bigserial PRIMARY KEY,
  userid       bigint REFERENCES boo_users(id) ON DELETE SET NULL,
  username     varchar(100),
  type         varchar(100),
  successful   boolean,
  content      text,
  fields       jsonb,
  created_at   timestamp with time zone
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS boo_operation_logs;
-- +goose StatementEnd
