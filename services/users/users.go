package users

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/boo-admin/boo/client"
	"github.com/boo-admin/boo/errors"
	"github.com/boo-admin/boo/goutils/as"
	"github.com/boo-admin/boo/goutils/importer"
	"github.com/boo-admin/boo/services/authn"
	"github.com/boo-admin/boo/validation"
	gobatis "github.com/runner-mei/GoBatis"
	"golang.org/x/exp/slog"

	"github.com/hjson/hjson-go/v4"
	good_password "github.com/mei-rune/go-good-password"
)

const DeleteTag = "(deleted:"

func AddDeleteSuffix(name string) string {
	if strings.Contains(name, DeleteTag) {
		return name
	}
	return name + DeleteTag + " " + time.Now().Format(time.RFC3339) + ")"
}

var NewUserDaoHook func(ref gobatis.SqlSession) UserDao

func NewUserDaoWith(ref gobatis.SqlSession) UserDao {
	if NewUserDaoHook != nil {
		return NewUserDaoHook(ref)
	}
	return NewUserDao(ref)
}

func NewUsers(env *client.Environment,
	db *gobatis.SessionFactory,
	operationLogger OperationLogger) (*UserService, error) {
	enablePasswordCheck := env.Config.BoolWithDefault("enable_password_check", false)
	var fields []CustomField
	if s := env.Config.StringWithDefault("usercustomfields", ""); s != "" {
		filename := client.GetRealDir(context.Background(), env, s)
		bs, err := ioutil.ReadFile(filename)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, errors.Wrap(err, "加载用户字段的配置失败")
			}
		} else if err = hjson.Unmarshal(bs, &fields); err != nil {
			return nil, errors.Wrap(err, "加载用户字段的配置失败")
		}

		if fields == nil {
			fields = client.DefaultFields
		}
	} else {
		fields = client.DefaultFields
	}

	var defaultUsernames = env.Config.StringsWithDefault("users.default_names", nil)

	passwordHasher, err := NewUserPassworder(env)
	if err != nil {
		return nil, errors.Wrap(err, "加载用户的 Hasher 失败")
	}

	sess := db.SessionReference()
	return &UserService{
		env:              env,
		logger:           env.Logger.WithGroup("users"),
		operationLogger:  operationLogger,
		db:               db,
		defaultUsernames: defaultUsernames,

		enablePasswordCheck: enablePasswordCheck,
		users:               NewUserDaoWith(sess),
		departments:         NewDepartmentDaoWith(sess),
		fields:              fields,
		passwordHasher:      passwordHasher,
	}, nil
}

type UserService struct {
	env              *client.Environment
	logger           *slog.Logger
	operationLogger  OperationLogger
	db               *gobatis.SessionFactory
	defaultUsernames []string

	enablePasswordCheck bool
	users               UserDao
	departments         DepartmentDao
	fields              []CustomField
	passwordHasher      UserPasswordHasher
}

func (svc UserService) ValidatePassword(usernames []string, password string) error {
	if svc.enablePasswordCheck {
		score, _ := good_password.Check(password, usernames)
		if score < 3 {
			// 1 ("terrible"): "something" (one type)
			// 2 ("weak"): "somethin1", "somethingnew" (two types)
			// 3 ("okay"): "Somethin1", "somethinglonger" (three types)
			// 4 ("good"): "Someth!n1", "somethingmuchlonger" (four types)
			// >=5 ("strong"): "Someth!n10", "correct horse battery staple" (five types)

			return errors.New("密码强度不足")
		}
	}
	return nil
}

func (svc UserService) ValidateUser(v *validation.Validation, user *User) bool {
	v.Required("name", user.Name)
	v.Required("nickname", user.Nickname)
	if user.Source != "ldap" && user.Source != "cas" && user.Source != "oauth" && !isAllStar(user.Password) {
		v.MinSize("Password", user.Password, 8)
		v.MaxSize("Password", user.Password, 250)

		if err := svc.ValidatePassword([]string{user.Name, user.Nickname}, user.Password); err != nil {
			v.Error("Password", err.Error())
		}
	}

	o := user.Fields[client.WhiteAddressList.ID]
	if o != nil {
		var ss = as.ToStrings(o)
		if len(ss) != 0 {
			for _, s := range ss {
				v.IPAddr("fields."+client.WhiteAddressList.ID, s,
					validation.IPAny,
					validation.IPv4CIDR,
					validation.IPv6CIDR,
					validation.IPv4MappedIPv6CIDR,
					validation.IPv4RangeV1,
					validation.IPv4RangeV2)
			}
		}
	}
	return v.HasErrors()
}

func (svc UserService) Create(ctx context.Context, user *User) (int64, error) {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return 0, err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpCreateUser); err != nil {
		return 0, errors.Wrap(err, "判断当前用户是否有权限失败")
	} else if !ok {
		return 0, errors.NewOperationReject(authn.OpCreateUser)
	}
	return svc.insert(ctx, currentUser, user, actionNormal)
}

func (svc UserService) insert(ctx context.Context, currentUser authn.AuthUser, user *User, importUser int) (int64, error) {
	v := validation.Default.New()
	if exists, err := svc.users.UsernameExists(ctx, user.Name); err != nil {
		return 0, errors.Wrap(err, "查询用户名 '"+user.Name+"' 是否已存在失败")
	} else if exists {
		v.Error("name", "无法新建用户 '"+user.Name+"'，该用户名已存在")
	}
	if exists, err := svc.users.NicknameExists(ctx, user.Nickname); err != nil {
		return 0, errors.Wrap(err, "查询用户呢称 '"+user.Nickname+"' 是否已存在失败")
	} else if exists {
		v.Error("name", "无法新建用户 '"+user.Name+"'，该用户呢称 '"+user.Nickname+"' 已存在")
	}
	if svc.ValidateUser(v, user) {
		return 0, v.ToError()
	}

	if password, err := svc.passwordHasher.Hash(ctx, user.Password); err != nil {
		return 0, errors.Wrap(err, "加密用户密码失败")
	} else {
		user.Password = password
	}

	var id int64
	var err error
	err = svc.db.InTx(ctx, nil, false, func(ctx context.Context, tx *gobatis.Tx) error {
		id, err = svc.users.Insert(ctx, user)
		if err != nil {
			return err
		}

		svc.logCreate(ctx, tx, currentUser, id, user, importUser)
		return nil
	})
	return id, err
}
func (svc UserService) UpdateByID(ctx context.Context, id int64, user *User) error {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpUpdateUser); err != nil {
		return errors.Wrap(err, "判断当前用户是否有权限失败")
	} else if !ok {
		return errors.NewOperationReject(authn.OpUpdateUser)
	}
	old, err := svc.users.FindByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "更新用户 '"+strconv.FormatInt(id, 10)+"' 失败")
	}
	return svc.update(ctx, currentUser, id, user, old, actionNormal)
}

func (svc UserService) update(ctx context.Context, currentUser authn.AuthUser, id int64, user, old *User, importUser int) error {
	return svc.db.InTx(ctx, nil, false, func(ctx context.Context, tx *gobatis.Tx) error {
		if old.Name != user.Name {
			return errors.New("更新用户失败，用户名不可修改")
		}
		if old.Source != user.Source {
			return errors.New("更新用户失败，字段 'source' 不可修改")
		}
		if user.Password != "" && !isAllStar(user.Password) {
			return errors.New("更新用户失败，密码不可以用此方式修改")
		}

		v := validation.Default.New()

		if old.Nickname != user.Nickname {
			if exists, err := svc.users.NicknameExists(ctx, user.Nickname); err != nil {
				return errors.Wrap(err, "更新用户时，查询用户呢称 '"+user.Nickname+"' 是否已存在失败")
			} else if exists {
				v.Error("name", "无法新建用户 '"+user.Name+"'，该用户呢称 '"+user.Nickname+"' 已存在")
			}
		}
		newUser := *old
		newUser.DepartmentID = user.DepartmentID
		if user.Nickname != "" {
			newUser.Nickname = user.Nickname
		}
		if importUser == actionImport {
			if user.Description != "" {
				newUser.Description = user.Description
			}
			if user.Disabled {
				newUser.Disabled = user.Disabled
			}
		} else {
			newUser.Description = user.Description
			newUser.Disabled = user.Disabled
		}

		if len(user.Fields) > 0 {
			if newUser.Fields == nil {
				newUser.Fields = user.Fields
			} else {
				newUser.Fields = map[string]interface{}{}
				for key, value := range old.Fields {
					newUser.Fields[key] = value
				}
				for key, value := range user.Fields {
					newValue, exist := newUser.Fields[key]
					if exist {
						if newValue == nil {
							delete(newUser.Fields, key)
						} else {
							newUser.Fields[key] = value
						}
					} else {
						newUser.Fields[key] = value
					}
				}
			}
		}

		newUser.Password = "****"
		if svc.ValidateUser(v, &newUser) {
			return v.ToError()
		}
		newUser.Password = old.Password

		err := svc.users.UpdateByID(ctx, id, &newUser)
		if err != nil {
			return err
		}

		svc.logUpdate(ctx, tx, currentUser, id, &newUser, old, importUser)
		return nil
	})
}
func (svc UserService) ChangePassword(ctx context.Context, id int64, password string) error {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return err
	}

	var names []string
	if currentUser.ID() != id {
		if ok, err := currentUser.HasPermission(ctx, authn.OpResetPassword); err != nil {
			return errors.Wrap(err, "判断当前用户是否有权限失败")
		} else if !ok {
			return errors.NewOperationReject(authn.OpResetPassword)
		}
		old, err := svc.users.FindByID(ctx, id)
		if err != nil {
			return errors.Wrap(err, "重置用户密码时，查询用户 '"+strconv.FormatInt(id, 10)+"' 失败")
		}
		names = []string{old.Name, old.Nickname}
	} else {
		names = []string{currentUser.Name(), currentUser.Nickname()}
	}

	return svc.resetPassword(ctx, currentUser, id, names, password, false)
}

func (svc UserService) resetPassword(ctx context.Context, currentUser authn.AuthUser, id int64, names []string, password string, importUser bool) error {
	if err := svc.ValidatePassword(names, password); err != nil {
		return err
	}

	if pwd, err := svc.passwordHasher.Hash(ctx, password); err != nil {
		return errors.Wrap(err, "加密用户密码失败")
	} else {
		password = pwd
	}

	if err := svc.users.UpdateUserPassword(ctx, id, password); err != nil {
		return errors.Wrap(err, "修改密码失败")
	}

	svc.logResetPassword(ctx, nil, currentUser, id, names[0], importUser)
	return nil
}

func (svc UserService) DeleteByID(ctx context.Context, id int64, force bool) error {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpDeleteUser); err != nil {
		return errors.Wrap(err, "判断当前用户是否有权限失败")
	} else if !ok {
		return errors.NewOperationReject(authn.OpDeleteUser)
	}

	old, err := svc.users.FindByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "删除用户时，查询用户 '"+strconv.FormatInt(id, 10)+"' 失败")
	}

	return svc.db.InTx(ctx, nil, true, func(ctx context.Context, tx *gobatis.Tx) error {
		if !force {
			if newName := AddDeleteSuffix(old.Name); newName != old.Name {
				old.Nickname = AddDeleteSuffix(old.Nickname)

				err := svc.users.UpdateByID(ctx, id, old)
				if err != nil {
					return errors.Wrap(err, "删除用户时，更新用户 '"+strconv.FormatInt(id, 10)+"' 的名称失败")
				}
			}
		}

		err := svc.users.DeleteByID(ctx, id, force)
		if err != nil {
			return errors.Wrap(err, "删除用户失败")
		}

		svc.logDelete(ctx, nil, currentUser, old)
		return nil
	})
}

func (svc UserService) DeleteBatch(ctx context.Context, idlist []int64, force bool) error {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpDeleteUser); err != nil {
		return errors.Wrap(err, "判断当前用户是否有权限失败")
	} else if !ok {
		return errors.NewOperationReject(authn.OpDeleteUser)
	}

	oldList, err := svc.users.FindByIDList(ctx, idlist)
	if err != nil {
		return errors.Wrap(err, "删除用户时，查询用户失败")
	}
	var newList = make([]int64, 0, len(oldList))
	for _, old := range oldList {
		newList = append(newList, old.ID)
	}

	return svc.db.InTx(ctx, nil, true, func(ctx context.Context, tx *gobatis.Tx) error {
		if !force {
			for _, old := range oldList {
				if newName := AddDeleteSuffix(old.Name); newName != old.Name {
					old.Nickname = AddDeleteSuffix(old.Nickname)

					err = svc.users.UpdateByID(ctx, old.ID, &old)
					if err != nil {
						return errors.Wrap(err, "删除用户时，更新用户 '"+strconv.FormatInt(old.ID, 10)+"' 的名称失败")
					}
				}
			}
		}

		err = svc.users.DeleteByIDList(ctx, newList, force)
		if err != nil {
			return errors.Wrap(err, "删除用户失败")
		}
		for _, old := range oldList {
			svc.logDelete(ctx, tx, currentUser, &old)
		}
		return nil
	})
}

func (svc UserService) FindByID(ctx context.Context, id int64, includes ...string) (*User, error) {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if currentUser.ID() != id {
		if ok, err := currentUser.HasPermission(ctx, authn.OpViewUser); err != nil {
			return nil, errors.Wrap(err, "判断当前用户是否有权限失败")
		} else if !ok {
			return nil, errors.NewOperationReject(authn.OpViewUser)
		}
	}

	user, err := svc.users.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	svc.processUser(user)
	return user, nil
}
func (svc UserService) processUser(user *User) {
	user.Password = "******"

	for _, name := range svc.defaultUsernames {
		if user.Name == name {
			user.IsDefault = true
			break
		}
	}
}
func (svc UserService) FindByName(ctx context.Context, name string, includes ...string) (*User, error) {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if currentUser.Name() != name {
		if ok, err := currentUser.HasPermission(ctx, authn.OpViewUser); err != nil {
			return nil, errors.Wrap(err, "判断当前用户是否有权限失败")
		} else if !ok {
			return nil, errors.NewOperationReject(authn.OpViewUser)
		}
	}

	user, err := svc.users.FindByName(ctx, name)
	if err != nil {
		return nil, err
	}

	svc.processUser(user)
	return user, nil
}
func (svc UserService) Count(ctx context.Context, departmentID int64, tag, keyword string, deleted sql.NullBool) (int64, error) {
	return svc.users.Count(ctx, departmentID, 0, tag, keyword, deleted)
}
func (svc UserService) List(ctx context.Context, departmentID int64, tag, keyword string, deleted sql.NullBool, includes []string, sort string, offset, limit int64) ([]User, error) {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpViewUser); err != nil {
		return nil, errors.Wrap(err, "判断当前用户是否有权限失败")
	} else if !ok {
		return nil, errors.NewOperationReject(authn.OpViewUser)
	}

	list, err := svc.users.List(ctx, departmentID, 0, tag, keyword, deleted, sort, offset, limit)
	if err != nil {
		return nil, err
	}
	for idx := range list {
		svc.processUser(&list[idx])
	}
	return list, nil
}
func (svc UserService) Export(ctx context.Context, format string, inline bool, writer http.ResponseWriter) error {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpViewUser); err != nil {
		return errors.Wrap(err, "判断当前用户是否有权限失败")
	} else if !ok {
		return errors.NewOperationReject(authn.OpViewUser)
	}

	return importer.WriteHTTP(ctx, "users", format, inline, writer,
		importer.RecorderFunc(func(ctx context.Context) (importer.RecordIterator, []string, error) {
			list, err := svc.users.List(ctx, 0, 0, "", "", sql.NullBool{Valid: true}, "", 0, 0)
			if err != nil {
				return nil, nil, err
			}
			titles := []string{
				"用户名",
				"中文名",
				"部门",
			}
			for _, f := range svc.fields {
				titles = append(titles, f.Name)
			}
			titles = append(titles, []string{
				// "部门",
				// "工号",
				// "性别",
				// "类型",
				// "岗位",
				// "座机号",
				// "手机号",
				// "房间号",
				// "座位号",
				"创建时间",
				"更新时间",
			}...)
			departmentCache := map[int64]*Department{}
			index := -1

			return importer.RecorderFuncIterator{
				CloseFunc: func() error {
					return nil
				},
				NextFunc: func(ctx context.Context) bool {
					index++
					return index < len(list)
				},
				ReadFunc: func(ctx context.Context) ([]string, error) {
					department := departmentCache[list[index].DepartmentID]
					if department == nil {
						d, err := svc.departments.FindByID(ctx, list[index].DepartmentID)
						if err != nil {
							return nil, err
						}
						departmentCache[list[index].DepartmentID] = d
						department = d
					}

					var values = make([]string, 0, 5+len(svc.fields))
					values = append(values, list[index].Name)
					values = append(values, list[index].Nickname)
					values = append(values, department.Name)
					for _, f := range svc.fields {
						if s := list[index].GetStringWithDefault(f.ID, ""); s != "" {
							values = append(values, s)
						}
					}
					values = append(values,
						formatTime(list[index].CreatedAt),
						formatTime(list[index].UpdatedAt))
					return values, nil
				},
			}, titles, nil
		}))
}

func (svc UserService) Import(ctx context.Context, request *http.Request) error {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return err
	}

	canCreate := false
	if ok, err := currentUser.HasPermission(ctx, authn.OpCreateUser); err != nil {
		return errors.Wrap(err, "判断当前用户是否有权限失败")
	} else {
		canCreate = ok
	}

	canUpdate := false
	if ok, err := currentUser.HasPermission(ctx, authn.OpUpdateUser); err != nil {
		return errors.Wrap(err, "判断当前用户是否有权限失败")
	} else {
		canUpdate = ok
	}

	canResetPassword := false
	if ok, err := currentUser.HasPermission(ctx, authn.OpResetPassword); err != nil {
		return errors.Wrap(err, "判断当前用户是否有权限失败")
	} else {
		canResetPassword = ok
	}

	canCreateDepartment := false
	if ok, err := currentUser.HasPermission(ctx, authn.OpCreateDepartment); err != nil {
		return errors.Wrap(err, "判断当前用户是否有权限失败")
	} else {
		canCreateDepartment = ok
	}

	ctx = context.WithValue(ctx, importer.ContextToRealDirKey, client.ToRealDirFunc(svc.env))
	reader, closer, err := importer.ReadHTTP(ctx, request)
	if err != nil {
		return err
	}
	defer closer.Close()

	override := request.URL.Query().Get("override") == "true"
	departmentAutoCreate := request.URL.Query().Get("department_auto_create") == "true"

	return importer.Import(ctx, "", reader, func(ctx context.Context, lineNumber int) (importer.Row, error) {
		record := &User{}

		var columns = make([]importer.Column, 0, 5+len(svc.fields))
		columns = append(columns, importer.StrColumn([]string{"name", "用户", "姓名"}, true,
			func(ctx context.Context, lineNumber int, origin, value string) error {
				record.Name = value
				return nil
			}))
		columns = append(columns, importer.StrColumn([]string{"zh_name", "中文名"}, false,
			func(ctx context.Context, lineNumber int, origin, value string) error {
				record.Nickname = value
				return nil
			}))
		columns = append(columns, importer.StrColumn([]string{"password", "密码"}, false,
			func(ctx context.Context, lineNumber int, origin, value string) error {
				record.Password = value
				return nil
			}))

		for _, f := range svc.fields {
			func(f client.CustomField) {
				columns = append(columns, importer.StrColumn(append([]string{f.ID, f.Name}, f.Alias...), false,
					func(ctx context.Context, lineNumber int, origin, value string) error {
						if record.Fields == nil {
							record.Fields = map[string]interface{}{}
						}
						record.Fields[f.ID] = value
						return nil
					}))
			}(f)
		}

		columns = append(columns, importer.StrColumn([]string{"department", "部门处室", "部门"}, false,
			func(ctx context.Context, lineNumber int, origin, value string) error {
				depart, err := svc.departments.FindByName(ctx, value)
				if err != nil {
					if !errors.IsNotFound(err) {
						return errors.Wrap(err, origin+" '"+value+"' 查询失败")
					}

					if !departmentAutoCreate {
						return errors.New(origin + " '" + value + "' 没有找到")
					}
					if !canCreateDepartment {
						return errors.New("没有创建部门的权限，部门 '" + value + "' 不存在")
					}
					id, err := svc.departments.Insert(ctx, &Department{
						Name: value,
					})
					if err != nil {
						return errors.Wrap(err, "创建 "+origin+" '"+value+"' 失败")
					}
					record.DepartmentID = id
					return nil
				}
				record.DepartmentID = depart.ID
				return nil
			}))

		return importer.Row{
			Columns: columns,
			Commit: func(ctx context.Context) error {
				old, err := svc.users.FindByName(ctx, record.Name)
				if err != nil && !errors.IsNotFound(err) {
					return err
				}
				if old != nil {
					if override {
						err = errors.New("用户 '" + record.Name + "' 已存在")
					} else if canUpdate {
						password := record.Password
						record.Password = ""

						if !canResetPassword && password != "" && !isAllStar(password) {
							err = errors.New("没有重置用户密码的权限，用户 '" + record.Name + "' 没有更新")
						} else {
							err = svc.update(ctx, currentUser, old.ID, record, old, actionImport)
							if err == nil {
								if password != "" && !isAllStar(password) {
									names := []string{record.Name, record.Nickname}
									if record.Nickname == "" {
										names[1] = old.Nickname
									}
									err = svc.resetPassword(ctx, currentUser, old.ID, names, password, true)
								}
							}
						}
					} else {
						err = errors.New("没有更新用户的权限，用户 '" + record.Name + "' 没有更新")
					}
				} else {
					if canCreate {
						if record.Nickname == "" {
							record.Nickname = record.Name
						}
						_, err = svc.insert(ctx, currentUser, record, actionImport)
					} else {
						err = errors.New("没有新建用户的权限，用户 '" + record.Name + "' 没有创建")
					}
				}
				return err
			},
		}, nil
	})
}

const (
	actionNormal = iota
	actionSync
	actionImport
)

func (svc UserService) logCreate(ctx context.Context, tx *gobatis.Tx, currentUser authn.AuthUser, id int64, user *User, importUser int) {
	if !enableOplog {
		return
	}
	records := make([]ChangeRecord, 0, 10)
	if user.DepartmentID > 0 {
		record := ChangeRecord{
			Name:        "department_id",
			DisplayName: "部门",
			NewValue:    user.DepartmentID,
		}
		d, err := svc.departments.FindByID(ctx, user.DepartmentID)
		if err != nil {
			svc.logger.WarnContext(ctx, "查询部门失败", slog.Any("err", err))
		} else if d != nil {
			record.NewDisplayValue = d.Name
		}
		records = append(records, record)
	}
	records = append(records, ChangeRecord{
		Name:        "name",
		DisplayName: "用户名",
		NewValue:    user.Name,
	}, ChangeRecord{
		Name:        "nickname",
		DisplayName: "呢称",
		NewValue:    user.Nickname,
	}, ChangeRecord{
		Name:        "description",
		DisplayName: "描述",
		NewValue:    user.Description,
	})
	for _, field := range svc.fields {
		fv := user.Fields[field.ID]
		if fv == nil {
			continue
		}
		records = append(records, ChangeRecord{
			Name:        field.ID,
			DisplayName: field.Name,
			NewValue:    fv,
		})
	}

	typeStr := authn.OpCreateUser
	content := "创建用户成功"
	switch importUser {
	case actionNormal:
	case actionSync:
		typeStr = "synccreateuser"
		content = "同步用户成功"
	case actionImport:
		typeStr = "importcreateuser"
		content = "导入用户成功"
	}

	err := svc.operationLogger.WithTx(tx.DB()).LogRecord(ctx, &OperationLog{
		UserID:     currentUser.ID(),
		Username:   currentUser.Nickname(),
		Successful: true,
		Type:       typeStr,
		Content:    content,
		Fields: &OperationLogRecord{
			ObjectType: "user",
			ObjectID:   id,
			Records:    records,
		},
	})
	if err != nil {
		svc.logger.WarnContext(ctx, "记录新建用户的操作失败", slog.Any("err", err))
	}
}

func (svc UserService) logUpdate(ctx context.Context, tx *gobatis.Tx, currentUser authn.AuthUser, id int64, user, old *User, importUser int) {
	if !enableOplog {
		return
	}
	records := make([]ChangeRecord, 0, 10)
	if user.DepartmentID != old.DepartmentID {
		var oldDepart, newDepart string
		if old.DepartmentID > 0 {
			d, err := svc.departments.FindByID(ctx, old.DepartmentID)
			if err != nil {
				svc.logger.WarnContext(ctx, "查询部门失败", slog.Any("err", err))
			} else {
				oldDepart = d.Name
			}
		}
		if user.DepartmentID > 0 {
			d, err := svc.departments.FindByID(ctx, user.DepartmentID)
			if err != nil {
				svc.logger.WarnContext(ctx, "查询部门失败", slog.Any("err", err))
			} else {
				newDepart = d.Name
			}
		}
		records = append(records, ChangeRecord{
			Name:            "department_id",
			DisplayName:     "部门",
			OldValue:        old.DepartmentID,
			NewValue:        user.DepartmentID,
			OldDisplayValue: oldDepart,
			NewDisplayValue: newDepart,
		})
	}

	if user.Name != old.Name {
		records = append(records, ChangeRecord{
			Name:        "name",
			DisplayName: "用户名",
			OldValue:    old.Name,
			NewValue:    user.Name,
		})
	}

	if user.Nickname != old.Nickname {
		records = append(records, ChangeRecord{
			Name:        "nickname",
			DisplayName: "呢称",
			OldValue:    old.Nickname,
			NewValue:    user.Nickname,
		})
	}

	if user.Description != old.Description {
		records = append(records, ChangeRecord{
			Name:        "description",
			DisplayName: "描述",
			OldValue:    old.Description,
			NewValue:    user.Description,
		})
	}

	for _, field := range svc.fields {
		var oldfv, newfv interface{}
		if len(old.Fields) > 0 {
			oldfv = old.Fields[field.ID]
		}
		if len(user.Fields) > 0 {
			newfv = user.Fields[field.ID]
		}
		if oldfv == nil && newfv == nil {
			continue
		}
		if oldfv != nil && newfv != nil {
			if fmt.Sprint(oldfv) == fmt.Sprint(newfv) {
				continue
			}
		}

		records = append(records, ChangeRecord{
			Name:        field.ID,
			DisplayName: field.Name,
			OldValue:    oldfv,
			NewValue:    newfv,
		})
	}

	typeStr := authn.OpUpdateUser
	content := "更新用户成功"
	switch importUser {
	case actionNormal:
	case actionSync:
		typeStr = "syncupdateuser"
		content = "同步更新用户成功"
	case actionImport:
		typeStr = "importupdateuser"
		content = "导入更新用户成功"
	}
	err := svc.operationLogger.WithTx(tx.DB()).LogRecord(ctx, &OperationLog{
		UserID:     currentUser.ID(),
		Username:   currentUser.Nickname(),
		Successful: true,
		Type:       typeStr,
		Content:    content,
		Fields: &OperationLogRecord{
			ObjectType: "user",
			ObjectID:   id,
			Records:    records,
		},
	})
	if err != nil {
		svc.logger.WarnContext(ctx, "记录更新用户的操作失败", slog.Any("err", err))
	}
}

func (svc UserService) logResetPassword(ctx context.Context, tx *gobatis.Tx, currentUser authn.AuthUser, id int64, username string, importUser bool) {
	if !enableOplog {
		return
	}
	oplogger := svc.operationLogger
	if tx != nil {
		oplogger = oplogger.WithTx(tx.DB())
	}
	typeStr := authn.OpResetPassword
	if importUser {
		typeStr = "importuser"
	}
	err := oplogger.LogRecord(ctx, &OperationLog{
		UserID:     currentUser.ID(),
		Username:   currentUser.Nickname(),
		Successful: true,
		Type:       typeStr,
		Content:    "更新用户 '" + username + "' 的密码成功",
		Fields: &OperationLogRecord{
			ObjectType: "user",
			ObjectID:   id,
		},
	})
	if err != nil {
		svc.logger.WarnContext(ctx, "记录重置用户密码的操作失败", slog.Any("err", err))
	}
}

func (svc UserService) logDelete(ctx context.Context, tx *gobatis.Tx, currentUser authn.AuthUser, oldUser *User) {
	if !enableOplog {
		return
	}
	oplogger := svc.operationLogger
	if tx != nil {
		oplogger = oplogger.WithTx(tx.DB())
	}
	err := oplogger.LogRecord(ctx, &OperationLog{
		UserID:     currentUser.ID(),
		Username:   currentUser.Nickname(),
		Successful: true,
		Type:       authn.OpDeleteUser,
		Content:    "删除用户 '" + oldUser.Name + "' 成功",
		Fields: &OperationLogRecord{
			ObjectType: "user",
			ObjectID:   oldUser.ID,
		},
	})
	if err != nil {
		svc.logger.WarnContext(ctx, "记录删除用户的操作失败", slog.Any("err", err))
	}
}

// func parseTime(s string) (time.Time, error) {
// 	s = strings.Replace(s, "/", "-", -1)

// 	for _, layout := range append(as.TimeFormats,
// 		"2006-01-02 15:04",
// 		"2006-_1-02 15:04",
// 		"2006-01-_2 15:04",
// 		"2006-_1-_2 15:04",

// 		"01-02-06",
// 		"01-02-06 15:04",
// 		"01/02/06",
// 		"01/02/06 15:04",
// 		"1/02/06 15:04",
// 		"1/2/06 15:04",
// 	) {
// 		m, e := time.ParseInLocation(layout, s, as.TimeLocal)
// 		if nil == e {
// 			return m, nil
// 		}
// 	}
// 	return time.Time{}, errors.New("'" + s + "' 不是一个有效的时间")
// }

// func parseDateAndTime(dateStr, timeStr string) (time.Time, error) {
// 	if timeStr != "" {
// 		if dateStr == "" {
// 			return parseTime(timeStr)
// 		}
// 		return parseTime(dateStr + " " + timeStr)
// 	}

// 	return parseTime(dateStr)
// }

// nolint:unused
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

// nolint:unused
func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("2006-01-02")
}

// nolint:unused
func formatOnlyTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("15:04:05")
}

func isAllStar(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c != '*' {
			return false
		}
	}
	return true
}
