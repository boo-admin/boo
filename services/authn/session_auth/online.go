//go:generate gogenv2 server -ext=.server-gen.go online.go
//go:generate gogenv2 client -ext=.client-gen.go online.go

package session_auth

import (
	"context"

	"github.com/boo-admin/boo/booclient"
	"github.com/boo-admin/boo/services/authn/session_auth/session_core"
	"github.com/boo-admin/boo/errors"
)

type ErrOnline struct {
	OnlineList []booclient.OnlineInfo
}

func (err *ErrOnline) Error() string {
	if len(err.OnlineList) == 1 {
		return "用户已在 " + err.OnlineList[0].Address +
			" 上登录，最后一次活动时间为 " +
			err.OnlineList[0].UpdatedAt.Format("2006-01-02 15:04:05Z07:00")
	}
	return "用户已在其他机器上登录"
}

func IsOnlinedError(err error) ([]booclient.OnlineInfo, bool) {
	for err != nil {
		oe, ok := err.(*ErrOnline)
		if ok {
			return oe.OnlineList, true
		}
		err = errors.Unwrap(err)
	}
	return nil, false
}

type OnlineAdder interface {
	// @Summary 创建一个在线会话(内部使用，客户将无法使用)
	// @Description 访问本接口时需要额外的 password
	// @Tags     Inner
	// @Param username body string   true        "用户的名称"
	// @Param address  body string   true        "登录地址"
	// @Param api_key  body string   true        "访问本接口时的 password"
	// @Accept  json
	// @Produce  json
	// @Router /online-users [post]
	// @Success 200 {string}  string  "返回新建会话的 ID"
	Login(ctx context.Context, username, address, apiKey string) (string, error)

	// @Summary 更新一下在线会话的存活时间(内部使用，客户将无法使用)
	// @Description 访问本接口时需要额外的 password
	// @Tags     Inner
	// @Param    uuid     path string   true        "用户的 ID"
	// @Param    api_key  body string   true        "访问本接口时的 password"
	// @Accept   json
	// @Produce  json
	// @Router /online-users/by_uuid/{uuid}/keeplive [patch]
	// @Success 200 {string}  string  "返回一个无意义的 'ok'"
	UpdateNow(ctx context.Context, uuid, apiKey string) error
}

type Onlines interface {
	booclient.OnlineQueryer

	session_core.OnlineChecker

	OnlineAdder
}

type OnlineStore interface {
	Load(context.Context) error
	Store(context.Context) error
}

func OnlineCount(ctx context.Context, onlines Onlines, username string, address string) (int, error) {
	list, err := onlines.List(ctx)
	if err != nil {
		return 0, err
	}

	filter := func(si *booclient.OnlineInfo) bool {
		return true
	}
	if username != "" {
		if address != "" {
			filter = func(si *booclient.OnlineInfo) bool {
				return si.Username == username && si.Address == address
			}
		} else {
			filter = func(si *booclient.OnlineInfo) bool {
				return si.Username == username
			}
		}
	} else if address != "" {
		filter = func(si *booclient.OnlineInfo) bool {
			return si.Address == address
		}
	}

	var count = 0
	for _, s := range list {
		if filter(&s) {
			count++
		}
	}
	return count, nil
}

type Locker = session_core.Locker
type IsLocker = session_core.IsLocker

type NoneLocker struct{}

func (NoneLocker) Lock(ctx *session_core.AuthContext) error {
	return nil
}

func (NoneLocker) IsLocked(ctx *session_core.AuthContext) error {
	return nil
}
