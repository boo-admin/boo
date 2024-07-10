//go:generate gobatis dao.go

package users

import (
	"context"
	"database/sql"
	"time"

	"github.com/boo-admin/boo/client"
	gobatis "github.com/runner-mei/GoBatis"
)

// @gobatis.namespace boo
type DepartmentDao interface {
	// @type select
	// @postgres SELECT true FROM <tablename type="Department" /> WHERE lower(name) = lower(#{name})  LIMIT 1
	// @default SELECT 1 FROM <tablename type="Department" /> WHERE lower(name) = lower(#{name})  LIMIT 1
	NameExists(ctx context.Context, name string) (bool, error)

	Insert(ctx context.Context, department *Department) (int64, error)
	UpdateByID(ctx context.Context, id int64, department *Department) error
	DeleteByID(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*Department, error)
	FindByName(ctx context.Context, name string) (*Department, error)
	// @default SELECT count(*) from <tablename /> <if test="isNotEmpty(keyword)"> WHERE
	//   name like <like value="keyword" /> or uuid like <like value="keyword" /> </if>
	Count(ctx context.Context, keyword string) (int64, error)
	// @default SELECT * from <tablename /> <if test="isNotEmpty(keyword)"> WHERE
	//   name like <like value="keyword" /> or uuid like <like value="keyword" /> </if>
	// <pagination /> <sort_by />
	List(ctx context.Context, keyword string, sort string, offset, limit int64) ([]Department, error)

	FindByIDList(ctx context.Context, id []int64) ([]Department, error)
}

// @gobatis.namespace boo
type UserDao interface {
	// @type select
	// @postgres SELECT true FROM <tablename type="User" /> WHERE lower(name) = lower(#{name})  LIMIT 1
	// @default SELECT 1 FROM <tablename type="User" /> WHERE lower(name) = lower(#{name})  LIMIT 1
	UsernameExists(ctx context.Context, name string) (bool, error)

	// @type select
	// @postgres SELECT true FROM <tablename type="User" /> WHERE nickname = #{name} LIMIT 1
	// @default SELECT 1 FROM <tablename type="User" /> WHERE nickname = #{name} LIMIT 1
	NicknameExists(ctx context.Context, name string) (bool, error)

	Insert(ctx context.Context, user *User) (int64, error)
	UpdateByID(ctx context.Context, id int64, u *User) error
	// @default UPDATE <tablename /> SET password = #{password}, last_password_modified_at = now() WHERE id = #{id}
	UpdateUserPassword(ctx context.Context, id int64, password string) error
	DeleteByID(ctx context.Context, id int64) error
	DeleteByIDList(ctx context.Context, id []int64) error

	FindByID(ctx context.Context, id int64) (*User, error)
	FindByName(ctx context.Context, name string) (*User, error)
	// @default SELECT count(*) from <tablename /> <where>
	//   <if test="departmentID &gt; 0" >department_id = #{departmentID} AND </if>
	//   <if test="isNotEmpty(keyword)">
	//   name like <like value="keyword" />
	//   OR nickname like <like value="keyword" />
	//   OR fields->>'<print value="constants.user_phone" />' like <like value="keyword" />
	//   OR fields->>'<print value="constants.user_email" />' like <like value="keyword" /></if>
	//   </where>
	// @mysql SELECT count(*) from <tablename /> <where>
	//   <if test="departmentID &gt; 0" >department_id = #{departmentID} AND </if>
	//   <if test="isNotEmpty(keyword)">
	//   name like <like value="keyword" />
	//   OR nickname like <like value="keyword" />
	//   OR fields->>'$.<print value="constants.user_phone" />' like <like value="keyword" />
	//   OR fields->>'$.<print value="constants.user_email" />' like <like value="keyword" /></if>
	//   </where>
	Count(ctx context.Context, departmentID int64, keyword string) (int64, error)
	// @default SELECT * from <tablename /> <where>
	//   <if test="departmentID &gt; 0" >department_id = #{departmentID} AND </if>
	//   <if test="isNotEmpty(keyword)">
	//   name like <like value="keyword" />
	//   OR nickname like <like value="keyword" />
	//   OR fields->>'<print value="constants.user_phone" />' like <like value="keyword" />
	//   OR fields->>'<print value="constants.user_email" />' like <like value="keyword" />
	//   </if>
	//   </where>
	// <pagination /> <sort_by />
	// @mysql SELECT * from <tablename /> <where>
	//   <if test="departmentID &gt; 0" >department_id = #{departmentID} AND </if>
	//   <if test="isNotEmpty(keyword)">
	//   name like <like value="keyword" />
	//   OR nickname like <like value="keyword" />
	//   OR fields->>'$.<print value="constants.user_phone" />' like <like value="keyword" />
	//   OR fields->>'$.<print value="constants.user_email" />' like <like value="keyword" /></if>
	//   </where>
	// <pagination /> <sort_by />
	List(ctx context.Context, departmentID int64, keyword string, sort string, offset, limit int64) ([]User, error)
	FindByIDList(ctx context.Context, id []int64) ([]User, error)
}

// @gobatis.namespace boo
type UserProfileDao interface {
	// @type select
	// @default SELECT value FROM <tablename type="UserProfile" /> WHERE user_id = #{userID} and name = #{name}
	ReadProfile(ctx context.Context, userID int64, name string) (string, error)

	// @type upsert
	// @record_type UserProfile
	WriteProfileByKey(ctx context.Context, userID int64, name, value string) error

	// @record_type UserProfile
	DeleteProfile(ctx context.Context, userID int64, name string) (int64, error)

	// @record_type UserProfile
	DeleteAllByUserID(ctx context.Context, userID int64) error

	// @record_type UserProfile
	DeleteAllByName(ctx context.Context, name string) error

	// @default SELECT name, value FROM <tablename type="UserProfile" /> WHERE user_id = #{userID}
	QueryBy(ctx context.Context, userID int64) (map[string]string, error)
}

type UserProfile struct {
	TableName struct{} `json:"-" xorm:"boo_user_profiles"`
	UserID    int64    `json:"user_id" xorm:"user_id pk unique(key)"`
	Name      string   `json:"name" xorm:"name pk unique(key) notnull"`
	Value     string   `json:"value,omitempty" xorm:"value"`

	CreatedAt time.Time `json:"created_at,omitempty" xorm:"created_at created"`
	UpdatedAt time.Time `json:"updated_at,omitempty" xorm:"updated_at updated"`
}

// @gobatis.namespace boo
type EmployeeDao interface {
	// @type select
	// @postgres SELECT true FROM <tablename type="Employee" /> WHERE lower(name) = lower(#{name})  LIMIT 1
	// @default SELECT 1 FROM <tablename type="Employee" /> WHERE lower(name) = lower(#{name})  LIMIT 1
	NameExists(ctx context.Context, name string) (bool, error)

	// @type select
	// @postgres SELECT true FROM <tablename type="Employee" /> WHERE nickname = #{name} LIMIT 1
	// @default SELECT 1 FROM <tablename type="Employee" /> WHERE nickname = #{name} LIMIT 1
	NicknameExists(ctx context.Context, name string) (bool, error)

	Insert(ctx context.Context, user *Employee) (int64, error)
	UpdateByID(ctx context.Context, id int64, u *Employee) error
	DeleteByID(ctx context.Context, id int64) error
	DeleteByIDList(ctx context.Context, id []int64) error

	FindByID(ctx context.Context, id int64) (*Employee, error)
	FindByName(ctx context.Context, name string) (*Employee, error)
	// @default SELECT count(*) from <tablename /> <where>
	//   <if test="departmentID &gt; 0" >department_id = #{departmentID} AND </if>
	//   <if test="isNotEmpty(keyword)">
	//   name like <like value="keyword" />
	//   OR nickname like <like value="keyword" />
	//   OR fields->>'<print value="constants.user_phone" />' like <like value="keyword" />
	//   OR fields->>'<print value="constants.user_email" />' like <like value="keyword" /></if>
	//   </where>
	// @mysql SELECT count(*) from <tablename /> <where>
	//   <if test="departmentID &gt; 0" >department_id = #{departmentID} AND </if>
	//   <if test="isNotEmpty(keyword)">
	//   name like <like value="keyword" />
	//   OR nickname like <like value="keyword" />
	//   OR fields->>'$.<print value="constants.user_phone" />' like <like value="keyword" />
	//   OR fields->>'$.<print value="constants.user_email" />' like <like value="keyword" /></if>
	//   </where>
	Count(ctx context.Context, departmentID int64, keyword string) (int64, error)
	// @default SELECT * from <tablename /> <where>
	//   <if test="departmentID &gt; 0" >department_id = #{departmentID} AND </if>
	//   <if test="isNotEmpty(keyword)">
	//   name like <like value="keyword" />
	//   OR nickname like <like value="keyword" />
	//   OR fields->>'<print value="constants.user_phone" />' like <like value="keyword" />
	//   OR fields->>'<print value="constants.user_email" />' like <like value="keyword" />
	//   </if>
	//   </where>
	// <pagination /> <sort_by />
	// @mysql SELECT * from <tablename /> <where>
	//   <if test="departmentID &gt; 0" >department_id = #{departmentID} AND </if>
	//   <if test="isNotEmpty(keyword)">
	//   name like <like value="keyword" />
	//   OR nickname like <like value="keyword" />
	//   OR fields->>'$.<print value="constants.user_phone" />' like <like value="keyword" />
	//   OR fields->>'$.<print value="constants.user_email" />' like <like value="keyword" /></if>
	//   </where>
	// <pagination /> <sort_by />
	List(ctx context.Context, departmentID int64, keyword string, sort string, offset, limit int64) ([]Employee, error)
	FindByIDList(ctx context.Context, id []int64) ([]Employee, error)
}

func init() {
	gobatis.Init(func(ctx *gobatis.InitContext) error {
		if ctx.Config == nil {
			ctx.Config = &gobatis.Config{}
		}
		if ctx.Config.Constants == nil {
			ctx.Config.Constants = map[string]interface{}{}
		}
		if _, ok := ctx.Config.Constants["user_phone"]; !ok {
			ctx.Config.Constants["user_phone"] = client.Phone.ID
		}
		if _, ok := ctx.Config.Constants["user_email"]; !ok {
			ctx.Config.Constants["user_email"] = client.Email.ID
		}
		return nil
	})
}

// @gobatis.namespace boo
type OperationLogDao interface {
	WithDB(gobatis.DBRunner) OperationLogDao
	Insert(ctx context.Context, ol *OperationLog) error
	DeleteBy(ctx context.Context, createdAt client.TimeRange) error
	Count(ctx context.Context, userids []int64, successful sql.NullBool, typeList []string, contentLike string, createdAt client.TimeRange) (int64, error)
	List(ctx context.Context, userids []int64, successful sql.NullBool, typeList []string, contentLike string, createdAt client.TimeRange, offset, limit int64, sortBy string) ([]OperationLog, error)
}
