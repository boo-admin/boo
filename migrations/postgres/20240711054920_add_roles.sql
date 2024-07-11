-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS boo_roles (
  id                          bigserial PRIMARY KEY,
  name                        VARCHAR(50),
  description                 VARCHAR(250),
  created_at                  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at                  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  unique(name)
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS boo_roles;

