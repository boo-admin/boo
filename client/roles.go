//go:generate gogenv2 server -ext=.server-gen.go roles.go
//go:generate gogenv2 client -ext=.client-gen.go roles.go

package client

import (
  "context"
  "time"
)

type Role struct {
  TableName   struct{}  `json:"-" xorm:"boo_roles"`
  ID          int64     `json:"id" xorm:"id pk autoincr"`
  Name        string    `json:"name" xorm:"name unique notnull"`
  Description string    `json:"description,omitempty" xorm:"description clob null"`
  CreatedAt   time.Time `json:"created_at,omitempty" xorm:"created_at created"`
  UpdatedAt   time.Time `json:"updated_at,omitempty" xorm:"updated_at updated"`

  IsDefault   bool      `json:"is_default,omitempty" xorm:"-"`
}

type Roles interface {
  // @Summary 新建一个角色
  // @Param    role     body Role    true     "角色定义"
  // @Accept   json
  // @Produce  json
  // @Router   /roles [post]
  // @Success 200 {int64} int64  "成功时返回新建角色的ID"
  Create(ctx context.Context, role *Role) (int64, error)

  // @Summary 修改角色名称
  // @Param    id            path int                       true     "角色ID"
  // @Param    role     body Role    true     "角色定义"
  // @Accept   json
  // @Produce  json
  // @Router /roles/{id} [put]
  // @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
  UpdateByID(ctx context.Context, id int64, role *Role) error

  // @Summary 删除指定的角色
  // @Param   id            path int                       true     "角色ID"
  // @Accept  json
  // @Produce json
  // @Router  /roles/{id} [delete]
  // @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
  DeleteByID(ctx context.Context, id int64) error

  // @Summary 查询指定的角色
  // @Param   id            path int                       true     "角色ID"
  // @Accept  json
  // @Produce json
  // @Router  /roles/{id} [get]
  // @Success 200 {object} Role  "返回指定的角色"
  FindByID(ctx context.Context, id int64) (*Role, error)

  // @Summary 按名称查询指定的角色
  // @Param   name            path string                       true     "角色名称"
  // @Accept  json
  // @Produce json
  // @Router  /roles/by_name/{name} [get]
  // @Success 200 {array} Role  "返回所有角色"
  FindByName(ctx context.Context, name string) (*Role, error)

  // @Summary 查询角色数目
  // @Accept   json
  // @Produce  json
  // @Param    keyword       query string                   false     "查询参数"
  // @Router   /roles/count [get]
  // @Success 200 {int64} int64  "返回所有角色数目"
  Count(ctx context.Context, keyword string) (int64, error)

  // @Summary 查询所有角色
  // @Accept  json
  // @Produce json
  // @Param    keyword       query string                   false     "查询参数"
  // @Param    sort          query string                   false     "排序字段"
  // @Param    offset        query int                      false     "offset"
  // @Param    limit         query int                      false     "limit"
  // @Router  /roles [get]
  // @Success 200 {array} Role  "返回所有角色"
  List(ctx context.Context, keyword string, sort string, offset, limit int64) ([]Role, error)
}
