package users

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/boo-admin/boo/booclient"
	"github.com/boo-admin/boo/errors"
	"github.com/boo-admin/boo/goutils/tid"
	"github.com/boo-admin/boo/goutils/importer"
	"github.com/boo-admin/boo/services/authn"
	"github.com/boo-admin/boo/validation"
	gobatis "github.com/runner-mei/GoBatis"
	"golang.org/x/exp/slog"

	"github.com/hjson/hjson-go/v4"
)

var NewEmployeeDaoHook func(ref gobatis.SqlSession) EmployeeDao

func NewEmployeeDaoWith(ref gobatis.SqlSession) EmployeeDao {
	if NewEmployeeDaoHook != nil {
		return NewEmployeeDaoHook(ref)
	}
	return NewEmployeeDao(ref)
}

var NewEmployeeTagDaoHook func(ref gobatis.SqlSession) EmployeeTagDao

func NewEmployeeTagDaoWith(ref gobatis.SqlSession) EmployeeTagDao {
	if NewEmployeeTagDaoHook != nil {
		return NewEmployeeTagDaoHook(ref)
	}
	return NewEmployeeTagDao(ref)
}

var NewEmployee2TagDaoHook func(ref gobatis.SqlSession) Employee2TagDao

func NewEmployee2TagDaoWith(ref gobatis.SqlSession) Employee2TagDao {
	if NewEmployee2TagDaoHook != nil {
		return NewEmployee2TagDaoHook(ref)
	}
	return NewEmployee2TagDao(ref)
}

func NewEmployees(env *booclient.Environment,
	db *gobatis.SessionFactory,
	users *UserService,
	operationLogger OperationLogger) (Employees, error) {
	var fields []CustomField
	if s := env.Config.StringWithDefault("employeecustomfields", ""); s != "" {
		filename := booclient.GetRealDir(context.Background(), env, s)
		bs, err := ioutil.ReadFile(filename)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, errors.Wrap(err, "加载员工字段的配置失败")
			}
		} else if err = hjson.Unmarshal(bs, &fields); err != nil {
			return nil, errors.Wrap(err, "加载员工字段的配置失败")
		}

		if fields == nil {
			fields = booclient.DefaultFields
		}
	} else {
		fields = booclient.DefaultFields
	}

	sess := db.SessionReference()
	return employeeService{
		env:             env,
		logger:          env.Logger.WithGroup("employees"),
		operationLogger: operationLogger,
		db:              db,
		departmentDao:     NewDepartmentDaoWith(sess),
		employeeDao:     NewEmployeeDaoWith(sess),
		employeeTagDao:  NewEmployeeTagDaoWith(sess),
		employee2TagDao: NewEmployee2TagDaoWith(sess),
		users:           users,
		fields:          fields,
	}, nil
}

type employeeService struct {
	env             *booclient.Environment
	logger          *slog.Logger
	operationLogger OperationLogger
	db              *gobatis.SessionFactory

	enablePasswordCheck bool
	departmentDao         DepartmentDao
	employeeDao         EmployeeDao
	employeeTagDao      EmployeeTagDao
	employee2TagDao     Employee2TagDao
	users               *UserService
	fields              []CustomField
}

func (svc employeeService) ValidateEmployee(v *validation.Validation, employee *Employee) bool {
	v.Required("name", employee.Name)
	v.Required("nickname", employee.Nickname)
	return v.HasErrors()
}

func (svc employeeService) Create(ctx context.Context, employee *Employee) (int64, error) {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return 0, err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpCreateEmployee); err != nil {
		return 0, errors.Wrap(err, "判断当前用户是否有权限失败")
	} else if !ok {
		return 0, errors.NewOperationReject(authn.OpCreateEmployee)
	}
	return svc.insert(ctx, currentUser, employee, actionNormal)
}

func (svc employeeService) insert(ctx context.Context, currentUser authn.AuthUser, employee *Employee, importEmployee int) (int64, error) {
	v := validation.Default.New()
	if exists, err := svc.employeeDao.NameExists(ctx, employee.Name); err != nil {
		return 0, errors.Wrap(err, "查询员工名 '"+employee.Name+"' 是否已存在失败")
	} else if exists {
		v.Error("name", "无法新建员工 '"+employee.Name+"'，该员工名已存在")
	}
	if exists, err := svc.employeeDao.NicknameExists(ctx, employee.Nickname); err != nil {
		return 0, errors.Wrap(err, "查询员工呢称 '"+employee.Nickname+"' 是否已存在失败")
	} else if exists {
		v.Error("name", "无法新建员工 '"+employee.Name+"'，该员工呢称 '"+employee.Nickname+"' 已存在")
	}
	if svc.ValidateEmployee(v, employee) {
		return 0, v.ToError()
	}

	var id int64
	var err error
	err = svc.db.InTx(ctx, nil, true, func(ctx context.Context, tx *gobatis.Tx) error {
		id, err = svc.employeeDao.Insert(ctx, employee)
		if err != nil {
			return err
		}

		var contents []ChangeRecord
		if importEmployee == actionNormal {
			if contents, err = svc.updateTags(ctx, id, employee, false); err != nil {
				return err
			}
		}

		svc.logCreate(ctx, tx, currentUser, id, employee, importEmployee, contents)
		return nil
	})
	return id, err
}

func (svc employeeService) UpdateByID(ctx context.Context, id int64, employee *Employee, mode booclient.UpdateMode) error {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpUpdateEmployee); err != nil {
		return errors.Wrap(err, "判断当前用户是否有权限失败")
	} else if !ok {
		return errors.NewOperationReject(authn.OpUpdateEmployee)
	}
	old, err := svc.employeeDao.FindByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "更新员工 '"+strconv.FormatInt(id, 10)+"' 失败")
	}
	return svc.update(ctx, currentUser, id, employee, old, mode, actionNormal)
}

func (svc employeeService) update(ctx context.Context, currentUser authn.AuthUser, id int64, employee, old *Employee, mode booclient.UpdateMode, importEmployee int) error {
	return svc.db.InTx(ctx, nil, true, func(ctx context.Context, tx *gobatis.Tx) error {
		if old.Name != employee.Name {
			return errors.New("更新员工失败，员工名不可修改")
		}

		v := validation.Default.New()

		if old.Nickname != employee.Nickname {
			if exists, err := svc.employeeDao.NicknameExists(ctx, employee.Nickname); err != nil {
				return errors.Wrap(err, "更新员工时，查询员工呢称 '"+employee.Nickname+"' 是否已存在失败")
			} else if exists {
				v.Error("name", "无法新建员工 '"+employee.Name+"'，该员工呢称 '"+employee.Nickname+"' 已存在")
			}
		}
		newEmployee := *old
		newEmployee.DepartmentID = employee.DepartmentID
		if employee.Nickname != "" {
			newEmployee.Nickname = employee.Nickname
		}
		newEmployee.Description = employee.Description
		// newEmployee.Disabled = employee.Disabled

		if len(employee.Fields) > 0 {
			if newEmployee.Fields == nil {
				newEmployee.Fields = employee.Fields
			} else {
				newEmployee.Fields = map[string]interface{}{}
				for key, value := range old.Fields {
					newEmployee.Fields[key] = value
				}
				for key, value := range employee.Fields {
					newValue, exist := newEmployee.Fields[key]
					if exist {
						if newValue == nil {
							delete(newEmployee.Fields, key)
						} else {
							newEmployee.Fields[key] = value
						}
					} else {
						newEmployee.Fields[key] = value
					}
				}
			}
		}

		if svc.ValidateEmployee(v, &newEmployee) {
			return v.ToError()
		}

		err := svc.employeeDao.UpdateByID(ctx, id, &newEmployee)
		if err != nil {
			return err
		}

		var contents []ChangeRecord
		switch mode {
		case booclient.UpdateModeOverride:
			if contents, err = svc.updateTags(ctx, id, employee, true); err != nil {
				return err
			}
		case booclient.UpdateModeAdd:
			if contents, err = svc.updateTags(ctx, id, employee, false); err != nil {
				return err
			}
		case booclient.UpdateModeSkip:
		default:
			return errors.New("不可识别的更新模式 - " + mode.String())
		}
		svc.logUpdate(ctx, tx, currentUser, id, &newEmployee, old, importEmployee, contents)
		return nil
	})
}

func (svc employeeService) updateTags(ctx context.Context, id int64, employee *Employee, isUpdate bool) ([]ChangeRecord, error) {
	var oldTags []Employee2Tag
	var err error
	if isUpdate {
		oldTags, err = svc.employee2TagDao.QueryByEmployeeID(ctx, id)
		if err != nil {
			return nil, errors.Wrap(err, "更新员工时查询旧 tag 列表失败")
		}
	}

	var contents = make([]ChangeRecord, 0, len(employee.Tags))
	for idx := range employee.Tags {
		tag := &employee.Tags[idx]

		isNewTag := false
		if tag.ID <= 0 {
			var old *EmployeeTag

			if tag.UUID != "" {
				old, err = svc.employeeTagDao.FindByUUID(ctx, tag.UUID)
				if err != nil {
					if !errors.Is(err, sql.ErrNoRows) {
						if isUpdate {
							return nil, errors.Wrap(err, "更新员工时查询关联 tag 失败")
						}
						return nil, errors.Wrap(err, "创建员工时查询关联 tag 失败")
					}
				}
			} else if tag.Title != "" {
				old, err = svc.employeeTagDao.FindByTitle(ctx, tag.Title)
				if err != nil {
					if !errors.Is(err, sql.ErrNoRows) {
						if isUpdate {
							return nil, errors.Wrap(err, "更新员工时查询关联 tag 失败")
						}
						return nil, errors.Wrap(err, "创建员工时查询关联 tag 失败")
					}
				}
			} else {
				continue
			}

			if old == nil {
				if tag.UUID == "" {
					tag.UUID = tid.GenerateID()
				}
				if tag.Title == "" {
					tag.Title = tag.UUID
				}
				tag.ID, err = svc.employeeTagDao.Insert(ctx, ToEmployeeTagFrom(tag))
				if err != nil {
					if isUpdate {
						return nil, errors.Wrap(err, "更新员工时查询关联 tag 失败")
					}
					return nil, errors.Wrap(err, "创建员工时查询关联 tag 失败")
				}
				isNewTag = true
			} else {
				tag.ID = old.ID
			}
		}

		if !isNewTag && isUpdate {
			found := false
			for _, old := range oldTags {
				if old.TagID == tag.ID {
					found = true
					break
				}
			}
			if found {
				continue
			}
		}

		err = svc.employee2TagDao.Upsert(ctx, id, tag.ID)
		if err != nil {
			if isUpdate {
				return nil, errors.Wrap(err, "更新员工时关联 tag 失败")
			}
			return nil, errors.Wrap(err, "创建员工时关联 tag 失败")
		}

		if tag.UUID != "" {
			contents = append(contents, ChangeRecord{
				Name:        "addTagUUID",
				DisplayName: "添加关联 Tag - '" + tag.UUID + "'",
				NewValue:    tag.UUID,
			})
		} else {
			contents = append(contents, ChangeRecord{
				Name:        "addTagID",
				DisplayName: "添加关联 Tag - '" + strconv.FormatInt(tag.ID, 10) + "'",
				NewValue:    tag.ID,
			})
		}
	}
	if isUpdate {
		for _, old := range oldTags {
			found := false
			for _, tag := range employee.Tags {
				if old.TagID == tag.ID {
					found = true
					break
				}
			}
			if found {
				continue
			}
			err = svc.employee2TagDao.Delete(ctx, old.EmployeeID, old.TagID)
			if err != nil {
				return nil, errors.Wrap(err, "创建员工时删除关联 tag 失败")
			}

			contents = append(contents, ChangeRecord{
				Name:        "deleteTagID",
				DisplayName: "删除关联 tag - '" + strconv.FormatInt(old.TagID, 10) + "'",
				NewValue:    old.TagID,
			})
		}
	}
	return contents, nil
}

func (svc employeeService) DeleteByID(ctx context.Context, id int64, force bool) error {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpDeleteEmployee); err != nil {
		return errors.Wrap(err, "判断当前用户是否有权限失败")
	} else if !ok {
		return errors.NewOperationReject(authn.OpDeleteEmployee)
	}

	old, err := svc.employeeDao.FindByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "删除员工时，查询员工 '"+strconv.FormatInt(id, 10)+"' 失败")
	}
	return svc.db.InTx(ctx, nil, true, func(ctx context.Context, tx *gobatis.Tx) error {
		if !force {
			if newName := AddDeleteSuffix(old.Name); newName != old.Name {
				old.Nickname = AddDeleteSuffix(old.Nickname)

				err := svc.employeeDao.UpdateByID(ctx, id, old)
				if err != nil {
					return errors.Wrap(err, "删除员工时，更新员工 '"+strconv.FormatInt(id, 10)+"' 的名称失败")
				}
			}
		}
		err := svc.employeeDao.DeleteByID(ctx, id, force)
		if err != nil {
			return errors.Wrap(err, "删除员工失败")
		}

		svc.logDelete(ctx, nil, currentUser, old)
		return nil
	})
}

func (svc employeeService) DeleteBatch(ctx context.Context, idlist []int64, force bool) error {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpDeleteEmployee); err != nil {
		return errors.Wrap(err, "判断当前用户是否有权限失败")
	} else if !ok {
		return errors.NewOperationReject(authn.OpDeleteEmployee)
	}

	oldList, err := svc.employeeDao.FindByIDList(ctx, idlist)
	if err != nil {
		return errors.Wrap(err, "删除员工时，查询员工失败")
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

					err = svc.employeeDao.UpdateByID(ctx, old.ID, &old)
					if err != nil {
						return errors.Wrap(err, "删除员工时，更新员工 '"+strconv.FormatInt(old.ID, 10)+"' 的名称失败")
					}
				}
			}
		}

		err = svc.employeeDao.DeleteByIDList(ctx, newList, force)
		if err != nil {
			return errors.Wrap(err, "删除员工失败")
		}
		for _, old := range oldList {
			svc.logDelete(ctx, tx, currentUser, &old)
		}
		return nil
	})
}

func (svc employeeService) FindByID(ctx context.Context, id int64, includes ...string) (*Employee, error) {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if currentUser.ID() != id {
		if ok, err := currentUser.HasPermission(ctx, authn.OpViewEmployee); err != nil {
			return nil, errors.Wrap(err, "判断当前用户是否有权限失败")
		} else if !ok {
			return nil, errors.NewOperationReject(authn.OpViewEmployee)
		}
	}

	employee, err := svc.employeeDao.FindByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "查询员工失败")
	}

	includes = splitIncludes(includes, GetEmployeeAllIncludes())
	return svc.loadEmployee(ctx, employee, includes)
}
func (svc employeeService) FindByName(ctx context.Context, name string, includes ...string) (*Employee, error) {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if ok, err := currentUser.HasPermission(ctx, authn.OpViewEmployee); err != nil {
		return nil, errors.Wrap(err, "判断当前用户是否有权限失败")
	} else if !ok {
		return nil, errors.NewOperationReject(authn.OpViewEmployee)
	}

	employee, err := svc.employeeDao.FindByName(ctx, name)
	if err != nil {
		return nil, errors.Wrap(err, "查询员工失败")
	}
	
	includes = splitIncludes(includes, GetEmployeeAllIncludes())
	return svc.loadEmployee(ctx, employee, includes)
}

func (svc employeeService) loadEmployee(ctx context.Context, employee *Employee, includes []string) (*Employee, error) {
	for _, include := range includes {
		switch include {
		case "department":
			if employee.DepartmentID > 0 {
				department, err := svc.departmentDao.FindByID(ctx, employee.DepartmentID)
				if err != nil {
					return nil, errors.Wrap(err, "加载部门失败")
				}
				employee.Department = department
			}
		case "tag", "tags":
			tags, err := svc.employeeTagDao.QueryByEmployeeID(ctx, employee.ID)
			if err != nil {
				return nil, errors.Wrap(err, "加载 Tag 失败")
			}
			employee.Tags = tags
		default:
			return nil, errors.WithCode(errors.New("参数 'include' 不正确 - '"+include+"'"), http.StatusBadRequest)
		}
	}
	return employee, nil
}
func (svc employeeService) Count(ctx context.Context, departmentID int64, tag, keyword string, deleted sql.NullBool) (int64, error) {
	return svc.employeeDao.Count(ctx, departmentID, 0, tag, keyword, deleted)
}
func (svc employeeService) List(ctx context.Context, departmentID int64, tag, keyword string, deleted sql.NullBool, includes []string, sort string, offset, limit int64) ([]Employee, error) {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpViewEmployee); err != nil {
		return nil, errors.Wrap(err, "判断当前用户是否有权限失败")
	} else if !ok {
		return nil, errors.NewOperationReject(authn.OpViewEmployee)
	}

	list, err := svc.employeeDao.List(ctx, departmentID, 0, tag, keyword, deleted, sort, offset, limit)
	if err != nil {
		return nil, errors.Wrap(err, "查询员工列表失败")
	}

	includes = splitIncludes(includes, GetEmployeeAllIncludes())

	for idx := range list {
		employee := &list[idx]
		_, err = svc.loadEmployee(ctx, employee, includes)
		if err != nil {
			return nil, err
		}
	}
	return list, nil
}

func (svc employeeService) PushToUser(ctx context.Context, id int64, password string) (int64, error) {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return 0, err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpUpdateEmployee); err != nil {
		return 0, errors.Wrap(err, "判断当前用户是否有权限失败")
	} else if !ok {
		return 0, errors.NewOperationReject(authn.OpUpdateEmployee)
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpCreateUser); err != nil {
		return 0, errors.Wrap(err, "判断当前用户是否有权限失败")
	} else if !ok {
		return 0, errors.NewOperationReject(authn.OpCreateUser)
	}

	employee, err := svc.employeeDao.FindByID(ctx, id)
	if err != nil {
		return 0, errors.Wrap(err, "查询员工失败")
	}

	u := employee.ToUser()
	u.Password = password

	var userID int64
	err = svc.db.InTx(ctx, nil, true, func(ctx context.Context, tx *gobatis.Tx) error {
		userID, err = svc.users.insert(ctx, currentUser, u, actionSync)
		if err != nil {
			return errors.Wrap(err, "新建用户失败")
		}

		err = svc.employeeDao.BindToUser(ctx, id, userID)
		if err != nil {
			return err
		}

		svc.logBind(ctx, tx, currentUser, employee)
		return nil
	})
	return userID, err
}

func (svc employeeService) BindToUser(ctx context.Context, id int64, userID int64, fields map[string]interface{}) error {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpUpdateEmployee); err != nil {
		return errors.Wrap(err, "判断当前用户是否有权限失败")
	} else if !ok {
		return errors.NewOperationReject(authn.OpUpdateEmployee)
	}

	employee, err := svc.employeeDao.FindByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "查询员工失败")
	}

	return svc.db.InTx(ctx, nil, true, func(ctx context.Context, tx *gobatis.Tx) error {
		err := svc.employeeDao.BindToUser(ctx, id, userID)
		if err != nil {
			return err
		}
		svc.logBind(ctx, tx, currentUser, employee)
		return nil
	})
}

func (svc employeeService) SyncWithUsers(ctx context.Context, fromUsers []int64, toUsers []int64, password string, createIfNotExist bool) error {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return err
	}
	if len(fromUsers) > 0 {
		if ok, err := currentUser.HasPermission(ctx, authn.OpUpdateEmployee); err != nil {
			return errors.Wrap(err, "判断当前用户是否有权限失败")
		} else if !ok {
			return errors.NewOperationReject(authn.OpUpdateEmployee)
		}

		if createIfNotExist {
			if ok, err := currentUser.HasPermission(ctx, authn.OpCreateEmployee); err != nil {
				return errors.Wrap(err, "判断当前用户是否有权限失败")
			} else if !ok {
				return errors.NewOperationReject(authn.OpCreateEmployee)
			}
		}

		if ok, err := currentUser.HasPermission(ctx, authn.OpViewUser); err != nil {
			return errors.Wrap(err, "判断当前用户是否有权限失败")
		} else if !ok {
			return errors.NewOperationReject(authn.OpViewUser)
		}

		for _, userID := range fromUsers {
			err := svc.syncFromUser(ctx, currentUser, userID, createIfNotExist)
			if err != nil {
				return err
			}
		}
	}

	if len(toUsers) > 0 {
		if ok, err := currentUser.HasPermission(ctx, authn.OpUpdateUser); err != nil {
			return errors.Wrap(err, "判断当前用户是否有权限失败")
		} else if !ok {
			return errors.NewOperationReject(authn.OpUpdateUser)
		}

		if ok, err := currentUser.HasPermission(ctx, authn.OpViewEmployee); err != nil {
			return errors.Wrap(err, "判断当前用户是否有权限失败")
		} else if !ok {
			return errors.NewOperationReject(authn.OpViewEmployee)
		}
		for _, employeeID := range toUsers {
			err := svc.syncToUser(ctx, currentUser, employeeID, password, createIfNotExist)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (svc employeeService) syncFromUser(ctx context.Context, currentUser authn.AuthUser, userID int64, createIfNotExist bool) error {
	user, err := svc.users.userDao.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	oldEmployee, err := svc.employeeDao.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) && createIfNotExist {
			return errors.Wrap(errors.ErrUnimplemented, "自动创建员工未实现")
		}

		// if ok, err := currentUser.HasPermission(ctx, authn.OpCreateEmployee); err != nil {
		// 	return errors.Wrap(err, "判断当前用户是否有权限失败")
		// } else if !ok {
		// 	return errors.NewOperationReject(authn.OpCreateEmployee)
		// }
		return err
	}

	newEmployee := *oldEmployee
	newEmployee.Name = user.Name
	newEmployee.Nickname = user.Nickname
	newEmployee.DepartmentID = user.DepartmentID

	return svc.db.InTx(ctx, nil, true, func(ctx context.Context, tx *gobatis.Tx) error {
		err = svc.employeeDao.UpdateByID(ctx, newEmployee.ID, &newEmployee)
		if err != nil {
			return err
		}

		svc.logUpdate(ctx, tx, currentUser, newEmployee.ID, &newEmployee, oldEmployee, actionSync, nil)
		return nil
	})
}

func (svc employeeService) syncToUser(ctx context.Context, currentUser authn.AuthUser, employeeID int64, password string, createIfNotExist bool) error {
	emp, err := svc.employeeDao.FindByID(ctx, employeeID)
	if err != nil {
		return err
	}

	if emp.UserID <= 0 {
		if !createIfNotExist {
			return errors.New("该员工没有关联可登录用户")
		}
		// if ok, err := currentUser.HasPermission(ctx, authn.OpCreateUser); err != nil {
		// 	return errors.Wrap(err, "判断当前用户是否有权限失败")
		// } else if !ok {
		// 	return errors.NewOperationReject(authn.OpCreateUser)
		// }
		return errors.Wrap(errors.ErrUnimplemented, "自动创建可登录用户未实现")
	}
	oldUser, err := svc.users.userDao.FindByID(ctx, emp.UserID)
	if err != nil {
		return err
	}

	newUser := *oldUser
	newUser.Name = emp.Name
	newUser.Nickname = emp.Nickname
	newUser.DepartmentID = emp.DepartmentID

	return svc.db.InTx(ctx, nil, true, func(ctx context.Context, tx *gobatis.Tx) error {
		return svc.users.update(ctx, currentUser, newUser.ID, &newUser, oldUser, booclient.UpdateModeSkip, actionSync)
	})
}

func (svc employeeService) GetUserEmployeeDiff(ctx context.Context) ([]booclient.UserEmployeeDiff, error) {
	return svc.employeeDao.GetUserEmployeeDiff(ctx)
}

func (svc employeeService) Export(ctx context.Context, format string, inline bool, writer http.ResponseWriter) error {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpViewEmployee); err != nil {
		return errors.Wrap(err, "判断当前用户是否有权限失败")
	} else if !ok {
		return errors.NewOperationReject(authn.OpViewEmployee)
	}

	return importer.WriteHTTP(ctx, "employeeDao", format, inline, writer,
		importer.RecorderFunc(func(ctx context.Context) (importer.RecordIterator, []string, error) {
			list, err := svc.employeeDao.List(ctx, 0, 0, "", "", sql.NullBool{Valid: true}, "", 0, 0)
			if err != nil {
				return nil, nil, err
			}
			titles := []string{
				"员工名",
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
						d, err := svc.departmentDao.FindByID(ctx, list[index].DepartmentID)
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
						values = append(values, list[index].GetString(f.ID))
					}
					values = append(values,
						formatTime(list[index].CreatedAt),
						formatTime(list[index].UpdatedAt))
					return values, nil
				},
			}, titles, nil
		}))
}

func (svc employeeService) Import(ctx context.Context, request *http.Request) error {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return err
	}

	canCreate := false
	if ok, err := currentUser.HasPermission(ctx, authn.OpCreateEmployee); err != nil {
		return errors.Wrap(err, "判断当前用户是否有权限失败")
	} else {
		canCreate = ok
	}

	canUpdate := false
	if ok, err := currentUser.HasPermission(ctx, authn.OpUpdateEmployee); err != nil {
		return errors.Wrap(err, "判断当前用户是否有权限失败")
	} else {
		canUpdate = ok
	}

	canCreateDepartment := false
	if ok, err := currentUser.HasPermission(ctx, authn.OpCreateDepartment); err != nil {
		return errors.Wrap(err, "判断当前用户是否有权限失败")
	} else {
		canCreateDepartment = ok
	}

	ctx = context.WithValue(ctx, importer.ContextToRealDirKey, booclient.ToRealDirFunc(svc.env))
	reader, closer, err := importer.ReadHTTP(ctx, request)
	if err != nil {
		return err
	}
	defer closer.Close()

	override := request.URL.Query().Get("override") == "true"
	departmentAutoCreate := request.URL.Query().Get("department_auto_create") == "true"

	return importer.Import(ctx, "", reader, func(ctx context.Context, lineNumber int) (importer.Row, error) {
		record := &Employee{}

		var columns = make([]importer.Column, 0, 5+len(svc.fields))
		columns = append(columns, importer.StrColumn([]string{"name", "用户", "用户名", "用户名称", "员工", "员工名", "员工名称"}, true,
			func(ctx context.Context, lineNumber int, origin, value string) error {
				record.Name = value
				return nil
			}))
		columns = append(columns, importer.StrColumn([]string{"zh_name", "中文名", "姓名", "呢称"}, false,
			func(ctx context.Context, lineNumber int, origin, value string) error {
				record.Nickname = value
				return nil
			}))

		for _, f := range svc.fields {
			func(f booclient.CustomField) {
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
				depart, err := svc.departmentDao.FindByName(ctx, value)
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
					id, err := svc.departmentDao.Insert(ctx, &Department{
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
				old, err := svc.employeeDao.FindByName(ctx, record.Name)
				if err != nil && !errors.IsNotFound(err) {
					return err
				}
				if old != nil {
					if override {
						err = errors.New("员工 '" + record.Name + "' 已存在")
					} else if canUpdate {
						err = svc.update(ctx, currentUser, old.ID, record, old, booclient.UpdateModeSkip, actionImport)
					} else {
						err = errors.New("没有更新员工的权限，员工 '" + record.Name + "' 没有更新")
					}
				} else {
					if canCreate {
						if record.Nickname == "" {
							record.Nickname = record.Name
						}
						_, err = svc.insert(ctx, currentUser, record, actionImport)
					} else {
						err = errors.New("没有新建员工的权限，员工 '" + record.Name + "' 没有创建")
					}
				}
				return err
			},
		}, nil
	})
}

func (svc employeeService) logCreate(ctx context.Context, tx *gobatis.Tx, currentUser authn.AuthUser, id int64, employee *Employee, importEmployee int, contents []ChangeRecord) {
	if !enableOplog {
		return
	}
	records := make([]ChangeRecord, 0, 10)
	if employee.DepartmentID > 0 {
		record := ChangeRecord{
			Name:        "department_id",
			DisplayName: "部门",
			NewValue:    employee.DepartmentID,
		}
		d, err := svc.departmentDao.FindByID(ctx, employee.DepartmentID)
		if err != nil {
			svc.logger.WarnContext(ctx, "查询部门失败", slog.Any("err", err))
		} else if d != nil {
			record.NewDisplayValue = d.Name
		}
		records = append(records, record)
	}
	records = append(records, ChangeRecord{
		Name:        "name",
		DisplayName: "员工名",
		NewValue:    employee.Name,
	}, ChangeRecord{
		Name:        "nickname",
		DisplayName: "呢称",
		NewValue:    employee.Nickname,
	}, ChangeRecord{
		Name:        "description",
		DisplayName: "描述",
		NewValue:    employee.Description,
	})
	for _, field := range svc.fields {
		fv := employee.Fields[field.ID]
		if fv == nil {
			continue
		}
		records = append(records, ChangeRecord{
			Name:        field.ID,
			DisplayName: field.Name,
			NewValue:    fv,
		})
	}

	if len(contents) > 0 {
		records = append(records, contents...)
	}

	typeStr := authn.OpCreateEmployee
	content := "创建员工成功"
	switch importEmployee {
	case actionNormal:
	case actionSync:
		typeStr = "synccreateemployee"
		content = "同步创建员工成功"
	case actionImport:
		typeStr = "importcreateemployee"
		content = "导入创建员工成功"
	}

	err := svc.operationLogger.WithTx(tx.DB()).LogRecord(ctx, &OperationLog{
		UserID:     currentUser.ID(),
		Username:   currentUser.Nickname(),
		Successful: true,
		Type:       typeStr,
		Content:    content,
		Fields: &OperationLogRecord{
			ObjectType: "employee",
			ObjectID:   id,
			Records:    records,
		},
	})
	if err != nil {
		svc.logger.WarnContext(ctx, "记录新建员工的操作失败", slog.Any("err", err))
	}
}

func (svc employeeService) logUpdate(ctx context.Context, tx *gobatis.Tx, currentUser authn.AuthUser, id int64, employee, old *Employee, importEmployee int, contents []ChangeRecord) {
	if !enableOplog {
		return
	}
	records := make([]ChangeRecord, 0, 10)
	if employee.DepartmentID != old.DepartmentID {
		var oldDepart, newDepart string
		if old.DepartmentID > 0 {
			d, err := svc.departmentDao.FindByID(ctx, old.DepartmentID)
			if err != nil {
				svc.logger.WarnContext(ctx, "查询部门失败", slog.Any("err", err))
			} else {
				oldDepart = d.Name
			}
		}
		if employee.DepartmentID > 0 {
			d, err := svc.departmentDao.FindByID(ctx, employee.DepartmentID)
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
			NewValue:        employee.DepartmentID,
			OldDisplayValue: oldDepart,
			NewDisplayValue: newDepart,
		})
	}

	if employee.Name != old.Name {
		records = append(records, ChangeRecord{
			Name:        "name",
			DisplayName: "员工名",
			OldValue:    old.Name,
			NewValue:    employee.Name,
		})
	}

	if employee.Nickname != old.Nickname {
		records = append(records, ChangeRecord{
			Name:        "nickname",
			DisplayName: "呢称",
			OldValue:    old.Nickname,
			NewValue:    employee.Nickname,
		})
	}

	if employee.Description != old.Description {
		records = append(records, ChangeRecord{
			Name:        "description",
			DisplayName: "描述",
			OldValue:    old.Description,
			NewValue:    employee.Description,
		})
	}

	for _, field := range svc.fields {
		var oldfv, newfv interface{}
		if len(old.Fields) > 0 {
			oldfv = old.Fields[field.ID]
		}
		if len(employee.Fields) > 0 {
			newfv = employee.Fields[field.ID]
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

	if len(contents) > 0 {
		records = append(records, contents...)
	}

	typeStr := authn.OpUpdateEmployee
	content := "更新员工成功"
	switch importEmployee {
	case actionNormal:
	case actionSync:
		typeStr = "syncupdateemployee"
		content = "同步更新员工成功"
	case actionImport:
		typeStr = "importupdateemployee"
		content = "导入更新员工成功"
	}
	err := svc.operationLogger.WithTx(tx.DB()).LogRecord(ctx, &OperationLog{
		UserID:     currentUser.ID(),
		Username:   currentUser.Nickname(),
		Successful: true,
		Type:       typeStr,
		Content:    content,
		Fields: &OperationLogRecord{
			ObjectType: "employee",
			ObjectID:   id,
			Records:    records,
		},
	})
	if err != nil {
		svc.logger.WarnContext(ctx, "记录更新员工的操作失败", slog.Any("err", err))
	}
}

func (svc employeeService) logBind(ctx context.Context, tx *gobatis.Tx, currentUser authn.AuthUser, employee *Employee) {
	if !enableOplog {
		return
	}
	err := svc.operationLogger.WithTx(tx.DB()).LogRecord(ctx, &OperationLog{
		UserID:     currentUser.ID(),
		Username:   currentUser.Nickname(),
		Successful: true,
		Type:       authn.OpUpdateEmployee,
		Content:    "绑定一个可登录用户",
		Fields: &OperationLogRecord{
			ObjectType: "employee",
			ObjectID:   employee.ID,
			Records:    []ChangeRecord{},
		},
	})
	if err != nil {
		svc.logger.WarnContext(ctx, "记录新建员工的操作失败", slog.Any("err", err))
	}
}

func (svc employeeService) logDelete(ctx context.Context, tx *gobatis.Tx, currentUser authn.AuthUser, oldEmployee *Employee) {
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
		Type:       authn.OpDeleteEmployee,
		Content:    "删除员工 '" + oldEmployee.Name + "' 成功",
		Fields: &OperationLogRecord{
			ObjectType: "employee",
			ObjectID:   oldEmployee.ID,
		},
	})
	if err != nil {
		svc.logger.WarnContext(ctx, "记录删除员工的操作失败", slog.Any("err", err))
	}
}

func NewEmployeeTags(env *booclient.Environment,
	db *gobatis.SessionFactory,
	operationLogger OperationLogger) (booclient.EmployeeTags, error) {
	sess := db.SessionReference()
	return &employeeTagService{
		env:             env,
		logger:          env.Logger.WithGroup("employees"),
		operationLogger: operationLogger,
		db:              db,
		employeeTagDao:  NewEmployeeTagDaoWith(sess),
		employee2TagDao:  NewEmployee2TagDaoWith(sess),
	}, nil
}

type employeeTagService struct {
	env             *booclient.Environment
	logger          *slog.Logger
	operationLogger OperationLogger
	db              *gobatis.SessionFactory

	employeeTagDao      EmployeeTagDao
	employee2TagDao     Employee2TagDao
}

func (svc *employeeTagService) fromTag(tag *EmployeeTag) *booclient.TagData {
	return &booclient.TagData{
		ID: tag.ID,
		UUID: tag.UUID,
		Title: tag.Title,
	}
}

func (svc *employeeTagService) toTag(tag *booclient.TagData) *EmployeeTag {
	return &EmployeeTag{
		ID: tag.ID,
		UUID: tag.UUID,
		Title: tag.Title,
	}
}

func (svc *employeeTagService) Create(ctx context.Context, tag *booclient.TagData) (int64, error) {
	return svc.employeeTagDao.Insert(ctx, svc.toTag(tag))
}

func (svc *employeeTagService) UpdateByID(ctx context.Context, id int64, tag *booclient.TagData) error {
	return svc.employeeTagDao.UpdateByID(ctx, id, svc.toTag(tag))
}

func (svc *employeeTagService) DeleteByID(ctx context.Context, id int64) error {
	if err := svc.employee2TagDao.DeleteByTagID(ctx, id); err != nil {
		return err
	}
	return svc.employeeTagDao.DeleteByID(ctx, id)
}

func (svc *employeeTagService) DeleteBatch(ctx context.Context, id []int64) error {
	for _, a := range id {
		if err := svc.employee2TagDao.DeleteByTagID(ctx, a); err != nil {
			return err
		}
		if err := svc.employeeTagDao.DeleteByID(ctx, a); err != nil {
			return err
		}
	}
	return nil
}

func (svc *employeeTagService) FindByID(ctx context.Context, id int64) (*booclient.TagData, error) {
	tag, err := svc.employeeTagDao.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return svc.fromTag(tag), nil
}

func (svc *employeeTagService) List(ctx context.Context, sort string, offset, limit int64) ([]booclient.TagData, error) {
	tags, err := svc.employeeTagDao.List(ctx, "", sort, offset, limit)
	if err != nil {
		return nil, err
	}

	var results = make([]booclient.TagData, 0, len(tags))
	for _, tag := range tags {
		results = append(results, *svc.fromTag(&tag))
	}
	return results, nil
}

func GetEmployeeAllIncludes() []string {
	return []string{
		"department",
		"tag",
	}
}