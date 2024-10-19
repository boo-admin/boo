-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS boo_user_roles (
  id                          bigserial PRIMARY KEY,
  uuid                        VARCHAR(50),
  title                       VARCHAR(250),
  description                 VARCHAR(250),
  created_at                  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at                  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  unique(uuid),
  unique(title)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS boo_user_to_roles (
    user_id          bigint REFERENCES boo_users ON DELETE CASCADE,
    role_id          bigint REFERENCES boo_user_roles ON DELETE CASCADE,

    UNIQUE(user_id, role_id)
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS boo_user_to_roles;
DROP TABLE IF EXISTS boo_user_roles;
