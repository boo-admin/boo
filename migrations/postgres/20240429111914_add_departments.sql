-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS boo_departments (
  id                          bigserial PRIMARY KEY,
  parent_id                   int NULL REFERENCES boo_departments(id) on delete set null,
  uuid                        VARCHAR(50),
  name                        VARCHAR(50),
  order_num                   int,
  fields                      jsonb,
  created_at                  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at                  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  unique(name)
);
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS boo_departments;
