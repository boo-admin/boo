package users

import (
	"context"
	"fmt"
)

// Option 用户选项
type Option interface {
	apply(opts *InternalOptions)
}

type userIncludeDisabled struct{}

func (u userIncludeDisabled) apply(opts *InternalOptions) {
	opts.UserIncludeDisabled = true
}

// UserIncludeDisabled 禁用的用户也返回
func UserIncludeDisabled() Option {
	return userIncludeDisabled{}
}

// 内部使用，不可补外部使用，它不应该被导出，但因为要兼容我的老程
// 序，不得不将它写成 public
type InternalOptions struct {
	UserIncludeDisabled bool
}

// 内部使用，不可补外部使用，它不应该被导出，但因为要兼容我的老程
// 序，不得不将它写成 public
func InternalApply(opts ...Option) InternalOptions {
	var o InternalOptions
	for _, opt := range opts {
		opt.apply(&o)
	}
	return o
}

type WelcomeLocator interface {
	Locate(ctx context.Context, userID interface{}, username, defaultURL string) (string, error)
}

type WelcomeLocatorFunc func(ctx context.Context, userID interface{}, username, defaultURL string) (string, error)

func (f WelcomeLocatorFunc) Locate(ctx context.Context, userID interface{}, username, defaultURL string) (string, error) {
	return f(ctx, userID, username, defaultURL)
}

// UserManager 用户管理
type UserManager interface {
	UserByName(ctx context.Context, username string, opts ...Option) (User, error)
	UserByID(ctx context.Context, userID int64, opts ...Option) (User, error)
}

type createUserInLogin string

func (createUserInLogin) String() string {
	return "boo-create-user-in-login-key"
}

const createUserInLoginKey = createUserInLogin("boo-create-user-in-login-key")

func ContextWithCreateUserInLogin(ctx context.Context) context.Context {
	return context.WithValue(ctx, createUserInLoginKey, struct{}{})
}

func IsCreateUserInLoginFromContext(ctx context.Context) bool {
	o := ctx.Value(createUserInLoginKey)
	return o != nil
}

type UserPasswordHasher interface {
	Hash(context.Context, string) (string, error)
}

type UserPasswordComparer interface {
	Compare(ctx context.Context, password, hashedPassword string) error
}

type UserPassworder interface {
	UserPasswordHasher
	UserPasswordComparer
}

type UserPasswordHasherFunc func(context.Context, string) (string, error)

func (f UserPasswordHasherFunc) Hash(ctx context.Context, s string) (string, error) {
	return f(ctx, s)
}

type UserPasswordComparerFunc func(ctx context.Context, password, hashedPassword string) error

func (f UserPasswordComparerFunc) Compare(ctx context.Context, password, hashedPassword string) error {
	return f(ctx, password, hashedPassword)
}

type StringerFunc func() string

func (fn StringerFunc) String() string {
	return fn()
}

var _ fmt.Stringer = StringerFunc(func() string { return "" })
