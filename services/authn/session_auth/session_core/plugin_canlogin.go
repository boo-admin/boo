package session_core

import "fmt"

type CanLoginable interface {
	Loginable() bool
}

func CanLogin() AuthOption {
	return AuthOptionFunc(func(auth *AuthService) error {
		auth.OnAfterLoad(AuthFunc(func(ctx *AuthContext) error {
			if ctx.Authentication == nil {
				return nil
			}
			u, ok := ctx.Authentication.(CanLoginable)
			if !ok {
				msg := fmt.Sprintf("user is unsupported for the loginable - %T", ctx.Authentication)
				ctx.Logger.Warn(msg)
				return nil // fmt.Errorf("user is unsupported for the error lock - %T", ctx.Authentication)
			}
			if !u.Loginable() {
				return ErrUserCanLogin
			}
			return nil
		}))
		return nil
	})
}
