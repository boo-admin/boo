//go:generate gogenv2 server -ext=.server-gen.go employees.go
//go:generate gogenv2 client -ext=.client-gen.go employees.go

package client

import (
	"context"
	"time"

	"github.com/runner-mei/resty"
)

type Employee struct {
	TableName    struct{}               `json:"-" xorm:"boo_employees"`
	ID           int64                  `json:"id" xorm:"id pk autoincr"`
	DepartmentID int64                  `json:"department_id,omitempty" xorm:"department_id null"`
	UserID       int64                  `json:"user_id,omitempty" xorm:"user_id null"`
	Name         string                 `json:"name" xorm:"name unique notnull"`
	Nickname     string                 `json:"nickname" xorm:"nickname unique notnull"`
	Description  string                 `json:"description,omitempty" xorm:"description clob null"`
	Source       string                 `json:"source,omitempty" xorm:"source null"`
	Disabled     bool                   `json:"disabled,omitempty" xorm:"disabled null"`
	Fields       map[string]interface{} `json:"fields" xorm:"fields jsonb null"`
	CreatedAt    time.Time              `json:"created_at,omitempty" xorm:"created_at created"`
	UpdatedAt    time.Time              `json:"updated_at,omitempty" xorm:"updated_at updated"`

	Department *Department `json:"department,omitempty" xorm:"-"`
}

func (u *Employee) GetPhone() string {
	return u.GetString(Phone.ID)
}

func (u *Employee) GetEmail() string {
	return u.GetString(Email.ID)
}

func (u *Employee) GetString(key string) string {
	if u.Fields == nil {
		return ""
	}
	o := u.Fields[key]
	if o == nil {
		return ""
	}
	s, _ := o.(string)
	return s
}

type Employees interface {
	// @Summary 新建一个用户
	// @Param    employee     body Employee    true     "用户定义"
	// @Accept   json
	// @Produce  json
	// @Router /employees [post]
	// @Success 200 {int64} int64  "成功时返回新建用户的ID"
	Create(ctx context.Context, employee *Employee) (int64, error)

	// @Summary 修改用户名称
	// @Param    id      path int     true     "用户ID"
	// @Param    employee    body Employee    true     "用户信息"
	// @Accept   json
	// @Produce  json
	// @Router /employees/{id} [put]
	// @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
	UpdateByID(ctx context.Context, id int64, employee *Employee) error

	// @Summary 删除指定的用户
	// @Param   id            path int                       true     "用户ID"
	// @Accept  json
	// @Produce json
	// @Router  /employees/{id} [delete]
	// @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
	DeleteByID(ctx context.Context, id int64) error

	// @Summary 批量删除指定的用户
	// @Param   id            query int64                       true     "用户ID"
	// @Accept  json
	// @Produce json
	// @Router  /employees/batch [delete]
	// @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
	DeleteBatch(ctx context.Context, id []int64) error

	// @Summary 查询指定的用户
	// @Param id            path int                       true     "用户ID"
	// @Accept  json
	// @Produce json
	// @Router  /employees/{id} [get]
	// @Success 200 {object} Employee  "返回指定的用户"
	FindByID(ctx context.Context, id int64) (*Employee, error)

	// @Summary 按名称查询指定的用户
	// @Param   name            path string                       true     "用户名"
	// @Accept  json
	// @Produce json
	// @Router  /employees/by_name/{name} [get]
	// @Success 200 {array} Employee  "返回所有用户"
	FindByName(ctx context.Context, name string) (*Employee, error)

	// @Summary 按关键字查询用户数目，关键字可以是用户名，邮箱以及电话
	// @Param   department_id      query int                          false        "部门"
	// @Param   keyword            query string                       false        "搜索关键字"
	// @Accept  json
	// @Produce json
	// @Router  /employees/count [get]
	// @Success 200 {int64} int64  "返回所有用户数目"
	Count(ctx context.Context, departmentID int64, keyword string) (int64, error)

	// @Summary 按关键字查询用户，关键字可以是用户名，邮箱以及电话
	// @Param   department_id      query int                          false        "部门"
	// @Param   keyword            query string                       false        "搜索关键字"
	// @Param   offset             query int                          false        "offset"
	// @Param   limit              query int                          false        "limit"
	// @Param   sort               query string                       false        "排序字段"
	// @Accept  json
	// @Produce json
	// @Router  /employees [get]
	// @Success 200 {array} Employee  "返回所有用户"
	List(ctx context.Context, departmentID int64, keyword string, sort string, offset, limit int64) ([]Employee, error)
}

func NewRemoteEmployees(pxy *resty.Proxy) Employees {
	return EmployeesClient{
		Proxy: pxy,
	}
}
