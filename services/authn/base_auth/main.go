package base_auth

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/boo-admin/boo/services/authn"
)

const HeaderAuthorization = "Authorization"

const (
	basic        = "basic"
	defaultRealm = "Restricted"
)

func Verify(validator func(ctx context.Context, req *http.Request, username string, password string) (context.Context, error)) authn.AuthValidateFunc {
	return func(ctx context.Context, req *http.Request) (context.Context, error) {
		auth := req.Header.Get(HeaderAuthorization)
		l := len(basic)

		if len(auth) > l+1 && strings.EqualFold(auth[:l], basic) {
			// Invalid base64 shouldn't be treated as error
			// instead should be treated as invalid client input
			b, err := base64.StdEncoding.DecodeString(auth[l+1:])
			if err != nil {
				return nil, authn.ErrInvalidCredentials
			}

			cred := string(b)
			for i := 0; i < len(cred); i++ {
				if cred[i] == ':' {
					// Verify credentials
					return validator(ctx, req, cred[:i], cred[i+1:])
				}
			}
		}

		// realm := defaultRealm
		// if config.Realm != defaultRealm {
		// 	realm = strconv.Quote(config.Realm)
		// }

		// // Need to return `401` for browsers to pop-up login box.
		// c.Response().Header().Set(echo.HeaderWWWAuthenticate, basic+" realm="+realm)
		// return echo.ErrUnauthorized

		return nil, authn.ErrTokenNotFound
	}
}
