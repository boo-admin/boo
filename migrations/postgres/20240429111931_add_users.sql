-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS boo_users (
  id                          bigserial PRIMARY KEY,
  department_id               bigint NULL REFERENCES boo_departments(id) on delete set null,
  name                        varchar(100) NOT NULL UNIQUE,
  nickname                    varchar(100) NOT NULL UNIQUE,
  password                    varchar(500) ,
  last_password_modified_at   timestamp WITH TIME ZONE,
  description                 text,
  disabled                    boolean,
  source                      varchar(50),
  fields                      jsonb,
  created_at                  timestamp WITH TIME ZONE,
  updated_at                  timestamp WITH TIME ZONE
);

CREATE TABLE IF NOT EXISTS boo_user_profiles (
    id          bigserial PRIMARY KEY,
    user_id     bigint REFERENCES boo_users ON DELETE CASCADE,
    name        varchar(100) NOT NULL,
    value       text,
    created_at  TIMESTAMP WITH TIME ZONE,
    updated_at  TIMESTAMP WITH TIME ZONE,

    UNIQUE(user_id,name)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS boo_user_profiles;
DROP TABLE IF EXISTS boo_users;
-- +goose StatementEnd
