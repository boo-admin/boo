package authncore

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
)

type User interface {
	HasLock

	HasSource

	HasWhitelist

	HasRoles
}

type HasRoles interface {
	RoleNames() []string
}

// // LockedUser 被锁定的用户
// type LockedUser struct {
// 	Name     string
// 	LockedAt time.Time
// }

// UserManager 读用户配置的 Handler
type UserManager interface {
	Create(ctx context.Context, name, nickname, source, password string, fields map[string]interface{}, roles []string, skipIfRoleNotExists bool) (interface{}, error)

	Read(*AuthContext) (interface{}, User, error)
	// Lock(*AuthContext) error
	// Locked(ctx context.Context) ([]LockedUser, error)
	// Unlock(*AuthContext) error
}

type LoginResult struct {
	IsOK              bool
	SessionID         string
	IsNewUser         bool
	IsPasswordExpired bool

	Data map[string]interface{}
}

type LoginType int

const (
	TokenNone LoginType = iota
	TokenJWT
)

func (t LoginType) String() string {
	switch t {
	case TokenNone:
		return "none"
	case TokenJWT:
		return "jwt"
	default:
		return "LoginType-" + strconv.Itoa(int(t))
	}
}

type LoginRequest struct {
	UserID       interface{} `json:"userid" xml:"userid" form:"-" query:"-"`
	Username     string      `json:"username" xml:"username" form:"username" query:"username"`
	Password     string      `json:"password" xml:"password" form:"password" query:"password"`
	Service      string      `json:"service" xml:"service" form:"service" query:"service"`
	ForceLogin   string      `json:"force,omitempty" xml:"force" form:"force" query:"force"`
	CaptchaKey   string      `json:"captcha_key,omitempty" xml:"captcha_key" form:"captcha_key" query:"captcha_key"`
	CaptchaValue string      `json:"captcha_value,omitempty" xml:"captcha_value" form:"captcha_value" query:"captcha_value"`

	Address   string
	LoginType LoginType
}

func (u *LoginRequest) IsForce() bool {
	u.ForceLogin = strings.ToLower(u.ForceLogin)
	return u.ForceLogin == "on" ||
		u.ForceLogin == "true" ||
		u.ForceLogin == "checked"
}

type AuthStep int

const (
	BeforeLoad AuthStep = iota
	Loading
	AfterLoaded
	BeforeAuth
	Authing
	AfterAuthed
)

type AuthContext struct {
	Logger   *slog.Logger
	Ctx      context.Context
	Step     AuthStep
	Request  LoginRequest
	Response LoginResult

	SkipCaptcha    bool
	Authentication interface{}
	ErrorCount     int
}

type AuthFunc func(*AuthContext) error

type AuthService struct {
	beforeLoadFuncs []AuthFunc
	loadFuncs       []func(*AuthContext) (interface{}, interface{}, error)
	afterLoadFuncs  []AuthFunc
	beforeAuthFuncs []AuthFunc
	authFuncs       []func(*AuthContext) (bool, error)
	afterAuthFuncs  []AuthFunc
	errFuncs        []func(ctx *AuthContext, err error) error
}

func (as *AuthService) OnBeforeLoad(cb AuthFunc) {
	as.beforeLoadFuncs = append(as.beforeLoadFuncs, cb)
}
func (as *AuthService) OnLoad(cb func(*AuthContext) (interface{}, interface{}, error)) {
	as.loadFuncs = append(as.loadFuncs, cb)
}
func (as *AuthService) OnAfterLoad(cb AuthFunc) {
	as.afterLoadFuncs = append(as.afterLoadFuncs, cb)
}
func (as *AuthService) OnBeforeAuth(cb AuthFunc) {
	as.beforeAuthFuncs = append(as.beforeAuthFuncs, cb)
}
func (as *AuthService) OnAuth(cb func(*AuthContext) (bool, error)) {
	as.authFuncs = append(as.authFuncs, cb)
}
func (as *AuthService) OnAfterAuth(cb AuthFunc) {
	as.afterAuthFuncs = append(as.afterAuthFuncs, cb)
}
func (as *AuthService) OnError(cb func(ctx *AuthContext, err error) error) {
	as.errFuncs = append(as.errFuncs, cb)
}
func (as *AuthService) Auth(ctx *AuthContext) error {
	ctx.Step = BeforeLoad

	for _, a := range as.beforeLoadFuncs {
		if err := a(ctx); err != nil {
			return as.callError(ctx, err)
		}
	}

	ctx.Step = Loading

	// isLoaded := false
	for _, a := range as.loadFuncs {
		id, authentication, err := a(ctx)
		if err != nil {
			return as.callError(ctx, err)
		}
		if authentication != nil {
			ctx.Request.UserID = id
			ctx.Authentication = authentication
			// isLoaded = true
			break
		}
	}

	// 删除这个检测，因为 ldap 或 cas 第一次登录时用户不在系统中
	// if !isLoaded {
	// 	return as.callError(ctx, ErrUserNotFound)
	// }

	ctx.Step = AfterLoaded
	for _, a := range as.afterLoadFuncs {
		if err := a(ctx); err != nil {
			return as.callError(ctx, err)
		}
	}
	ctx.Step = BeforeAuth
	for _, a := range as.beforeAuthFuncs {
		if err := a(ctx); err != nil {
			return as.callError(ctx, err)
		}
	}
	ctx.Step = Authing
	for _, a := range as.authFuncs {
		ok, err := a(ctx)
		if err != nil {
			if err == ErrPasswordNotMatch {
				ctx.Response.IsOK = false
				break
			}
			return as.callError(ctx, err)
		}
		if ok {
			ctx.Response.IsOK = true
			break
		}
	}
	ctx.Step = AfterAuthed
	for _, a := range as.afterAuthFuncs {
		if err := a(ctx); err != nil {
			return as.callError(ctx, err)
		}
	}

	return nil
}

func (as *AuthService) callError(ctx *AuthContext, err error) error {
	for _, a := range as.errFuncs {
		if e := a(ctx, err); e != nil {
			err = e
		}
	}
	return err
}

func NewAuthService(um UserManager, opts ...AuthOption) (*AuthService, error) {
	auth := &AuthService{}
	auth.OnLoad(func(a *AuthContext) (interface{}, interface{}, error) {
		return um.Read(a)
	})
	if err := DefaultUserCheck().apply(auth); err != nil {
		return nil, err
	}
	for _, opt := range opts {
		if err := opt.apply(auth); err != nil {
			return nil, err
		}
	}
	return auth, nil
}

type AuthOption interface {
	apply(auth *AuthService) error
}

type AuthOptionFunc func(auth *AuthService) error

func (cb AuthOptionFunc) apply(auth *AuthService) error {
	return cb(auth)
}
