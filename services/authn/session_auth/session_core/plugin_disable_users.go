package session_core

import "strings"

func DisableUsers(usernames []string) AuthOption {
	return AuthOptionFunc(func(auth *AuthService) error {
		auth.OnBeforeLoad(func(ctx *AuthContext) error {

			for _, name := range usernames {
				if strings.ToLower(name) == strings.ToLower(ctx.Request.Username) {
					return ErrUserDisabled
				}
			}
			return nil
		})
		return nil
	})
}
