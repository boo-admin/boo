//go:generate gogenv2 server -convert_param_types=UpdateMode -ext=.server-gen.go users.go
//go:generate gogenv2 client -convert_param_types=UpdateMode -ext=.client-gen.go users.go

package booclient

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/boo-admin/boo/errors"
	"github.com/boo-admin/boo/goutils/as"
	"github.com/runner-mei/resty"
	"golang.org/x/exp/slog"
)

var (
	None  = sql.NullBool{}
	False = sql.NullBool{Valid: true}
	True  = sql.NullBool{Valid: true, Bool: true}
)

type UpdateMode int

const (
	UpdateModeOverride UpdateMode = iota
	UpdateModeAdd
	UpdateModeSkip
)

func (mode UpdateMode) String() string {
	switch mode {
	case UpdateModeOverride:
		return "override"
	case UpdateModeAdd:
		return "add"
	case UpdateModeSkip:
		return "skip"
	}
	return fmt.Sprintf("unknown(%d)", int(mode))
}

func ParseUpdateMode(s string) (UpdateMode, error) {
	switch s {
	case "override", "":
		return UpdateModeOverride, nil
	case "add", "append":
		return UpdateModeAdd, nil
	case "skip":
		return UpdateModeSkip, nil
	}
	return UpdateModeOverride, errors.New("parse update mode fail - '" + s + "'")
}

type TagData struct {
	ID    int64  `json:"id" xorm:"id pk autoincr"`
	UUID  string `json:"uuid" xorm:"uuid unique notnull"`
	Title string `json:"title" xorm:"title unique notnull"`
}

type UserTags interface {
	// @Summary 新建一个用户标签
	// @Param    tag     body TagData    true     "用户标签"
	// @Accept   json
	// @Produce  json
	// @Router /users/tags [post]
	// @Success 200 {int64} int64  "成功时返回新建用户标签的ID"
	Create(ctx context.Context, tag *TagData) (int64, error)

	// @Summary 修改用户标签
	// @Param    id      path int        true     "用户标签ID"
	// @Param    tag     body TagData    true     "用户标签信息"
	// @Accept   json
	// @Produce  json
	// @Router /users/tags/{id} [put]
	// @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
	UpdateByID(ctx context.Context, id int64, tag *TagData) error

	// @Summary 删除指定的用户标签
	// @Param   id            path  int                       true     "用户标签ID"
	// @Accept  json
	// @Produce json
	// @Router  /users/tags/{id} [delete]
	// @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
	DeleteByID(ctx context.Context, id int64) error

	// @Summary 批量删除指定的用户标签
	// @Param   id            query int64                       true     "用户标签ID"
	// @Accept  json
	// @Produce json
	// @Router  /users/tags/batch [delete]
	// @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
	DeleteBatch(ctx context.Context, id []int64) error

	// @Summary 查询指定的用户标签
	// @Param   id              path  int                       true     "员工ID"
	// @Accept  json
	// @Produce json
	// @Router  /users/tags/{id} [get]
	// @Success 200 {object} TagData  "返回指定的用户标签"
	FindByID(ctx context.Context, id int64) (*TagData, error)

	// @Summary 按关键字查询用户标签
	// @Param   sort               query string                       false        "排序字段"
	// @Param   offset             query int                          false        "offset"
	// @Param   limit              query int                          false        "limit"
	// @Accept  json
	// @Produce json
	// @Router  /users/tags [get]
	// @Success 200 {array} TagData  "返回所有用户标签"
	List(ctx context.Context, sort string, offset, limit int64) ([]TagData, error)
}

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
	DeletedAt              *time.Time             `json:"deleted_at,omitempty" xorm:"deleted_at deleted"`
	CreatedAt              time.Time              `json:"created_at,omitempty" xorm:"created_at created"`
	UpdatedAt              time.Time              `json:"updated_at,omitempty" xorm:"updated_at updated"`

	IsDefault bool `json:"is_default" xorm:"-"`

	Department *Department `json:"department,omitempty" xorm:"-"`
	Roles      []Role      `json:"roles,omitempty" xorm:"-"`
	Tags       []TagData   `json:"tags,omitempty" xorm:"-"`
}

func (u *User) GetPhone() string {
	s := u.getString(Mobile.ID)
	if s != "" {
		return s
	}
	return u.getString(Telephone.ID)
}

func (u *User) GetEmail() string {
	return u.getString(Email.ID)
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

func (u *User) getString(key string) string {
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

func (u *User) GetStringWithDefault(key, defaultValue string) string {
	if u.Fields == nil {
		return defaultValue
	}
	o := u.Fields[key]
	if o == nil {
		return defaultValue
	}
	return as.StringWithDefault(o, defaultValue)
}

func (u *User) GetBoolWithDefault(key string, defaultValue bool) bool {
	if u.Fields == nil {
		return defaultValue
	}
	o := u.Fields[key]
	if o == nil {
		return defaultValue
	}
	return as.BoolWithDefault(o, defaultValue)
}

type Users interface {
	// @Summary 新建一个用户
	// @Param    user     body User    true     "用户定义"
	// @Accept   json
	// @Produce  json
	// @Router /users [post]
	// @Success 200 {int64} int64  "成功时返回新建用户的ID"
	Create(ctx context.Context, user *User) (int64, error)

	// @Summary 修改用户名称
	// @Param    id      path  int           true     "用户ID"
	// @Param    mode    query string        false     "更新模式, 可取值： override,add,skip"
	// @Param    user    body  User          true     "用户信息"
	// @Accept   json
	// @Produce  json
	// @Router /users/{id} [put]
	// @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
	UpdateByID(ctx context.Context, id int64, user *User, mode UpdateMode) error

	// @Summary 修改用户的密码
	// @Param    id           path int         true     "用户ID"
	// @Param    password     body string      true     "用户新密码"
	// @Accept   json
	// @Produce  json
	// @Router /users/{id}/change_password [put]
	// @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
	ChangePassword(ctx context.Context, id int64, password string) error

	// @Summary 删除指定的用户
	// @Param   id            path  int                       true     "用户ID"
	// @Param   force         query bool                      true     "是软删除还是真删除"
	// @Accept  json
	// @Produce json
	// @Router  /users/{id} [delete]
	// @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
	DeleteByID(ctx context.Context, id int64, force bool) error

	// @Summary 批量删除指定的用户
	// @Param   id            query int64                       true     "用户ID"
	// @Param   force         query bool                        true     "是软删除还是真删除"
	// @Accept  json
	// @Produce json
	// @Router  /users/batch [delete]
	// @Success 200 {string} string  "返回一个无意义的 'OK' 字符串"
	DeleteBatch(ctx context.Context, id []int64, force bool) error

	// @Summary 查询指定的用户
	// @Param   id              path int                       true     "用户ID"
	// @Param   include         query []string                     false        "指定返回的内容"
	// @Accept  json
	// @Produce json
	// @Router  /users/{id} [get]
	// @Success 200 {object} User  "返回指定的用户"
	FindByID(ctx context.Context, id int64, includes ...string) (*User, error)

	// @Summary 按名称查询指定的用户
	// @Param   name            path string                       true     "用户名"
	// @Param   include         query []string                     false        "指定返回的内容"
	// @Accept  json
	// @Produce json
	// @Router  /users/by_name/{name} [get]
	// @Success 200 {array} User  "返回所有用户"
	FindByName(ctx context.Context, name string, includes ...string) (*User, error)

	// @Summary 按关键字查询用户数目，关键字可以是用户名，邮箱以及电话
	// @Param   department_id      query int                          false        "部门"
	// @Param   role               query string                       false        "角色"
	// @Param   tag                query string                       false        "Tag"
	// @Param   keyword            query string                       false        "搜索关键字"
	// @Param   deleted            query sql.NullBool                 false        "指定是否包含删除的用户"
	// @Accept  json
	// @Produce json
	// @Router  /users/count [get]
	// @Success 200 {int64} int64  "返回所有用户数目"
	Count(ctx context.Context, departmentID int64, role, tag, keyword string, deleted sql.NullBool) (int64, error)

	// @Summary 按关键字查询用户，关键字可以是用户名，邮箱以及电话
	// @Param   department_id      query int                          false        "部门"
	// @Param   role               query string                       false        "角色"
	// @Param   tag                query string                       false        "Tag"
	// @Param   keyword            query string                       false        "搜索关键字"
	// @Param   deleted            query sql.NullBool                 false        "指定是否包含删除的用户"
	// @Param   include            query []string                     false        "指定返回的内容"
	// @Param   sort               query string                       false        "排序字段"
	// @Param   offset             query int                          false        "offset"
	// @Param   limit              query int                          false        "limit"
	// @Accept  json
	// @Produce json
	// @Router  /users [get]
	// @Success 200 {array} User  "返回所有用户"
	List(ctx context.Context, departmentID int64, role, tag, keyword string, deleted sql.NullBool, includes []string, sort string, offset, limit int64) ([]User, error)
}

func NewRemoteUsers(pxy *resty.Proxy) Users {
	return UsersClient{
		Proxy: pxy,
	}
}

func NewResty(baseURL string) (*resty.Proxy, error) {
	pxy, err := resty.New(baseURL)
	if err != nil {
		return nil, err
	}
	pxy = pxy.SetContentType(resty.MIMEApplicationJSONCharsetUTF8)
	pxy = pxy.ErrorFunc(errorFunc)
	return pxy, nil
}

func errorFunc(ctx context.Context, req *http.Request, resp *http.Response) resty.HTTPError {
	cached := resty.DefaultPool.Get()
	defer resty.DefaultPool.Put(cached)

	var err errors.EncodeError
	decoder := json.NewDecoder(io.TeeReader(resp.Body, cached))
	decoder.UseNumber()
	e := decoder.Decode(&err)
	if e != nil {
		return errors.WithHTTPCode(errors.Wrap(e, "request '"+req.Method+"' is ok and unmarshal response fail\r\n"+
			cached.String()), resty.ErrUnmarshalResponseFailCode())
	}
	if err.Message == "" {
		return errors.WithHTTPCode(errors.New("request '"+req.Method+"' is ok and unmarshal response fail\r\n"+
			cached.String()), resty.ErrUnmarshalResponseFailCode())
	}
	if err.Code == 0 {
		err.Code = resp.StatusCode
	}
	return &err
}
