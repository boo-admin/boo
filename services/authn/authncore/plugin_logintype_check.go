package authncore

import (
	"github.com/boo-admin/boo/errors"
)

func LoginTypeCheck() AuthOption {
	return AuthOptionFunc(func(auth *AuthService) error {
		auth.OnBeforeAuth(AuthFunc(func(ctx *AuthContext) error {
			if ctx.Authentication == nil {
				return nil
			}
			u, ok := ctx.Authentication.(HasSource)
			if ok {
				if ctx.Request.LoginType != TokenJWT {
					if u.Source() == "api" {
						return errors.New("api user cannot login ")
					}
				}
			}

			return nil
		}))
		return nil
	})
}
