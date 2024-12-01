//go:generate gogenv2 server -ext=.server-gen.go departments.go
//go:generate gogenv2 client -ext=.client-gen.go departments.go

package booclient

import (
	"context"
	"time"

	"github.com/runner-mei/resty"
)

type Department struct {
	TableName struct{}  `json:"-" xorm:"boo_departments"`
	ID        int64     `json:"id" xorm:"id pk autoincr"`
	ParentID  int64     `json:"parent_id" xorm:"parent_id null"`
	UUID      string    `json:"uuid" xorm:"uuid unique null"`
	Name      string    `json:"name" xorm:"name notnull"`
	OrderNum  int       `json:"order_num" xorm:"order_num null"`
	Fields    map[string]interface{}    `json:"fields" xorm:"fields null"`
	CreatedAt time.Time `json:"created_at,omitempty" xorm:"created_at created"`
	UpdatedAt time.Time `json:"updated_at,omitempty" xorm:"updated_at updated"`

	Children []*Department `json:"children" xorm:"-"`
}

type Departments interface {
	// @Summary 新建一个部门
	// @Param department     body Department    true     "部门定义"
	// @Accept  json
	// @Produce  json
	// @Router /departments [post]
	// @Success 200 {int64} int64  "成功时返回新建部门的ID"
	Create(ctx context.Context, department *Department) (int64, error)

	// @Summary 修改部门名称
	// @Param id            path int                       true     "部门ID"
	// @Param department     body Department    true     "部门定义"
	// @Accept  json
	// @Produce  json
	// @Router /departments/{id} [put]
	// @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
	UpdateByID(ctx context.Context, id int64, department *Department) error

	// @Summary 删除指定的部门
	// @Param   id            path int                       true     "部门ID"
	// @Accept  json
	// @Produce json
	// @Router  /departments/{id} [delete]
	// @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
	DeleteByID(ctx context.Context, id int64) error

	// @Summary 查询指定的部门
	// @Param id            path int                       true     "部门ID"
	// @Accept  json
	// @Produce json
	// @Router  /departments/{id} [get]
	// @Success 200 {object} Department  "返回指定的部门"
	FindByID(ctx context.Context, id int64) (*Department, error)

	// @Summary 按名称查询指定的部门
	// @Param   name            path string                       true     "部门名称"
	// @Accept  json
	// @Produce json
	// @Router  /departments/by_name/{name} [get]
	// @Success 200 {array} Department  "返回所有部门"
	FindByName(ctx context.Context, name string) (*Department, error)

	// @Summary 查询部门数目
	// @Accept  json
	// @Produce json
	// @Param    keyword       query string                   false     "查询参数"
	// @Router  /departments/count [get]
	// @Success 200 {int64} int64  "返回所有部门数目"
	Count(ctx context.Context, keyword string) (int64, error)

	// @Summary 查询所有部门
	// @Accept  json
	// @Produce json
	// @Param    keyword       query string                   false     "查询参数"
	// @Param    sort          query string                   false     "排序字段"
	// @Param    offset        query int                      false     "offset"
	// @Param    limit         query int                      false     "limit"
	// @Router  /departments [get]
	// @Success 200 {array} Department  "返回所有部门"
	List(ctx context.Context, keyword string, sort string, offset, limit int64) ([]Department, error)

	// @Summary 查询所有部门, 并将它转成 tree 形式返回
	// @Accept  json
	// @Produce json
	// @Router  /departments/tree [get]
	// @Success 200 {array} Department  "返回所有部门"
	GetTree(ctx context.Context) ([]*Department, error)
}

func NewRemoteDepartments(pxy *resty.Proxy) Departments {
	return DepartmentsClient{
		Proxy: pxy,
	}
}
