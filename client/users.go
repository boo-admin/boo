//go:generate gogenv2 server -ext=.server-gen.go users.go
//go:generate gogenv2 client -ext=.client-gen.go users.go

package client

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/runner-mei/resty"
	"golang.org/x/exp/slog"
)

type User struct {
	TableName              struct{}               `json:"-" xorm:"boo_users"`
	ID                     int64                  `json:"id" xorm:"id pk autoincr"`
	DepartmentID           int64                  `json:"department_id,omitempty" xorm:"department_id null"`
	Name                   string                 `json:"name" xorm:"name unique notnull"`
	Nickname               string                 `json:"nickname" xorm:"nickname unique notnull"`
	Password               string                 `json:"password,omitempty" xorm:"password null"`
	LastPasswordModifiedAt time.Time              `json:"last_password_modified_at,omitempty" xorm:"last_password_modified_at null"`
	Description            string                 `json:"description,omitempty" xorm:"description clob null"`
	Source                 string                 `json:"source,omitempty" xorm:"source null"`
	Disabled               bool                   `json:"disabled,omitempty" xorm:"disabled null"`
	Fields                 map[string]interface{} `json:"fields" xorm:"fields jsonb null"`
	CreatedAt              time.Time              `json:"created_at,omitempty" xorm:"created_at created"`
	UpdatedAt              time.Time              `json:"updated_at,omitempty" xorm:"updated_at updated"`
}

func (u *User) GetPhone() string {
	return u.GetString(Phone.ID)
}

func (u *User) GetEmail() string {
	return u.GetString(Email.ID)
}

func (u *User) GetWhiteAddressList() []string {
	if u.Fields == nil {
		return nil
	}
	o := u.Fields[WhiteAddressList.ID]
	if o == nil {
		return nil
	}

	switch v := o.(type) {
	case string:
		if strings.HasPrefix(v, "[") {
			var ss []string
			err := json.Unmarshal([]byte(v), &ss)
			if err != nil {
				slog.Warn("GetWhiteAddressList() fail", slog.Any("err", err))
			}
			return ss
		}
		return strings.Split(v, ",")
	case []string:
		return v
	case []interface{}:
		var ss []string
		for _, o := range v {
			if o == nil {
				continue
			}
			s, _ := o.(string)
			if s != "" {
				ss = append(ss, s)
			}
		}
		return ss
	}
	return nil
}

func (u *User) GetString(key string) string {
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

type Users interface {
	// @Summary 新建一个用户
	// @Param    user     body User    true     "用户定义"
	// @Accept   json
	// @Produce  json
	// @Router /users [post]
	// @Success 200 {int64} int64  "成功时返回新建用户的ID"
	Insert(ctx context.Context, user *User) (int64, error)

	// @Summary 修改用户名称
	// @Param    id      path int     true     "用户ID"
	// @Param    user    body User    true     "用户信息"
	// @Accept   json
	// @Produce  json
	// @Router /users/{id} [put]
	// @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
	UpdateByID(ctx context.Context, id int64, user *User) error

	// @Summary 修改用户的密码
	// @Param    id           path int         true     "用户ID"
	// @Param    password     body string      true     "用户新密码"
	// @Accept   json
	// @Produce  json
	// @Router /users/{id}/change_password [put]
	// @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
	ChangePassword(ctx context.Context, id int64, password string) error

	// @Summary 删除指定的用户
	// @Param   id            path int                       true     "用户ID"
	// @Accept  json
	// @Produce json
	// @Router  /users/{id} [delete]
	// @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
	DeleteByID(ctx context.Context, id int64) error

	// @Summary 批量删除指定的用户
	// @Param   id            query int64                       true     "用户ID"
	// @Accept  json
	// @Produce json
	// @Router  /users/batch [delete]
	// @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
	DeleteBatch(ctx context.Context, id []int64) error

	// @Summary 查询指定的用户
	// @Param id            path int                       true     "用户ID"
	// @Accept  json
	// @Produce json
	// @Router  /users/{id} [get]
	// @Success 200 {object} User  "返回指定的用户"
	FindByID(ctx context.Context, id int64) (*User, error)

	// @Summary 按名称查询指定的用户
	// @Param   name            path string                       true     "用户名"
	// @Accept  json
	// @Produce json
	// @Router  /users/by_name/{name} [get]
	// @Success 200 {array} User  "返回所有用户"
	FindByName(ctx context.Context, name string) (*User, error)

	// @Summary 按关键字查询用户数目，关键字可以是用户名，邮箱以及电话
	// @Param   keyword            query string                       false     "搜索关键字"
	// @Accept  json
	// @Produce json
	// @Router  /users/count [get]
	// @Success 200 {int64} int64  "返回所有用户数目"
	Count(ctx context.Context, keyword string) (int64, error)

	// @Summary 按关键字查询用户，关键字可以是用户名，邮箱以及电话
	// @Param   keyword            query string                       false     "搜索关键字"
	// @Param   offset             query int                          false        "offset"
	// @Param   limit              query int                          false        "limit"
	// @Param   sort               query string                       false        "排序字段"
	// @Accept  json
	// @Produce json
	// @Router  /users [get]
	// @Success 200 {array} User  "返回所有用户"
	List(ctx context.Context, keyword string, sort string, offset, limit int64) ([]User, error)
}

func NewRemoteUsers(pxy *resty.Proxy) Users {
	return UsersClient{
		Proxy: pxy,
	}
}

func NewResty(baseURL string) (*resty.Proxy, error) {
	return resty.New(baseURL)
}
