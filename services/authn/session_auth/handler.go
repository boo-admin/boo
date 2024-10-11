package session_auth

import (
	"context"
	"crypto/sha1"
	"hash"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/boo-admin/boo/client"
	"github.com/boo-admin/boo/errors"
	"github.com/boo-admin/boo/services/authn"
)

const (
	CfgUserSessionPath       = "users.cookie.path"
	CfgUserSessionName       = "users.cookie.name"
	CfgUserSessionDomain     = "users.cookie.domain"
	CfgUserSessionHashFunc   = "users.cookie.hash_method"
	CfgUserSessionHashSecret = "users.cookie.hash_secret"
	CfgUserSessionMaxAge     = "users.cookie.maxage"
	CfgUserSessionSecure     = "users.cookie.secure"
	CfgUserSessionHttpOnly   = "users.cookie.httponly"
	CfgUserSessionSameSite   = "users.cookie.samesite"
)

type Option struct {
	SessionPath       string
	SessionName       string
	SessionHashFunc   func() hash.Hash
	SessionHashSecret []byte

	SessionDomain   string
	SessionHttpOnly bool
	SessionSecure   bool
	SessionMaxAge   int
	SessionSameSite http.SameSite
}

func CreateCookie(opt *Option, values url.Values) *http.Cookie {
	var ts = "session"
	values.Set(SESSION_EXPIRE_KEY, ts)
	values.Set("_TS", ts)
	// if sessionTimeout > 0 {
	// 	ts = strconv.FormatInt(time.Now().Add(time.Duration(sessionTimeout)*time.Minute).Unix(), 10)
	// }
	values.Set("issued_at", time.Now().Format(time.RFC3339))

	return &http.Cookie{
		Name:     opt.SessionName,
		Value:    Encode(values, opt.SessionHashFunc, opt.SessionHashSecret),
		Domain:   opt.SessionDomain,
		Path:     opt.SessionPath,
		HttpOnly: opt.SessionHttpOnly,
		Secure:   opt.SessionSecure,
		MaxAge:   opt.SessionMaxAge,
		SameSite: opt.SessionSameSite,

		// Expires: ts.UTC(), // 不指定过期时间，那么关闭浏览器后 cookie 会删除
	}
}

func SessionVerify(opt *Option, handle func(ctx context.Context, req *http.Request, values url.Values) (context.Context, error)) authn.AuthValidateFunc {
	sessionKey := opt.SessionName
	sessionPath := opt.SessionPath
	secretKey := opt.SessionHashSecret

	if sessionPath == "" {
		sessionPath = "/" // 必须指定 Path, 否则会被自动赋成当前请求的 url 中的 path
	} else if !strings.HasPrefix(sessionPath, "/") {
		sessionPath = "/" + sessionPath
	}

	// currentURL := opt.CurrentURL
	// if currentURL == nil {
	// 	currentURL = func(req *http.Request) url.URL {
	// 		return *req.URL
	// 	}
	// }

	h := opt.SessionHashFunc
	if h == nil {
		h = sha1.New
	}

	return func(ctx context.Context, req *http.Request) (context.Context, error) {
		values, err := GetValues(req, sessionKey, h, secretKey)
		if err != nil {
			if err == ErrCookieNotFound || err == ErrCookieEmpty {
				return nil, authn.ErrTokenNotFound
			}
			return nil, err
		}

		return handle(ctx, req, values)
	}
}

func New(env *client.Environment, sessionUser func(ctx context.Context, req *http.Request, values url.Values) (context.Context, error)) (authn.AuthValidateFunc, error) {
	var sessionOpt Option
	sessionOpt.SessionPath = env.Config.StringWithDefault(CfgUserSessionPath, env.AppPathWithoutSlash)
	sessionOpt.SessionName = env.Config.StringWithDefault(CfgUserSessionName, "boo_session")
	sessionOpt.SessionDomain = env.Config.StringWithDefault(CfgUserSessionDomain, "")

	var sessionHash = env.Config.StringWithDefault(CfgUserSessionHashFunc, "sha1")
	if h, err := GetHash(sessionHash); err != nil {
		return nil, errors.Wrap(err, "参数 '"+CfgUserSessionHashFunc+"' 的值 '"+sessionHash+"' 是未知的 hash 算法")
	} else {
		sessionOpt.SessionHashFunc = h
	}
	sessionOpt.SessionHashSecret = []byte(env.Config.StringWithDefault(CfgUserSessionHashSecret, ""))
	if len(sessionOpt.SessionHashSecret) == 0 {
		return nil, errors.New("读 " + CfgUserSessionHashSecret + " 失败，没有在配置中找到它")
	}

	sessionOpt.SessionMaxAge = env.Config.IntWithDefault(CfgUserSessionMaxAge, 0)
	sessionOpt.SessionSecure = env.Config.BoolWithDefault(CfgUserSessionSecure, false)
	sessionOpt.SessionHttpOnly = env.Config.BoolWithDefault(CfgUserSessionHttpOnly, false)

	var sessionSameSite = env.Config.StringWithDefault(CfgUserSessionSameSite, "")
	if sameSite, err := ParseHttpSameSite(sessionSameSite); err != nil {
		return nil, errors.Wrap(err, "参数 '"+CfgUserSessionSameSite+"' 的值 '"+sessionSameSite+"' 是未知的 hash 算法")
	} else {
		sessionOpt.SessionSameSite = sameSite
	}

	return SessionVerify(&sessionOpt, sessionUser), nil
}

func ParseHttpSameSite(s string) (http.SameSite, error) {
	switch strings.ToLower(s) {
	case "", "default":
		return http.SameSiteDefaultMode, nil
	case "lax":
		return http.SameSiteLaxMode, nil
	case "strict":
		return http.SameSiteStrictMode, nil
	case "none":
		return http.SameSiteNoneMode, nil
	}
	return http.SameSiteNoneMode, errors.New("samesite '" + s + "' is invalid")
}
