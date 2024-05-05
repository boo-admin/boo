-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS boo_departments (
  id                          bigserial PRIMARY KEY,
  parent_id                   int NULL REFERENCES boo_departments(id) on delete set null,
  name                        VARCHAR(50),
  order_num                   int,
  created_at                  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at                  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  unique(name)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS boo_departments;
-- +goose StatementEnd
