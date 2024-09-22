package authncore

import (
	"context"
	"errors"
	"strings"
)

type OnlineChecker interface {
	IsOnlineExists(ctx context.Context, userid interface{}, username, loginAddress string) error
}

// IsOnlineExists(userid interface{}, username, loginAddress string) error

//   // 判断用户是不是已经在其它主机上登录
//   onlineList, err := online.Query(ctx.Request.Username)
//   if err != nil {
//     return err
//   }

//   if len(onlineList) != 0 {
//     found := false
//     for _, ol := range onlineList {
//       if ol.Address == ctx.Request.Address {
//         found = true
//         break
//       }
//     }
//     if !found {
//       return &ErrOnline{onlineList: onlineList}
//     }
//   }
//   return nil

func OnlineCheck(online OnlineChecker, loginConflict string) AuthOption {
	return AuthOptionFunc(func(auth *AuthService) error {
		if online == nil {
			return errors.New("online is missing")
		}

		loginConflict := strings.ToLower(loginConflict)

		if loginConflict != "force" &&
			loginConflict != "" &&
			loginConflict != "auto" &&
			loginConflict != "disableforce" {
			return errors.New("loginConflict is invalid - " + loginConflict)
		}

		auth.OnBeforeLoad(AuthFunc(func(ctx *AuthContext) error {
			var isForce = ctx.Request.IsForce()
			switch loginConflict {
			case "force":
				isForce = true
			case "", "auto":
				// 请从界面发送的请求中的参数决定
			case "disableforce":
				isForce = false
			}

			if isForce || ctx.Request.Address == "127.0.0.1" {
				return nil
			}

			return online.IsOnlineExists(ctx.Ctx, ctx.Request.UserID, ctx.Request.Username, ctx.Request.Address)
		}))

		return nil
	})
}
