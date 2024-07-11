package authn

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"

	"github.com/boo-admin/boo/errors"
)

// AuthUser 用户信息
type AuthUser interface {
	ID() int64

	// 用户登录名
	Name() string

	// 呢称
	Nickname() string

	// 显示名称
	DisplayName(ctx context.Context, fmt ...string) string

	// Profile 是用于保存用户在界面上的一些个性化数据
	// WriteProfile 保存 profiles
	WriteProfile(key, value string) error

	// Profile 是用于保存用户在界面上的一些个性化数据
	// ReadProfile 读 profiles
	ReadProfile(key string) (string, error)

	// 用户扩展属性
	Data(ctx context.Context, key string) interface{}

	// 用户角色ID列表
	RoleIDs() []int64

	// 用户角色名称列表
	RoleNames() []string

	// 用户是否有指定的权限
	HasPermission(ctx context.Context, permissionID string) (bool, error)

	// 用户是否有指定的权限
	HasPermissionAny(ctx context.Context, permissionIDs []string) (bool, error)

	// 是不是有一个指定的角色
	HasRole(string) bool

	// 是不是有一个指定的角色
	HasRoleID(id int64) bool

	// // 本用户是不是指定的用户组的成员
	// IsMemberOf(int64) bool

	// 用户属性
	ForEach(func(string, interface{}))
}

type userKey string

func (s userKey) constUserKey() {} // nolint:unused

const UserKey = userKey("boo-user-key")

type ReadCurrentUserFunc func(context.Context) (AuthUser, error)

func ContextWithUser(ctx context.Context, u AuthUser) context.Context {
	return context.WithValue(ctx, UserKey, u)
}

func ContextWithReadCurrentUser(ctx context.Context, u ReadCurrentUserFunc) context.Context {
	return context.WithValue(ctx, UserKey, u)
}

func UserFromContext(ctx context.Context) ReadCurrentUserFunc {
	o := ctx.Value(UserKey)
	if o == nil {
		return nil
	}
	f, _ := o.(ReadCurrentUserFunc)
	return f
}

func ReadUserFromContext(ctx context.Context) (AuthUser, error) {
	o := ctx.Value(UserKey)
	if o == nil {
		return nil, errors.Wrap(errors.ErrUnauthorized, "user isnot exists because session is unauthorized")
	}
	f, ok := o.(ReadCurrentUserFunc)
	if ok {
		return f(ctx)
	}
	u, ok := o.(AuthUser)
	if ok {
		return u, nil
	}
	return nil, fmt.Errorf("user is unknown type - %T", o)
}

func UserIDFromContext(ctx context.Context) (int64, error) {
	u, err := ReadUserFromContext(ctx)
	if err != nil {
		return 0, err
	}
	if u == nil {
		return 0, errors.ErrNotFound
	}
	return u.ID(), nil
}

func UsernameFromContext(ctx context.Context) (string, error) {
	u, err := ReadUserFromContext(ctx)
	if err != nil {
		return "", err
	}
	if u == nil {
		return "", errors.ErrNotFound
	}
	return u.Name(), nil
}

func UsernicknameFromContext(ctx context.Context) (string, error) {
	u, err := ReadUserFromContext(ctx)
	if err != nil {
		return "", err
	}
	if u == nil {
		return "", errors.ErrNotFound
	}
	return u.Nickname(), nil
}

const (
	OpCreateUser    = "createuser"
	OpUpdateUser    = "updateuser"
	OpResetPassword = "resetpassword"
	OpDeleteUser    = "deleteuser"
	OpViewUser      = "viewuser"

	OpUpdateDepartment = "updatedepartment"
	OpCreateDepartment = "createdepartment"
	OpDeleteDepartment = "deletedepartment"
	OpViewDepartment   = "viewdepartment"

	OpCreateEmployee = "createemployee"
	OpUpdateEmployee = "updateemployee"
	OpDeleteEmployee = "deleteemployee"
	OpViewEmployee   = "viewemployee"

	OpUpdateRole = "updateRole"
	OpCreateRole = "createRole"
	OpDeleteRole = "deleteRole"
	OpViewRole   = "viewRole"
)

func GetHash(alg string) (func() hash.Hash, error) {
	switch alg {
	case "":
		return nil, nil
	case "md5":
		return md5.New, nil
	case "sha1":
		return sha1.New, nil
	case "sha256":
		return sha256.New, nil
	case "sha512":
		return sha512.New, nil
	default:
		return getOtherHash(alg)
	}
}
