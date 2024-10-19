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

	// @default select count(*) FROM <tablename type="User" as="u" /> WHERE u.department_id = #{id} AND u.deleted_at IS NULL
	GetUserCount(ctx context.Context, id int64) (int64, error)

	// @default select count(*) FROM <tablename type="Employee" as="em" /> WHERE em.department_id = #{id} AND em.deleted_at IS NULL
	GetEmployeeCount(ctx context.Context, id int64) (int64, error)

	// @default UPDATE <tablename type="User" /> SET department_id = NULL WHERE department_id = #{id}
	UnsetDepartmentForUser(ctx context.Context, id int64) error

	// @default UPDATE <tablename type="Employee" /> SET department_id = NULL WHERE department_id = #{id}
	UnsetDepartmentForEmployee(ctx context.Context, id int64) error

	// @default UPDATE <tablename type="Department" /> SET parent_id = NULL WHERE parent_id = #{id}
	UnsetDepartmentForDepartment(ctx context.Context, id int64) error

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
	DeleteByID(ctx context.Context, id int64, force bool) error
	DeleteByIDList(ctx context.Context, id []int64, force bool) error

	FindByID(ctx context.Context, id int64) (*User, error)
	FindByName(ctx context.Context, name string) (*User, error)
	// @default SELECT count(*) from <tablename /> <where>
	//   <if test="departmentID &gt; 0" >department_id = #{departmentID} AND </if>
	//   <if test="tagID &gt; 0" >id in (select user_id from <tablename type="User2Tag" as="u2t" /> where u2t.tag_id =#{tagID})) AND </if>
	//   <if test="isNotEmpty(tag)" >id in (select user_id from <tablename type="User2Tag" as="u2t" /> where u2t.tag_id in (select id from <tablename type="UserTag" as="tag" /> where uuid=#{tag})) AND </if>
	//   <if test="deleted.Valid"><if test="deleted.Bool">deleted_at IS NOT NULL AND<else/>deleted_at IS NULL AND</if></if>
	//   <if test="isNotEmpty(keyword)">
	//   name like <like value="keyword" />
	//   OR nickname like <like value="keyword" />
	//   OR fields->>'<print value="constants.user_phone" />' like <like value="keyword" />
	//   OR fields->>'<print value="constants.user_email" />' like <like value="keyword" /></if>
	//   </where>
	// @mysql SELECT count(*) from <tablename /> <where>
	//   <if test="departmentID &gt; 0" >department_id = #{departmentID} AND </if>
	//   <if test="tagID &gt; 0" >id in (select user_id from <tablename type="User2Tag" as="u2t" /> where u2t.tag_id =#{tagID})) AND </if>
	//   <if test="isNotEmpty(tag)" >id in (select user_id from <tablename type="User2Tag" as="u2t" /> where u2t.tag_id in (select id from <tablename type="UserTag" as="tag" /> where uuid=#{tag})) AND </if>
	//   <if test="deleted.Valid"><if test="deleted.Bool">deleted_at IS NOT NULL AND<else/>deleted_at IS NULL AND</if></if>
	//   <if test="isNotEmpty(keyword)">
	//   name like <like value="keyword" />
	//   OR nickname like <like value="keyword" />
	//   OR fields->>'$.<print value="constants.user_phone" />' like <like value="keyword" />
	//   OR fields->>'$.<print value="constants.user_email" />' like <like value="keyword" /></if>
	//   </where>
	Count(ctx context.Context, departmentID int64, tagID int64, tag, keyword string, deleted sql.NullBool) (int64, error)
	// @default SELECT * from <tablename /> <where>
	//   <if test="departmentID &gt; 0" >department_id = #{departmentID} AND </if>
	//   <if test="tagID &gt; 0" >id in (select user_id from <tablename type="User2Tag" as="u2t" /> where u2t.tag_id =#{tagID})) AND </if>
	//   <if test="isNotEmpty(tag)" >id in (select user_id from <tablename type="User2Tag" as="u2t" /> where u2t.tag_id in (select id from <tablename type="UserTag" as="tag" /> where uuid=#{tag})) AND </if>
	//   <if test="deleted.Valid"><if test="deleted.Bool">deleted_at IS NOT NULL AND<else/>deleted_at IS NULL AND</if></if>
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
	//   <if test="tagID &gt; 0" >id in (select user_id from <tablename type="User2Tag" as="u2t" /> where u2t.tag_id =#{tagID})) AND </if>
	//   <if test="isNotEmpty(tag)" >id in (select user_id from <tablename type="User2Tag" as="u2t" /> where u2t.tag_id in (select id from <tablename type="UserTag" as="tag" /> where uuid=#{tag})) AND </if>
	//   <if test="deleted.Valid"><if test="deleted.Bool">deleted_at IS NOT NULL AND<else/>deleted_at IS NULL AND</if></if>
	//   <if test="isNotEmpty(keyword)">
	//   name like <like value="keyword" />
	//   OR nickname like <like value="keyword" />
	//   OR fields->>'$.<print value="constants.user_phone" />' like <like value="keyword" />
	//   OR fields->>'$.<print value="constants.user_email" />' like <like value="keyword" /></if>
	//   </where>
	// <pagination /> <sort_by />
	List(ctx context.Context, departmentID int64, tagID int64, tag, keyword string, deleted sql.NullBool, sort string, offset, limit int64) ([]User, error)
	FindByIDList(ctx context.Context, id []int64) ([]User, error)
}

type UserTag struct {
	TableName struct{}  `json:"-" xorm:"boo_user_tags"`
	ID        int64     `json:"id" xorm:"id pk autoincr"`
	UUID      string    `json:"uuid" xorm:"uuid unique notnull"`
	Title     string    `json:"title" xorm:"title unique notnull"`
	CreatedAt time.Time `json:"created_at,omitempty" xorm:"created_at created"`
	UpdatedAt time.Time `json:"updated_at,omitempty" xorm:"updated_at updated"`
}

func (t UserTag) ToTagData() client.TagData {
	return client.TagData{
		ID:    t.ID,
		UUID:  t.UUID,
		Title: t.Title,
	}
}

func ToUserTagFrom(t *client.TagData) *UserTag {
	return &UserTag{
		ID:    t.ID,
		UUID:  t.UUID,
		Title: t.Title,
	}
}

type User2Tag struct {
	TableName struct{} `json:"-" xorm:"boo_user_to_tags"`
	UserID    int64    `json:"user_id" xorm:"user_id unique(user_tag)"`
	TagID     int64    `json:"tag_id" xorm:"tag_id unique(user_tag)"`
}

// @gobatis.namespace boo
type UserTagDao interface {
	Insert(ctx context.Context, tag *UserTag) (int64, error)
	UpdateByID(ctx context.Context, id int64, tag *UserTag) error
	DeleteByID(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*UserTag, error)
	FindByUUID(ctx context.Context, uuid string) (*UserTag, error)

	// @default SELECT count(*) from <tablename /> <if test="isNotEmpty(keyword)"> WHERE
	//   uuid like <like value="keyword" /> or title like <like value="keyword" /> </if>
	Count(ctx context.Context, keyword string) (int64, error)
	// @default SELECT * from <tablename /> <if test="isNotEmpty(keyword)"> WHERE
	//   UUID like <like value="keyword" /> or title like <like value="keyword" /> </if>
	// <pagination /> <sort_by />
	List(ctx context.Context, keyword string, sort string, offset, limit int64) ([]UserTag, error)

	// @default SELECT id, uuid, title from <tablename type="UserTag" /> where id in (select tag_id from <tablename type="User2Tag" /> where user_id = #{userID})
	QueryByUserID(ctx context.Context, userID int64) ([]client.TagData, error)
}

// @gobatis.namespace boo
type User2TagDao interface {
	// @record_type User2Tag
	Upsert(ctx context.Context, userID, tagID int64) error
	// @record_type User2Tag
	Delete(ctx context.Context, userID, tagID int64) error
	// @record_type User2Tag
	DeleteByUserID(ctx context.Context, userID int64) error
	// @record_type User2Tag
	DeleteByTagID(ctx context.Context, tagID int64) error

	QueryByUserID(ctx context.Context, userID int64) ([]User2Tag, error)
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
type RoleDao interface {
	// @type select
	// @postgres SELECT true FROM <tablename type="Role" /> WHERE lower(name) = lower(#{name})  LIMIT 1
	// @default SELECT 1 FROM <tablename type="Role" /> WHERE lower(name) = lower(#{name})  LIMIT 1
	NameExists(ctx context.Context, name string) (bool, error)

	Insert(ctx context.Context, role *Role) (int64, error)
	UpdateByID(ctx context.Context, id int64, role *Role) error
	DeleteByID(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*Role, error)
	FindByName(ctx context.Context, name string) (*Role, error)
	// @default SELECT count(*) from <tablename /> <if test="isNotEmpty(keyword)"> WHERE
	//   name like <like value="keyword" /> or uuid like <like value="keyword" /> </if>
	Count(ctx context.Context, keyword string) (int64, error)
	// @default SELECT * from <tablename /> <if test="isNotEmpty(keyword)"> WHERE
	//   name like <like value="keyword" /> or uuid like <like value="keyword" /> </if>
	// <pagination /> <sort_by />
	List(ctx context.Context, keyword string, sort string, offset, limit int64) ([]Role, error)

	FindByIDList(ctx context.Context, id []int64) ([]Role, error)

	// @default SELECT * from <tablename type="Role" /> where id in (select role_id from <tablename type="User2Role" /> where user_id = #{userID})
	QueryByUserID(ctx context.Context, userID int64) ([]Role, error)
}

type User2Role struct {
	TableName struct{} `json:"-" xorm:"boo_user_to_roles"`
	UserID    int64    `json:"user_id" xorm:"user_id unique(user_role)"`
	RoleID    int64    `json:"role_id" xorm:"role_id unique(user_role)"`
}

// @gobatis.namespace boo
type User2RoleDao interface {
	// @record_type User2Role
	Upsert(ctx context.Context, userID, roleID int64) error
	// @record_type User2Role
	Delete(ctx context.Context, userID, roleID int64) error
	// @record_type User2Role
	DeleteByUserID(ctx context.Context, userID int64) error
	// @record_type User2Role
	DeleteByRoleID(ctx context.Context, roleID int64) error

	QueryByUserID(ctx context.Context, userID int64) ([]User2Role, error)
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
	// @type update
	BindToUser(ctx context.Context, id, userID int64) error
	DeleteByID(ctx context.Context, id int64, force bool) error
	DeleteByIDList(ctx context.Context, id []int64, force bool) error

	FindByUserID(ctx context.Context, userid int64) (*Employee, error)

	FindByID(ctx context.Context, id int64) (*Employee, error)
	FindByName(ctx context.Context, name string) (*Employee, error)
	// @default SELECT count(*) from <tablename /> <where>
	//   <if test="departmentID &gt; 0" >department_id = #{departmentID} AND </if>
	//   <if test="tagID &gt; 0" >id in (select user_id from <tablename type="Employee2Tag" as="e2t" /> where e2t.tag_id =#{tagID})) AND </if>
	//   <if test="isNotEmpty(tag)" >id in (select user_id from <tablename type="Employee2Tag" as="e2t" /> where e2t.tag_id in (select id from <tablename type="Employee2Tag" as="tag" /> where uuid=#{tag})) AND </if>
	//   <if test="deleted.Valid"><if test="deleted.Bool">deleted_at IS NOT NULL AND<else/>deleted_at IS NULL AND</if></if>
	//   <if test="isNotEmpty(keyword)">
	//   name like <like value="keyword" />
	//   OR nickname like <like value="keyword" />
	//   OR fields->>'<print value="constants.user_phone" />' like <like value="keyword" />
	//   OR fields->>'<print value="constants.user_email" />' like <like value="keyword" /></if>
	//   </where>
	// @mysql SELECT count(*) from <tablename /> <where>
	//   <if test="departmentID &gt; 0" >department_id = #{departmentID} AND </if>
	//   <if test="tagID &gt; 0" >id in (select user_id from <tablename type="Employee2Tag" as="e2t" /> where e2t.tag_id =#{tagID})) AND </if>
	//   <if test="isNotEmpty(tag)" >id in (select user_id from <tablename type="Employee2Tag" as="e2t" /> where e2t.tag_id in (select id from <tablename type="Employee2Tag" as="tag" /> where uuid=#{tag})) AND </if>
	//   <if test="deleted.Valid"><if test="deleted.Bool">deleted_at IS NOT NULL AND<else/>deleted_at IS NULL AND</if></if>
	//   <if test="isNotEmpty(keyword)">
	//   name like <like value="keyword" />
	//   OR nickname like <like value="keyword" />
	//   OR fields->>'$.<print value="constants.user_phone" />' like <like value="keyword" />
	//   OR fields->>'$.<print value="constants.user_email" />' like <like value="keyword" /></if>
	//   </where>
	Count(ctx context.Context, departmentID int64, tagID int64, tag, keyword string, deleted sql.NullBool) (int64, error)

	// @default SELECT * from <tablename /> <where>
	//   <if test="departmentID &gt; 0" >department_id = #{departmentID} AND </if>
	//   <if test="tagID &gt; 0" >id in (select user_id from <tablename type="Employee2Tag" as="e2t" /> where e2t.tag_id =#{tagID})) AND </if>
	//   <if test="isNotEmpty(tag)" >id in (select user_id from <tablename type="Employee2Tag" as="e2t" /> where e2t.tag_id in (select id from <tablename type="Employee2Tag" as="tag" /> where uuid=#{tag})) AND </if>
	//   <if test="deleted.Valid"><if test="deleted.Bool">deleted_at IS NOT NULL AND<else/>deleted_at IS NULL AND</if></if>
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
	//   <if test="tagID &gt; 0" >id in (select user_id from <tablename type="Employee2Tag" as="e2t" /> where e2t.tag_id =#{tagID})) AND </if>
	//   <if test="isNotEmpty(tag)" >id in (select user_id from <tablename type="Employee2Tag" as="e2t" /> where e2t.tag_id in (select id from <tablename type="Employee2Tag" as="tag" /> where uuid=#{tag})) AND </if>
	//   <if test="deleted.Valid"><if test="deleted.Bool">deleted_at IS NOT NULL AND<else/>deleted_at IS NULL AND</if></if>
	//   <if test="isNotEmpty(keyword)">
	//   name like <like value="keyword" />
	//   OR nickname like <like value="keyword" />
	//   OR fields->>'$.<print value="constants.user_phone" />' like <like value="keyword" />
	//   OR fields->>'$.<print value="constants.user_email" />' like <like value="keyword" /></if>
	//   </where>
	// <pagination /> <sort_by />
	List(ctx context.Context, departmentID int64, tagID int64, tag, keyword string, deleted sql.NullBool, sort string, offset, limit int64) ([]Employee, error)
	FindByIDList(ctx context.Context, id []int64) ([]Employee, error)

	// @default SELECT u.id as user_id, emp.id as employee_id, u.nickname as user_nickname, emp.nickname as employee_nickname, u.department_id as user_department_id, emp.department_id as employee_department_id
	// FROM <tablename type="User" alias="u" /> INNER JOIN  <tablename type="Employee" alias="emp" /> ON u.id == emp.user_id
	GetUserEmployeeDiff(ctx context.Context) ([]client.UserEmployeeDiff, error)
}

type EmployeeTag struct {
	TableName struct{}  `json:"-" xorm:"boo_employee_tags"`
	ID        int64     `json:"id" xorm:"id pk autoincr"`
	UUID      string    `json:"uuid" xorm:"uuid unique notnull"`
	Title     string    `json:"title" xorm:"title unique notnull"`
	CreatedAt time.Time `json:"created_at,omitempty" xorm:"created_at created"`
	UpdatedAt time.Time `json:"updated_at,omitempty" xorm:"updated_at updated"`
}

func (t EmployeeTag) ToTagData() client.TagData {
	return client.TagData{
		ID:    t.ID,
		UUID:  t.UUID,
		Title: t.Title,
	}
}

func ToEmployeeTagFrom(t *client.TagData) *EmployeeTag {
	return &EmployeeTag{
		ID:    t.ID,
		UUID:  t.UUID,
		Title: t.Title,
	}
}

type Employee2Tag struct {
	TableName  struct{} `json:"-" xorm:"boo_employee_to_tags"`
	EmployeeID int64    `json:"user_id" xorm:"employee_id unique(employee_tag)"`
	TagID      int64    `json:"tag_id" xorm:"tag_id unique(employee_tag)"`
}

// @gobatis.namespace boo
type EmployeeTagDao interface {
	Insert(ctx context.Context, tag *EmployeeTag) (int64, error)
	UpdateByID(ctx context.Context, id int64, tag *EmployeeTag) error
	DeleteByID(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*EmployeeTag, error)
	FindByUUID(ctx context.Context, uuid string) (*EmployeeTag, error)

	// @default SELECT count(*) from <tablename /> <if test="isNotEmpty(keyword)"> WHERE
	//   uuid like <like value="keyword" /> or title like <like value="keyword" /> </if>
	Count(ctx context.Context, keyword string) (int64, error)
	// @default SELECT * from <tablename /> <if test="isNotEmpty(keyword)"> WHERE
	//   UUID like <like value="keyword" /> or title like <like value="keyword" /> </if>
	// <pagination /> <sort_by />
	List(ctx context.Context, keyword string, sort string, offset, limit int64) ([]EmployeeTag, error)

	// @default SELECT id, uuid, title from <tablename type="EmployeeTag" /> where id in (select tag_id from <tablename type="Employee2Tag" /> where employee_id = #{employeeID})
	QueryByEmployeeID(ctx context.Context, employeeID int64) ([]client.TagData, error)
}

// @gobatis.namespace boo
type Employee2TagDao interface {
	// @record_type Employee2Tag
	Upsert(ctx context.Context, employeeID, tagID int64) error
	// @record_type Employee2Tag
	Delete(ctx context.Context, employeeID, tagID int64) error
	// @record_type Employee2Tag
	DeleteByEmployeeID(ctx context.Context, employeeID int64) error
	// @record_type Employee2Tag
	DeleteByTagID(ctx context.Context, tagID int64) error

	QueryByEmployeeID(ctx context.Context, employeeID int64) ([]Employee2Tag, error)
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
