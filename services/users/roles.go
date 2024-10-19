package users

import (
	"context"
	"strconv"

	"github.com/boo-admin/boo/client"
	"github.com/boo-admin/boo/errors"
	"github.com/boo-admin/boo/goutils/tid"
	"github.com/boo-admin/boo/services/authn"
	"github.com/boo-admin/boo/validation"
	gobatis "github.com/runner-mei/GoBatis"
	"golang.org/x/exp/slog"
)

var NewRoleDaoHook func(ref gobatis.SqlSession) RoleDao

func NewRoleDaoWith(ref gobatis.SqlSession) RoleDao {
	if NewRoleDaoHook != nil {
		return NewRoleDaoHook(ref)
	}
	return NewRoleDao(ref)
}

func NewRoles(env *client.Environment,
	db *gobatis.SessionFactory,
	operationLogger OperationLogger) (client.Roles, error) {
	return roleService{
		env:             env,
		logger:          env.Logger.WithGroup("roles"),
		operationLogger: operationLogger,
		dao:             NewRoleDaoWith(db.SessionReference()),
	}, nil
}

type roleService struct {
	env             *client.Environment
	logger          *slog.Logger
	operationLogger OperationLogger
	// db              *gobatis.SessionFactory
	dao RoleDao
}

func (svc roleService) Create(ctx context.Context, role *Role) (int64, error) {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return 0, err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpCreateRole); err != nil {
		return 0, errors.Wrap(err, "判断当前角色是否有权限失败")
	} else if !ok {
		return 0, errors.NewOperationReject(authn.OpCreateRole)
	}

	v := validation.Default.New()
	if role.UUID == "" {
		role.UUID = tid.GenerateID()
	} else if exists, err := svc.dao.UUIDExists(ctx, role.UUID); err != nil {
		return 0, errors.Wrap(err, "查询角色名 '"+role.UUID+"' 是否已存在失败")
	} else if exists {
		v.Error("uuid", "无法新建角色 '"+role.UUID+"'，该角色已存在")
	}
	if role.Title == "" {
		v.Error("name", "无法新建角色 '"+role.Title+"'，该角色名为空")
	} else if exists, err := svc.dao.TitleExists(ctx, role.Title); err != nil {
		return 0, errors.Wrap(err, "查询角色名 '"+role.Title+"' 是否已存在失败")
	} else if exists {
		v.Error("name", "无法新建角色 '"+role.Title+"'，该角色已存在")
	}
	if v.HasErrors() {
		return 0, v.ToError()
	}

	id, err := svc.dao.Insert(ctx, role)
	if err != nil {
		return 0, err
	}

	svc.logCreate(ctx, nil, currentUser, id, role)
	return id, nil
}

func (svc roleService) UpdateByID(ctx context.Context, id int64, role *Role) error {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpUpdateRole); err != nil {
		return errors.Wrap(err, "判断当前角色是否有权限失败")
	} else if !ok {
		return errors.NewOperationReject(authn.OpUpdateRole)
	}

	v := validation.Default.New()
	if role.Title == "" {
		v.Error("name", "无法更新角色 '"+role.Title+"'，该角色名为空")
	}
	if v.HasErrors() {
		return v.ToError()
	}

	old, err := svc.dao.FindByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "更新角色 '"+strconv.FormatInt(id, 10)+"' 失败")
	}

	if role.UUID != "" && role.UUID != old.UUID {
		v.Error("name", "无法更新角色 '"+role.Title+"'，UUID 不可修改")
	}
	role.UUID = old.UUID

	if role.Title != old.Title {
		if exists, err := svc.dao.TitleExists(ctx, role.Title); err != nil {
			return errors.Wrap(err, "查询角色名 '"+role.Title+"' 是否已存在失败")
		} else if exists {
			v.Error("title", "无法更新角色 '"+role.Title+"'，该角色的新名称已经存在")
		}
		if v.HasErrors() {
			return v.ToError()
		}
	}

	err = svc.dao.UpdateByID(ctx, id, role)
	if err != nil {
		return err
	}

	svc.logUpdate(ctx, nil, currentUser, id, role, old)
	return nil
}
func (svc roleService) DeleteByID(ctx context.Context, id int64) error {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpDeleteRole); err != nil {
		return errors.Wrap(err, "判断当前角色是否有权限失败")
	} else if !ok {
		return errors.NewOperationReject(authn.OpDeleteRole)
	}

	old, err := svc.dao.FindByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "删除角色时，查询角色 '"+strconv.FormatInt(id, 10)+"' 失败")
	}

	err = svc.dao.DeleteByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "删除角色失败")
	}

	svc.logDelete(ctx, nil, currentUser, old)
	return nil
}
func (svc roleService) FindByID(ctx context.Context, id int64) (*Role, error) {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if currentUser.ID() != id {
		if ok, err := currentUser.HasPermission(ctx, authn.OpViewRole); err != nil {
			return nil, errors.Wrap(err, "判断当前角色是否有权限失败")
		} else if !ok {
			return nil, errors.NewOperationReject(authn.OpViewRole)
		}
	}

	return svc.dao.FindByID(ctx, id)
}
func (svc roleService) FindByUUID(ctx context.Context, uuid string) (*Role, error) {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpViewRole); err != nil {
		return nil, errors.Wrap(err, "判断当前角色是否有权限失败")
	} else if !ok {
		return nil, errors.NewOperationReject(authn.OpViewRole)
	}

	return svc.dao.FindByUUID(ctx, uuid)
}
func (svc roleService) Count(ctx context.Context, keyword string) (int64, error) {
	return svc.dao.Count(ctx, keyword)
}
func (svc roleService) List(ctx context.Context, keyword string, sort string, offset, limit int64) ([]Role, error) {
	currentUser, err := authn.ReadUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if ok, err := currentUser.HasPermission(ctx, authn.OpViewRole); err != nil {
		return nil, errors.Wrap(err, "判断当前角色是否有权限失败")
	} else if !ok {
		return nil, errors.NewOperationReject(authn.OpViewRole)
	}

	return svc.dao.List(ctx, keyword, sort, offset, limit)
}

func (svc roleService) logCreate(ctx context.Context, tx *gobatis.Tx, currentUser authn.AuthUser, id int64, role *Role) {
	records := make([]ChangeRecord, 0, 10)
	records = append(records, ChangeRecord{
		Name:        "uuid",
		DisplayName: "角色编号",
		NewValue:    role.Title,
	})
	records = append(records, ChangeRecord{
		Name:        "title",
		DisplayName: "角色名称",
		NewValue:    role.Title,
	})
	records = append(records, ChangeRecord{
		Name:        "description",
		DisplayName: "角色描述",
		NewValue:    role.Description,
	})
	// for _, field := range svc.fields {
	// 	fv, _ := role.Fields[field.ID]
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
		Type:       authn.OpCreateRole,
		Content:    "创建角色成功",
		Fields: &OperationLogRecord{
			ObjectType: "user",
			ObjectID:   id,
			Records:    records,
		},
	})
	if err != nil {
		svc.logger.WarnContext(ctx, "记录新建角色的操作失败", slog.Any("err", err))
	}
}

func (svc roleService) logUpdate(ctx context.Context, tx *gobatis.Tx, currentUser authn.AuthUser, id int64, role, old *Role) {
	records := make([]ChangeRecord, 0, 10)
	if role.Title != old.Title {
		records = append(records, ChangeRecord{
			Name:        "title",
			DisplayName: "角色名",
			OldValue:    old.Title,
			NewValue:    role.Title,
		})
	}
	if role.Description != old.Description {
		records = append(records, ChangeRecord{
			Name:        "description",
			DisplayName: "角色描述",
			OldValue:    old.Description,
			NewValue:    role.Description,
		})
	}

	// for _, field := range svc.fields {
	// 	var oldfv, newfv interface{}
	// 	if len(old.Fields) > 0 {
	// 		oldfv = old.Fields[field.ID]
	// 	}
	// 	if len(role.Fields) > 0 {
	// 		newfv = role.Fields[field.ID]
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
		Type:       authn.OpUpdateRole,
		Content:    "更新角色成功",
		Fields: &OperationLogRecord{
			ObjectType: "role",
			ObjectID:   id,
			Records:    records,
		},
	})
	if err != nil {
		svc.logger.WarnContext(ctx, "记录更新角色的操作失败", slog.Any("err", err))
	}
}

func (svc roleService) logDelete(ctx context.Context, tx *gobatis.Tx, currentUser authn.AuthUser, oldRole *Role) {
	oplogger := svc.operationLogger
	if tx != nil {
		oplogger = oplogger.WithTx(tx.DB())
	}
	err := oplogger.LogRecord(ctx, &OperationLog{
		UserID:     currentUser.ID(),
		Username:   currentUser.Nickname(),
		Successful: true,
		Type:       authn.OpDeleteRole,
		Content:    "删除角色 '" + oldRole.Title + "' 成功",
		Fields: &OperationLogRecord{
			ObjectType: "role",
			ObjectID:   oldRole.ID,
		},
	})
	if err != nil {
		svc.logger.WarnContext(ctx, "记录更新角色的操作失败", slog.Any("err", err))
	}
}
