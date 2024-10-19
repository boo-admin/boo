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
  deleted_at                  timestamp WITH TIME ZONE,
  created_at                  timestamp WITH TIME ZONE,
  updated_at                  timestamp WITH TIME ZONE
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS boo_employee_tags (
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
CREATE TABLE IF NOT EXISTS boo_employee_to_tags (
    employee_id     bigint REFERENCES boo_employees ON DELETE CASCADE,
    tag_id          bigint REFERENCES boo_employee_tags ON DELETE CASCADE,

    UNIQUE(employee_id, tag_id)
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS boo_employee_to_tags;
DROP TABLE IF EXISTS boo_employees;
DROP TABLE IF EXISTS boo_employee_tags;
