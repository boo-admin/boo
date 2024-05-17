package users

import (
	"context"
	"sort"
	"strconv"

	"github.com/boo-admin/boo/client"
	"github.com/boo-admin/boo/errors"
	"github.com/boo-admin/boo/services/authn"
	"github.com/boo-admin/boo/validation"
	gobatis "github.com/runner-mei/GoBatis"
	"golang.org/x/exp/slog"
)

func NewDepartments(logger *slog.Logger,
	params map[string]string,
	db *gobatis.SessionFactory,
	operationLogger OperationLogger,
	toRealDir func(context.Context, string) string) (client.Departments, error) {
	return departmentService{
		logger:          logger,
		operationLogger: operationLogger,
		dao:             NewDepartmentDao(db.SessionReference()),
	}, nil
}

type departmentService struct {
	logger          *slog.Logger
	operationLogger OperationLogger
	// db              *gobatis.SessionFactory
	dao DepartmentDao
}

func (svc departmentService) Insert(ctx context.Context, department *Department) (int64, error) {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return 0, err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpCreateDepartment); err != nil {
		return 0, errors.Wrap(err, "判断当前部门是否有权限失败")
	} else if !ok {
		return 0, errors.NewOperationReject(authn.OpCreateDepartment)
	}

	v := validation.Default.New()
	if department.Name == "" {
		v.Error("name", "无法新建部门 '"+department.Name+"'，该部门名为空")
	} else if exists, err := svc.dao.NameExists(ctx, department.Name); err != nil {
		return 0, errors.Wrap(err, "查询部门名 '"+department.Name+"' 是否已存在失败")
	} else if exists {
		v.Error("name", "无法新建部门 '"+department.Name+"'，该部门已存在")
	}
	if v.HasErrors() {
		return 0, v.ToError()
	}

	id, err := svc.dao.Insert(ctx, department)
	if err != nil {
		return 0, err
	}

	svc.logCreate(ctx, nil, currentUser, id, department)
	return id, nil
}

func (svc departmentService) UpdateByID(ctx context.Context, id int64, department *Department) error {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpUpdateDepartment); err != nil {
		return errors.Wrap(err, "判断当前部门是否有权限失败")
	} else if !ok {
		return errors.NewOperationReject(authn.OpUpdateDepartment)
	}

	v := validation.Default.New()
	if department.Name == "" {
		v.Error("name", "无法新建部门 '"+department.Name+"'，该部门名为空")
	}
	if v.HasErrors() {
		return v.ToError()
	}

	old, err := svc.dao.FindByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "更新部门 '"+strconv.FormatInt(id, 10)+"' 失败")
	}
	if department.Name != old.Name {
		if exists, err := svc.dao.NameExists(ctx, department.Name); err != nil {
			return errors.Wrap(err, "查询部门名 '"+department.Name+"' 是否已存在失败")
		} else if exists {
			v.Error("name", "无法新建部门 '"+department.Name+"'，该部门已存在")
		}
		if v.HasErrors() {
			return v.ToError()
		}
	}

	err = svc.dao.UpdateByID(ctx, id, department)
	if err != nil {
		return err
	}

	svc.logUpdate(ctx, nil, currentUser, id, department, old)
	return nil
}
func (svc departmentService) DeleteByID(ctx context.Context, id int64) error {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpDeleteDepartment); err != nil {
		return errors.Wrap(err, "判断当前部门是否有权限失败")
	} else if !ok {
		return errors.NewOperationReject(authn.OpDeleteDepartment)
	}

	old, err := svc.dao.FindByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "删除部门时，查询部门 '"+strconv.FormatInt(id, 10)+"' 失败")
	}

	err = svc.dao.DeleteByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "删除部门失败")
	}

	svc.logDelete(ctx, nil, currentUser, old)
	return nil
}
func (svc departmentService) FindByID(ctx context.Context, id int64) (*Department, error) {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if currentUser.ID() != id {
		if ok, err := currentUser.HasPermission(ctx, authn.OpViewDepartment); err != nil {
			return nil, errors.Wrap(err, "判断当前部门是否有权限失败")
		} else if !ok {
			return nil, errors.NewOperationReject(authn.OpViewDepartment)
		}
	}

	return svc.dao.FindByID(ctx, id)
}
func (svc departmentService) FindByName(ctx context.Context, name string) (*Department, error) {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpViewDepartment); err != nil {
		return nil, errors.Wrap(err, "判断当前部门是否有权限失败")
	} else if !ok {
		return nil, errors.NewOperationReject(authn.OpViewDepartment)
	}

	return svc.dao.FindByName(ctx, name)
}
func (svc departmentService) Count(ctx context.Context) (int64, error) {
	return svc.dao.Count(ctx)
}
func (svc departmentService) List(ctx context.Context) ([]Department, error) {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpViewDepartment); err != nil {
		return nil, errors.Wrap(err, "判断当前部门是否有权限失败")
	} else if !ok {
		return nil, errors.NewOperationReject(authn.OpViewDepartment)
	}

	return svc.dao.List(ctx)
}
func (svc departmentService) GetTree(ctx context.Context) ([]*Department, error) {
	results, err := svc.List(ctx)
	if err != nil {
		return nil, err
	}
	return toDepartmentsTree(results), nil
}

func toDepartmentsTree(list []Department) []*Department {
	byID := map[int64]*Department{}
	for idx := range list {
		byID[list[idx].ID] = &list[idx]
	}

	var roots []*Department
	for idx := range list {
		if list[idx].ParentID <= 0 {
			roots = append(roots, &list[idx])

			sort.Slice(roots, func(i, j int) bool {
				return roots[i].OrderNum < roots[j].OrderNum
			})
			continue
		}
		root := byID[list[idx].ParentID]
		if root == nil {
			roots = append(roots, &list[idx])
			continue
		}
		root.Children = append(root.Children, &list[idx])

		sort.Slice(root.Children, func(i, j int) bool {
			return root.Children[i].OrderNum < root.Children[j].OrderNum
		})
	}
	return roots
}

func (svc departmentService) logCreate(ctx context.Context, tx *gobatis.Tx, currentUser authn.AuthUser, id int64, department *Department) {
	records := make([]ChangeRecord, 0, 10)
	if department.ParentID > 0 {
		record := ChangeRecord{
			Name:        "parent_id",
			DisplayName: "部门",
			NewValue:    department.ParentID,
		}
		d, err := svc.dao.FindByID(ctx, department.ParentID)
		if err != nil {
			svc.logger.WarnContext(ctx, "查询部门失败", slog.Any("err", err))
		} else if d != nil {
			record.NewDisplayValue = d.Name
		}
		records = append(records, record)
	}
	records = append(records, ChangeRecord{
		Name:        "name",
		DisplayName: "部门名称",
		NewValue:    department.Name,
	})
	// for _, field := range svc.fields {
	// 	fv, _ := department.Fields[field.ID]
	// 	if fv == nil {
	// 		continue
	// 	}
	// 	records = append(records, ChangeRecord{
	// 		Name:        field.ID,
	// 		DisplayName: field.Name,
	// 		NewValue:    fv,
	// 	})
	// }

	oplogger := svc.operationLogger
	if tx != nil {
		oplogger = oplogger.WithTx(tx.DB())
	}
	err := oplogger.LogRecord(ctx, &OperationLog{
		UserID:     currentUser.ID(),
		Username:   currentUser.Nickname(),
		Successful: true,
		Type:       authn.OpCreateDepartment,
		Content:    "创建部门成功",
		Fields: &OperationLogRecord{
			ObjectType: "user",
			ObjectID:   id,
			Records:    records,
		},
	})
	if err != nil {
		svc.logger.WarnContext(ctx, "记录新建部门的操作失败", slog.Any("err", err))
	}
}

func (svc departmentService) logUpdate(ctx context.Context, tx *gobatis.Tx, currentUser authn.AuthUser, id int64, department, old *Department) {
	records := make([]ChangeRecord, 0, 10)
	if department.ParentID != old.ParentID {
		var oldDepart, newDepart string
		if old.ParentID > 0 {
			d, err := svc.dao.FindByID(ctx, old.ParentID)
			if err != nil {
				svc.logger.WarnContext(ctx, "查询部门失败", slog.Any("err", err))
			} else {
				oldDepart = d.Name
			}
		}
		if department.ParentID > 0 {
			d, err := svc.dao.FindByID(ctx, department.ParentID)
			if err != nil {
				svc.logger.WarnContext(ctx, "查询部门失败", slog.Any("err", err))
			} else {
				newDepart = d.Name
			}
		}
		records = append(records, ChangeRecord{
			Name:            "parent_id",
			DisplayName:     "部门",
			OldValue:        old.ParentID,
			NewValue:        department.ParentID,
			OldDisplayValue: oldDepart,
			NewDisplayValue: newDepart,
		})
	}

	if department.Name != old.Name {
		records = append(records, ChangeRecord{
			Name:        "name",
			DisplayName: "部门名",
			OldValue:    old.Name,
			NewValue:    department.Name,
		})
	}

	// for _, field := range svc.fields {
	// 	var oldfv, newfv interface{}
	// 	if len(old.Fields) > 0 {
	// 		oldfv = old.Fields[field.ID]
	// 	}
	// 	if len(department.Fields) > 0 {
	// 		newfv = department.Fields[field.ID]
	// 	}
	// 	if oldfv == nil && newfv == nil {
	// 		continue
	// 	}
	// 	if oldfv != nil && newfv != nil {
	// 		if fmt.Sprint(oldfv) == fmt.Sprint(newfv) {
	// 			continue
	// 		}
	// 	}
	//
	// 	records = append(records, ChangeRecord{
	// 		Name:        field.ID,
	// 		DisplayName: field.Name,
	// 		OldValue:    oldfv,
	// 		NewValue:    newfv,
	// 	})
	// }

	oplogger := svc.operationLogger
	if tx != nil {
		oplogger = oplogger.WithTx(tx.DB())
	}
	err := oplogger.LogRecord(ctx, &OperationLog{
		UserID:     currentUser.ID(),
		Username:   currentUser.Nickname(),
		Successful: true,
		Type:       authn.OpUpdateDepartment,
		Content:    "更新部门成功",
		Fields: &OperationLogRecord{
			ObjectType: "department",
			ObjectID:   id,
			Records:    records,
		},
	})
	if err != nil {
		svc.logger.WarnContext(ctx, "记录更新部门的操作失败", slog.Any("err", err))
	}
}

func (svc departmentService) logDelete(ctx context.Context, tx *gobatis.Tx, currentUser authn.AuthUser, oldDepartment *Department) {
	oplogger := svc.operationLogger
	if tx != nil {
		oplogger = oplogger.WithTx(tx.DB())
	}
	err := oplogger.LogRecord(ctx, &OperationLog{
		UserID:     currentUser.ID(),
		Username:   currentUser.Nickname(),
		Successful: true,
		Type:       authn.OpDeleteDepartment,
		Content:    "删除部门 '" + oldDepartment.Name + "' 成功",
		Fields: &OperationLogRecord{
			ObjectType: "department",
			ObjectID:   oldDepartment.ID,
		},
	})
	if err != nil {
		svc.logger.WarnContext(ctx, "记录更新部门的操作失败", slog.Any("err", err))
	}
}
