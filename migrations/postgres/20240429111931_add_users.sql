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
  deleted_at                  timestamp WITH TIME ZONE,
  created_at                  timestamp WITH TIME ZONE,
  updated_at                  timestamp WITH TIME ZONE
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS boo_user_profiles (
    user_id     bigint REFERENCES boo_users ON DELETE CASCADE,
    name        varchar(100) NOT NULL,
    value       text,
    created_at  TIMESTAMP WITH TIME ZONE,
    updated_at  TIMESTAMP WITH TIME ZONE,

    UNIQUE(user_id,name)
);
-- +goose StatementEnd


-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS boo_user_tags (
    id          bigserial PRIMARY KEY,
    uuid        varchar(100) NOT NULL,
    title       varchar(100) NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE,
    updated_at  TIMESTAMP WITH TIME ZONE,

    UNIQUE(uuid),
    UNIQUE(title)
);
-- +goose StatementEnd


-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS boo_user_to_tags (
    user_id     bigint REFERENCES boo_users ON DELETE CASCADE,
    tag_id      bigint REFERENCES boo_user_tags ON DELETE CASCADE,

    UNIQUE(user_id, tag_id)
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS boo_user_to_tags;
DROP TABLE IF EXISTS boo_user_profiles;
DROP TABLE IF EXISTS boo_users;
DROP TABLE IF EXISTS boo_user_tags;
