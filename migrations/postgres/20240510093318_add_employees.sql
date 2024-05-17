-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS boo_employees (
  id                          bigserial PRIMARY KEY,
  department_id               bigint NULL REFERENCES boo_departments(id) on delete set null,
  user_id                     bigint NULL REFERENCES boo_users(id) on delete set null,
  name                        varchar(100) NOT NULL UNIQUE,
  nickname                    varchar(100) NOT NULL UNIQUE,
  description                 text,
  disabled                    boolean,
  source                      varchar(50),
  fields                      jsonb,
  created_at                  timestamp WITH TIME ZONE,
  updated_at                  timestamp WITH TIME ZONE
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS boo_employees;
