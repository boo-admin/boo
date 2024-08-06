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

func (u *Employee) ToUser() *User {
	fields := map[string]interface{}{}
	for key, value := range u.Fields {
		fields[key] = value
	}

	return &User{
		DepartmentID: u.DepartmentID,
		Name: u.Name,
		Nickname: u.Nickname,
		Description: u.Description,
		Source: u.Source,
		Fields: fields,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
		Department: u.Department,
	}
}

func (u *Employee) From(user *User) {
	if len(user.Fields) > 0 {
		if u.Fields == nil {
			u.Fields = map[string]interface{}{}
		}
		for key, value := range user.Fields {
			u.Fields[key] = value
		}
	}

	u.DepartmentID = user.DepartmentID
	u.Name= user.Name
	u.Nickname= user.Nickname
	u.Description= user.Description
	u.Source= user.Source
	u.CreatedAt= user.CreatedAt
	u.UpdatedAt= user.UpdatedAt
	u.Department= user.Department
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

type UserEmployeeDiff struct {
	UserID int64   `json:"user_id" xorm:"user_id null"`
	EmployeeID int64 `json:"employee_id" xorm:"employee_id null"`

	UserNickname  string `json:"user_nickname" xorm:"user_nickname null"`
	EmployeeNickname  string `json:"employee_nickname"  xorm:"employee_nickname null"`

	UserDepartmentID  int64 `json:"user_department_id" xorm:"user_department_id null"`
	EmployeeDepartmentID  int64 `json:"employee_department_id" xorm:"employee_department_id null"`
}

type Employees interface {
	// @Summary 新建一个员工
	// @Param    employee     body Employee    true     "员工定义"
	// @Accept   json
	// @Produce  json
	// @Router /employees [post]
	// @Success 200 {int64} int64  "成功时返回新建员工的ID"
	Create(ctx context.Context, employee *Employee) (int64, error)

	// @Summary 修改员工名称
	// @Param    id      path int     true     "员工ID"
	// @Param    employee    body Employee    true     "员工信息"
	// @Accept   json
	// @Produce  json
	// @Router /employees/{id} [put]
	// @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
	UpdateByID(ctx context.Context, id int64, employee *Employee) error

	// @Summary 删除指定的员工
	// @Param   id            path int                       true     "员工ID"
	// @Accept  json
	// @Produce json
	// @Router  /employees/{id} [delete]
	// @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
	DeleteByID(ctx context.Context, id int64) error

	// @Summary 批量删除指定的员工
	// @Param   id            query int64                       true     "员工ID"
	// @Accept  json
	// @Produce json
	// @Router  /employees/batch [delete]
	// @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
	DeleteBatch(ctx context.Context, id []int64) error

	// @Summary 查询指定的员工
	// @Param id            path int                       true     "员工ID"
	// @Accept  json
	// @Produce json
	// @Router  /employees/{id} [get]
	// @Success 200 {object} Employee  "返回指定的员工"
	FindByID(ctx context.Context, id int64) (*Employee, error)

	// @Summary 按名称查询指定的员工
	// @Param   name            path string                       true     "员工名"
	// @Accept  json
	// @Produce json
	// @Router  /employees/by_name/{name} [get]
	// @Success 200 {array} Employee  "返回所有员工"
	FindByName(ctx context.Context, name string) (*Employee, error)

	// @Summary 按关键字查询员工数目，关键字可以是员工名，邮箱以及电话
	// @Param   department_id      query int                          false        "部门"
	// @Param   keyword            query string                       false        "搜索关键字"
	// @Accept  json
	// @Produce json
	// @Router  /employees/count [get]
	// @Success 200 {int64} int64  "返回所有员工数目"
	Count(ctx context.Context, departmentID int64, keyword string) (int64, error)

	// @Summary 按关键字查询员工，关键字可以是员工名，邮箱以及电话
	// @Param   department_id      query int                          false        "部门"
	// @Param   keyword            query string                       false        "搜索关键字"
	// @Param   offset             query int                          false        "offset"
	// @Param   limit              query int                          false        "limit"
	// @Param   sort               query string                       false        "排序字段"
	// @Accept  json
	// @Produce json
	// @Router  /employees [get]
	// @Success 200 {array} Employee  "返回所有员工"
	List(ctx context.Context, departmentID int64, keyword string, sort string, offset, limit int64) ([]Employee, error)

	// @Summary  用员工信息新建一个可登录用
	// @Param    id          path int     true     "员工ID"
	// @Param    password    body int     true     "密码"
	// @Accept   json
	// @Produce  json
	// @Router   /employees/{id}/users [post]
	// @Success  200 {string} string  "返回一个新建用户的 id"
	PushToUser(ctx context.Context, id int64, password string) (int64, error)

	// @Summary  将员工绑定到一个可登录用户
	// @Param    id          path int                        true     "员工ID"
	// @Param    userID      path int                        true     "用户ID"
	// @Param    fields      body map[string]interface{}     true     "用户信息 (此参数暂时不起效，请传空)"
	// @Accept   json
	// @Produce  json
	// @Router   /employees/{id}/users/{userID} [put]
	// @Success  200 {string} string  "返回一个无意义的 ok 字符串"
	BindToUser(ctx context.Context, id int64, userID int64, fields map[string]interface{}) error

	// @Summary  同员工和可登录用户的数据
	// @Param    from_users            body []int64     true     "需要将从可登录用户同步到员工的列表"
	// @Param    to_users              body []int64     true     "需要将从员工同步到可登录用户的列表"
	// @Param    password              body string      false    "将从员工同步到可登录用户时如果用户不存在需要新建用户，本参数为新建用户的密码"
	// @Param    create_if_not_exist   body bool        true     "员工不存在时创建它"
	// @Accept   json
	// @Produce  json
	// @Router   /employees/users/sync [post]
	// @Success  200 {array} UserEmployeeDiff  "返回员工和可登录用户之间的差异"
	SyncWithUsers(ctx context.Context, fromUsers []int64, toUsers []int64, password string, createIfNotExist bool)  error

	// @Summary  获取员工和可登录用户之间的差异列表
	// @Accept   json
	// @Produce  json
	// @Router   /employees/users/diff [post]
	// @Success  200 {array} UserEmployeeDiff  "返回员工和可登录用户之间的差异"
	GetUserEmployeeDiff(ctx context.Context) ([]UserEmployeeDiff, error)
}

func NewRemoteEmployees(pxy *resty.Proxy) Employees {
	return EmployeesClient{
		Proxy: pxy,
	}
}
