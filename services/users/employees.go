package users

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/boo-admin/boo/client"
	"github.com/boo-admin/boo/errors"
	"github.com/boo-admin/boo/goutils/importer"
	"github.com/boo-admin/boo/services/authn"
	"github.com/boo-admin/boo/validation"
	gobatis "github.com/runner-mei/GoBatis"
	"golang.org/x/exp/slog"

	"github.com/hjson/hjson-go/v4"
)

func NewEmployees(env *client.Environment,
	db *gobatis.SessionFactory,
	operationLogger OperationLogger) (Employees, error) {
	var fields []CustomField
	if s := env.Config.StringWithDefault("employeecustomfields", ""); s != "" {
		filename := client.GetRealDir(context.Background(), env, s)
		bs, err := ioutil.ReadFile(filename)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, errors.Wrap(err, "加载员工字段的配置失败")
			}
		} else if err = hjson.Unmarshal(bs, &fields); err != nil {
			return nil, errors.Wrap(err, "加载员工字段的配置失败")
		}

		if fields == nil {
			fields = client.DefaultFields
		}
	} else {
		fields = client.DefaultFields
	}

	sess := db.SessionReference()
	return employeeService{
		env:             env,
		logger:          env.Logger.WithGroup("employees"),
		operationLogger: operationLogger,
		db:              db,
		employeeDao:     NewEmployeeDao(sess),
		departments:     NewDepartmentDao(sess),
		fields:          fields,
	}, nil
}

type employeeService struct {
	env             *client.Environment
	logger          *slog.Logger
	operationLogger OperationLogger
	db              *gobatis.SessionFactory
	toRealDir       func(context.Context, string) string

	enablePasswordCheck bool
	employeeDao         EmployeeDao
	departments         DepartmentDao
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
	return svc.insert(ctx, currentUser, employee, false)
}

func (svc employeeService) insert(ctx context.Context, currentUser authn.AuthUser, employee *Employee, importEmployee bool) (int64, error) {
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

		svc.logCreate(ctx, tx, currentUser, id, employee, importEmployee)
		return nil
	})
	return id, err
}
func (svc employeeService) UpdateByID(ctx context.Context, id int64, employee *Employee) error {
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
	return svc.update(ctx, currentUser, id, employee, old, false)
}

func (svc employeeService) update(ctx context.Context, currentUser authn.AuthUser, id int64, employee, old *Employee, importEmployee bool) error {
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
		newEmployee.Disabled = employee.Disabled

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

		svc.logUpdate(ctx, tx, currentUser, id, &newEmployee, old, importEmployee)
		return nil
	})
}

func (svc employeeService) DeleteByID(ctx context.Context, id int64) error {
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

	err = svc.employeeDao.DeleteByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "删除员工失败")
	}

	svc.logDelete(ctx, nil, currentUser, old)
	return nil
}

func (svc employeeService) DeleteBatch(ctx context.Context, idlist []int64) error {
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
		err = svc.employeeDao.DeleteByIDList(ctx, newList)
		if err != nil {
			return errors.Wrap(err, "删除员工失败")
		}
		for _, old := range oldList {
			svc.logDelete(ctx, tx, currentUser, &old)
		}
		return nil
	})
}

func (svc employeeService) FindByID(ctx context.Context, id int64) (*Employee, error) {
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

	return svc.employeeDao.FindByID(ctx, id)
}
func (svc employeeService) FindByName(ctx context.Context, name string) (*Employee, error) {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if ok, err := currentUser.HasPermission(ctx, authn.OpViewEmployee); err != nil {
		return nil, errors.Wrap(err, "判断当前用户是否有权限失败")
	} else if !ok {
		return nil, errors.NewOperationReject(authn.OpViewEmployee)
	}

	return svc.employeeDao.FindByName(ctx, name)
}
func (svc employeeService) Count(ctx context.Context, keyword string) (int64, error) {
	return svc.employeeDao.Count(ctx, keyword)
}
func (svc employeeService) List(ctx context.Context, keyword string, sort string, offset, limit int64) ([]Employee, error) {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpViewEmployee); err != nil {
		return nil, errors.Wrap(err, "判断当前用户是否有权限失败")
	} else if !ok {
		return nil, errors.NewOperationReject(authn.OpViewEmployee)
	}

	return svc.employeeDao.List(ctx, keyword, sort, offset, limit)
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
			list, err := svc.employeeDao.List(ctx, "", "", 0, 0)
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

	ctx = context.WithValue(ctx, importer.ContextToRealDirKey, svc.toRealDir)
	reader, closer, err := importer.ReadHTTP(ctx, request)
	if err != nil {
		return err
	}
	defer closer.Close()

	departmentAutoCreate := request.URL.Query().Get("department_auto_create") == "true"

	return importer.Import(ctx, "", reader, func(ctx context.Context, lineNumber int) (importer.Row, error) {
		record := &Employee{}

		var columns = make([]importer.Column, 0, 5+len(svc.fields))
		columns = append(columns, importer.StrColumn([]string{"name", "用户", "用户名", "用户名称"}, true,
			func(ctx context.Context, lineNumber int, origin, value string) error {
				record.Name = value
				return nil
			}))
		columns = append(columns, importer.StrColumn([]string{"zh_name", "中文名", "姓名"}, false,
			func(ctx context.Context, lineNumber int, origin, value string) error {
				record.Nickname = value
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
				old, err := svc.employeeDao.FindByName(ctx, record.Name)
				if err != nil && !errors.IsNotFound(err) {
					return err
				}
				if old != nil {
					if canUpdate {
						err = svc.update(ctx, currentUser, old.ID, record, old, true)
					} else {
						err = errors.New("没有更新员工的权限，员工 '" + record.Name + "' 没有更新")
					}
				} else {
					if canCreate {
						if record.Nickname == "" {
							record.Nickname = record.Name
						}
						_, err = svc.insert(ctx, currentUser, record, true)
					} else {
						err = errors.New("没有新建员工的权限，员工 '" + record.Name + "' 没有创建")
					}
				}
				return err
			},
		}, nil
	})
}

func (svc employeeService) logCreate(ctx context.Context, tx *gobatis.Tx, currentUser authn.AuthUser, id int64, employee *Employee, importEmployee bool) {
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
		d, err := svc.departments.FindByID(ctx, employee.DepartmentID)
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

	typeStr := authn.OpCreateEmployee
	content := "创建员工成功"
	if importEmployee {
		typeStr = "importuser"
		content = "导入员工成功"
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

func (svc employeeService) logUpdate(ctx context.Context, tx *gobatis.Tx, currentUser authn.AuthUser, id int64, employee, old *Employee, importEmployee bool) {
	if !enableOplog {
		return
	}
	records := make([]ChangeRecord, 0, 10)
	if employee.DepartmentID != old.DepartmentID {
		var oldDepart, newDepart string
		if old.DepartmentID > 0 {
			d, err := svc.departments.FindByID(ctx, old.DepartmentID)
			if err != nil {
				svc.logger.WarnContext(ctx, "查询部门失败", slog.Any("err", err))
			} else {
				oldDepart = d.Name
			}
		}
		if employee.DepartmentID > 0 {
			d, err := svc.departments.FindByID(ctx, employee.DepartmentID)
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

	typeStr := authn.OpUpdateEmployee
	if importEmployee {
		typeStr = "importuser"
	}
	err := svc.operationLogger.WithTx(tx.DB()).LogRecord(ctx, &OperationLog{
		UserID:     currentUser.ID(),
		Username:   currentUser.Nickname(),
		Successful: true,
		Type:       typeStr,
		Content:    "更新员工成功",
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
