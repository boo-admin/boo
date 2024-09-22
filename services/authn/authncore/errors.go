package authncore

import (
	"net/http"

	"github.com/boo-admin/boo/services/authn"
)

func newHTTPError(code int, msg string) error {
	return authn.NewHTTPError(code, msg)
}

var (
	// ErrUserDisabled 用户被禁用
	ErrUserDisabled = newHTTPError(http.StatusUnauthorized, "user name is disabled")

	// ErrUsernameEmpty 用户名为空
	ErrUsernameEmpty = newHTTPError(http.StatusUnauthorized, "user name is empty")

	// ErrPasswordEmpty 密码为空
	ErrPasswordEmpty = newHTTPError(http.StatusUnauthorized, "user password is empty")

	// ErrUserNotFound 用户未找到
	ErrUserNotFound = newHTTPError(http.StatusUnauthorized, "user isn't found")

	// ErrUserErrorCountExceedLimit 用户未找到
	ErrUserErrorCountExceedLimit = newHTTPError(http.StatusUnauthorized, "user isn't error count exceed limit")

	// ErrPasswordNotMatch 密码不正确
	ErrPasswordNotMatch = newHTTPError(http.StatusUnauthorized, "password isn't match")

	// ErrMutiUsers 找到多个用户
	ErrMutiUsers = newHTTPError(http.StatusUnauthorized, "muti users is found")

	// ErrUserLocked 用户已被锁定
	ErrUserLocked = newHTTPError(http.StatusUnauthorized, "user is locked")

	// ErrUserIPBlocked 用户不在指定的 IP 范围登录
	ErrUserIPBlocked = newHTTPError(http.StatusUnauthorized, "user address is blocked")

	// ErrServiceTicketNotFound Service ticket 没有找到
	ErrServiceTicketNotFound = newHTTPError(http.StatusUnauthorized, "service ticket isn't found")

	// ErrServiceTicketExpired Service ticket 已过期
	ErrServiceTicketExpired = newHTTPError(http.StatusUnauthorized, "service ticket isn't expired")

	// ErrUnauthorizedService Service 是未授权的
	ErrUnauthorizedService = newHTTPError(http.StatusUnauthorized, "service is unauthorized")

	// ErrUserAlreadyOnline 用户已登录
	ErrUserAlreadyOnline = newHTTPError(http.StatusUnauthorized, "user is already online")

	// ErrPermissionDenied 没有权限
	ErrPermissionDenied = newHTTPError(http.StatusUnauthorized, "permission is denied")

	// ErrUserCanLogin
	ErrUserCanLogin = newHTTPError(http.StatusUnauthorized, "user don't logined")

	// ErrCaptchaKey
	ErrCaptchaKey = newHTTPError(http.StatusUnauthorized, "captcha is error")

	// ErrCaptchaMissing
	ErrCaptchaMissing = newHTTPError(http.StatusUnauthorized, "captcha is missing")
)

type errAddress string

func (s errAddress) Error() string {
	return "client address is invalid - '" + string(s) + "'"
}

func IsAddressInvalid(e error) bool {
	_, ok := e.(errAddress)
	return ok
}

// type ErrOnline struct {
// 	onlineList []SessionInfo
// }

// func (err *ErrOnline) Error() string {
// 	if len(err.onlineList) == 1 {
// 		return "用户已在 " + err.onlineList[0].Address +
// 			" 上登录，最后一次活动时间为 " +
// 			err.onlineList[0].UpdatedAt.Format("2006-01-02 15:04:05Z07:00")

// 	}
// 	return "用户已在其他机器上登录"
// }

// func IsOnlinedError(err error) ([]SessionInfo, bool) {
// 	for err != nil {
// 		oe, ok := err.(*ErrOnline)
// 		if ok {
// 			return oe.onlineList, true
// 		}
// 		err = errors.Unwrap(err)
// 	}
// 	return nil, false
// }

type ErrExternalServer struct {
	Msg string
	Err error
}

func (e *ErrExternalServer) Error() string {
	return e.Msg
}

func (e *ErrExternalServer) Unwrap() error {
	return e.Err
}

func IsErrExternalServer(e error) bool {
	_, ok := e.(*ErrExternalServer)
	return ok
}
