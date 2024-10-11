package session_core

import (
	"errors"
	"fmt"
	"net"

	"github.com/mei-rune/iprange"
)

var localAddressList, _ = net.LookupHost("localhost")

type HasWhitelist interface {
	IngressIPList() ([]iprange.Checker, error)
}

func Whitelist() AuthOption {
	return AuthOptionFunc(func(auth *AuthService) error {
		auth.OnBeforeAuth(AuthFunc(func(ctx *AuthContext) error {
			if ctx.Request.Address == "" || ctx.Request.Address == "127.0.0.1" {
				return nil
			}

			for _, addr := range localAddressList {
				if ctx.Request.Address == addr {
					return nil
				}
			}

			if ctx.Authentication == nil {
				return nil
			}

			u, ok := ctx.Authentication.(HasWhitelist)
			if !ok {
				return errors.New("user is unsupported for the ip white list - " + fmt.Sprintf("%T", ctx.Authentication))
			}

			ingressIPList, err := u.IngressIPList()
			if err != nil {
				return err
			}

			if len(ingressIPList) == 0 {
				return nil
			}

			currentAddr := net.ParseIP(ctx.Request.Address)
			if currentAddr == nil {
				return errAddress(ctx.Request.Address)
			}

			blocked := true
			for _, checker := range ingressIPList {
				if checker.Contains(currentAddr) {
					blocked = false
					break
				}
			}

			if blocked {
				return ErrUserIPBlocked
			}

			return nil
		}))
		return nil
	})
}
